package factory

import (
	"github.com/ghodss/yaml"
	"github.com/jenkins-x/jx/pkg/cloud"
	amazonStorage "github.com/jenkins-x/jx/pkg/cloud/amazon/storage"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/gke/storage"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/pkg/errors"
)

// NewBucketProvider creates a new bucket provider for a given Kubernetes provider
func NewBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	switch requirements.Cluster.Provider {
	case cloud.GKE:
		return storage.NewGKEBucketProvider(requirements)
	case cloud.EKS:
		fallthrough
	case cloud.AWS:
		return amazonStorage.NewAmazonBucketProvider(requirements)
	default:
		return nil
	}
}

func NewBucketProviderFromClusterConfiguration() (buckets.Provider, error) {
	factory := clients.NewFactory()
	kubeClient, ns, err := factory.CreateKubeClient()
	if err != nil {
		return nil, err
	}

	data, err := kube.GetConfigMapData(kubeClient, "jx-requirements-config", ns)
	if err != nil {
		return nil, err
	}
	requirementsFileData := data["requirementsFile"]
	requirements := &config.RequirementsConfig{}
	err = yaml.Unmarshal([]byte(requirementsFileData), requirements)
	if err != nil {
		return nil, errors.Wrapf(err, "there was a problem unmarshaling the requirements file from ConfigMap")
	}
	return NewBucketProvider(requirements), nil
}