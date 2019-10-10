package collector

import (
	"github.com/jenkins-x/jx/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/cloud/factory"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/pkg/errors"

	// lets import all the blob providers we need
	_ "gocloud.dev/blob/azureblob"
	_ "gocloud.dev/blob/fileblob"
	_ "gocloud.dev/blob/gcsblob"
	_ "gocloud.dev/blob/s3blob"
)

// NewCollector creates a new collector from the storage configuration
func NewCollector(storageLocation v1.StorageLocation, gitter gits.Gitter) (Collector, error) {
	classifier := storageLocation.Classifier
	if classifier == "" {
		classifier = "default"
	}
	gitURL := storageLocation.GitURL
	if gitURL != "" {
		return NewGitCollector(gitter, gitURL, storageLocation.GetGitBranch())
	}
	log.Logger().Warn("Attempting to get a bucket provider")
	bucketProvider, err := defineBucketProvider(storageLocation)
	if err != nil {
		return nil, errors.Wrap(err, "error obtaining a bucket provider")
	}
	return NewBucketCollector(storageLocation.BucketURL, classifier, bucketProvider)
}

func defineBucketProvider(storageLocation v1.StorageLocation) (buckets.Provider, error) {
	bucketProvider, err := factory.NewBucketProviderFromTeamSettingsConfiguration()
	if err != nil {
		log.Logger().Errorf("there was a problem obtaining the bucket provider from cluster configuration %s", err.Error())
	}
	log.Logger().Warnf("Bucket provider: %+v", bucketProvider)
	log.Logger().Debugf("Bucket provider obtained %+v", bucketProvider)
	// LegacyBucketProvider is just here to keep backwards compatibility with non boot clusters, that's why we need to pass
	// some configuration in a different way, it shouldn't be the norm for providers
	switch t := bucketProvider.(type) {
	case *buckets.LegacyBucketProvider:
		err := t.Initialize(storageLocation.BucketURL, storageLocation.Classifier)
		if err != nil {
			return nil, err
		}
	case buckets.LegacyBucketProvider:
		log.Logger().Warn("LegacyBucketProvider not pointer")
		if err != nil {
			return nil, err
		}
	default:
		log.Logger().Debugf("not performing any additional initialization as the chosen provider is not Legacy")
	}
	return bucketProvider, nil
}
