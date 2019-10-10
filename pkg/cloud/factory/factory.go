package factory

import (
	"github.com/jenkins-x/jx/pkg/cloud"
	amazonStorage "github.com/jenkins-x/jx/pkg/cloud/amazon/storage"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/gke/storage"
	"github.com/jenkins-x/jx/pkg/cmd/clients"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/kube"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"
)

// NewBucketProvider creates a new bucket provider for a given Kubernetes provider
func NewBucketProvider(requirements *config.RequirementsConfig) buckets.Provider {
	if requirements == nil {
		log.Logger().Warn("Creating a legacy bucket provider")
		return buckets.NewLegacyBucketProvider()
	}
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

func NewBucketProviderFromTeamSettingsConfiguration() (buckets.Provider, error) {
	factory := clients.NewFactory()
	jxClient, ns, err := factory.CreateJXClient()
	if err != nil {
		return nil, err
	}
	log.Logger().Warn("Getting the dev environment")
	teamSettings, err := kube.GetDevEnvTeamSettings(jxClient, ns)
	if err != nil {
		return nil, errors.Wrap(err, "error obtaining the dev environment teamSettings to select the correct bucket provider")
	}
	log.Logger().Warnf("Environment %+v", teamSettings)
	if teamSettings != nil {
		log.Logger().Warnf("TEAMSETTINGS %+v", teamSettings)
		requirements, err := config.GetRequirementsConfigFromTeamSettings(teamSettings)
		if err != nil {
			return nil, errors.Wrap(err, "could not obtain the requirements file to decide the bucket provider")
		}
		if requirements != nil {
			log.Logger().Warnf("Requirements form Team Settings %+v", *requirements)
		}
		return NewBucketProvider(requirements), nil
	}
	return nil, nil
}
