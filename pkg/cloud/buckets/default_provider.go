package buckets

import (
	"bufio"
	"context"
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/gke"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	"time"
)

// GKEBucketProvider the bucket provider for GKE
type LegacyBucketProvider struct {
	gcloud gke.GClouder
	bucket *blob.Bucket
	classifier string
}

func (LegacyBucketProvider) CreateNewBucketForCluster(clusterName string, bucketKind string) (string, error) {
	return "", fmt.Errorf("CreateNewBucketForCluster not implemented for LegacyBucketProvider")
}
func (LegacyBucketProvider) EnsureBucketIsCreated(bucketURL string) error {
	return fmt.Errorf("EnsureBucketIsCreated not implemented for LegacyBucketProvider")

}
func (p LegacyBucketProvider) UploadFileToBucket(bytes []byte, outputName string, bucketURL string) (string, error) {
	opts := &blob.WriterOptions{
		ContentType: util.ContentTypeForFileName(outputName),
		Metadata: map[string]string{
			"classification": p.classifier,
		},
	}
	u := ""
	ctx := p.createContext()
	err := p.bucket.WriteAll(ctx, outputName, bytes, opts)
	if err != nil {
		return u, errors.Wrapf(err, "failed to write to bucket %s", outputName)
	}

	u = util.UrlJoin(bucketURL, outputName)
	return "", fmt.Errorf("UploadFileToBucket not implemented for LegacyBucketProvider")
}

func (LegacyBucketProvider) createContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), time.Second * 20)
	return ctx
}

func (LegacyBucketProvider) DownloadFileFromBucket(bucketURL string) (*bufio.Scanner, error) {
	return nil, fmt.Errorf("DownloadFileFromBucket not implemented for LegacyBucketProvider")
}

func (p *LegacyBucketProvider) Initialize(bucketURL string, classifier string) error {
	log.Logger().Warnf("Calling LegacyBucketProvider with bucketURL %s, classifier: %s", bucketURL, classifier)
	ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
	if bucketURL == "" {
		return fmt.Errorf("no BucketURL is configured for the storage location in the TeamSettings")
	}
	bucket, err := blob.Open(ctx, bucketURL)
	if err != nil {
		return errors.Wrapf(err, "failed to open bucket %s", bucketURL)
	}
	p.bucket = bucket
	p.classifier = classifier
	return nil
}

// NewGKEBucketProvider create a new provider for GKE
func NewLegacyBucketProvider() Provider {
	return &LegacyBucketProvider{
		gcloud: &gke.GCloud{},
	}
}
