package controller

import (
	"fmt"
	"time"

	v13 "k8s.io/api/rbac/v1"

	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/jenkins-x/jx/v2/pkg/client/clientset/versioned"

	"github.com/jenkins-x/jx/v2/pkg/kube/naming"

	"github.com/jenkins-x/jx/v2/pkg/cmd/deletecmd"

	"github.com/jenkins-x/jx/v2/pkg/log"

	v1 "github.com/jenkins-x/jx/v2/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/opts"
	"github.com/jenkins-x/jx/v2/pkg/kube"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/tools/cache"
)

const DevEnv = "dev"

type ControllerNamespaceOptions struct {
	*opts.CommonOptions
}

func NewCmdControllerNamespace(commonOpts *opts.CommonOptions) *cobra.Command {
	options := ControllerNamespaceOptions{
		CommonOptions: commonOpts,
	}
	cmd := &cobra.Command{
		Use:     "namespace",
		Short:   "Runs the service to create new namespaces.",
		Long:    serveBuildNumbersLong,
		Example: serveBuildNumbersExample,
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
	}
	return cmd
}

// Run will execute this command, starting the HTTP build number generation service with the specified options.
func (o *ControllerNamespaceOptions) Run() error {
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		return err
	}
	env := &v1.Environment{}
	listWatcher := cache.NewListWatchFromClient(jxClient.JenkinsV1().RESTClient(), "environments", ns, fields.Everything())
	kube.SortListWatchByName(listWatcher)
	_, controller := cache.NewInformer(listWatcher, env, time.Minute*10,
		cache.ResourceEventHandlerFuncs{
			AddFunc:    o.onAddEnvironment,
			UpdateFunc: func(oldObj, newObj interface{}) {},
			DeleteFunc: o.onDeleteEnvironment,
		})
	stop := make(chan struct{})
	go controller.Run(stop)

	// Wait forever
	select {}

	return nil
}

func (o *ControllerNamespaceOptions) onAddEnvironment(obj interface{}) {
	var addedEnv *v1.Environment
	switch obj.(type) {
	case *v1.Environment:
		addedEnv = obj.(*v1.Environment)
	default:
		log.Logger().Warnf("The received object is not of type %T", addedEnv)
		return
	}
	kubeClient, ns, err := o.KubeClientAndDevNamespace()
	if err != nil {
		log.Logger().Error(err)
		return
	}
	jxClient, _, err := o.JXClient()
	if err != nil {
		log.Logger().Error(err)
		return
	}
	err = kube.EnsureEnvironmentNamespaceSetup(kubeClient, jxClient, addedEnv, ns)
	if err != nil {
		log.Logger().Error(err)
		return
	}

	err = createEnvironmentRoleBinding(jxClient, ns, addedEnv)
	if err != nil {
		log.Logger().Error(err)
		return
	}
}

func (o *ControllerNamespaceOptions) onDeleteEnvironment(obj interface{}) {
	env := obj.(*v1.Environment)
	deleteOpts := deletecmd.DeleteNamespaceOptions{
		CommonOptions: o.CommonOptions,
		Confirm:       true,
	}
	deleteOpts.Args = []string{env.Spec.Namespace}
	deleteOpts.BatchMode = true
	err := deleteOpts.Run()
	if err != nil {
		log.Logger().Error(err)
		return
	}
	jxClient, ns, err := o.JXClientAndDevNamespace()
	if err != nil {
		log.Logger().Error(err)
		return
	}
	err = jxClient.JenkinsV1().EnvironmentRoleBindings(ns).Delete(naming.ToValidNameTruncated(fmt.Sprintf("%s-%s",
		env.Name, env.Spec.Namespace), 20), &v12.DeleteOptions{})
	if err != nil {
		log.Logger().Error(err)
	}
}

func (o *ControllerNamespaceOptions) onUpdateEnvironment(oldObj, newObj interface{}) {
	// When an environment is updated, we'll just make sure the environmentrolebinding is created
	jxClient, ns, err := o.JXClient()
	if err != nil {
		log.Logger().Error(err)
		return
	}
	switch newObj.(type) {
	case *v1.Environment:
		{
			env := newObj.(*v1.Environment)
			log.Logger().Infof("Updating environment %s", env.Name)
			err = createEnvironmentRoleBinding(jxClient, ns, env)
			if err != nil {
				log.Logger().Error(err)
				return
			}
		}
	default:
		log.Logger().Warnf("The received object is not of type %T", v1.Environment{})
		return
	}

}

func createEnvironmentRoleBinding(jxClient versioned.Interface, ns string, environment *v1.Environment) error {
	if environment.Name == DevEnv {
		log.Logger().Warnf("Not creating an EnvironmentRoleBinding for Environment %s", DevEnv)
		return nil
	}
	log.Logger().Debugf("Creating EnvironmentRoleBinding for env %s", environment.Name)
	_, err := jxClient.JenkinsV1().EnvironmentRoleBindings(ns).Create(&v1.EnvironmentRoleBinding{
		ObjectMeta: v12.ObjectMeta{
			Name: naming.ToValidNameTruncated(fmt.Sprintf("%s-%s", environment.Name, environment.Spec.Namespace), 20),
		},
		Spec: v1.EnvironmentRoleBindingSpec{
			Subjects: []v13.Subject{
				{
					Kind:      "ServiceAccount",
					Name:      "tekton-bot",
					Namespace: ns,
				},
			},
			RoleRef: v13.RoleRef{
				APIGroup: "rbac.authorization.k8s.io",
				Kind:     "Role",
				Name:     "tekton-bot",
			},
			Environments: []v1.EnvironmentFilter{
				{
					Includes: []string{environment.Name},
				},
			},
		},
	})
	return err
}
