package collector

import (
	"context"
	"github.com/jenkins-x/jx/pkg/cloud/buckets"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"gocloud.dev/blob"
	"io/ioutil"
	"path/filepath"
	"time"
)

// BucketCollector stores the state for the git collector
type BucketCollector struct {
	Timeout time.Duration

	bucketURL  string
	bucket     *blob.Bucket
	classifier string
	provider   buckets.Provider
}

// NewBucketCollector creates a new git based collector
func NewBucketCollector(bucketURL string, bucket *blob.Bucket, classifier string, provider buckets.Provider) (Collector, error) {
	return &BucketCollector{
		Timeout:    time.Second * 20,
		bucketURL:  bucketURL,
		bucket:     bucket,
		classifier: classifier,
		provider:   provider,
	}, nil
}

// CollectFiles collects files and returns the URLs
func (c *BucketCollector) CollectFiles(patterns []string, outputPath string, basedir string) ([]string, error) {
	urls := []string{}
	bucket := c.bucket
	ctx := c.createContext()
	for _, p := range patterns {
		fn := func(name string) error {
			var err error
			toName := name
			if basedir != "" {
				toName, err = filepath.Rel(basedir, name)
				if err != nil {
					return errors.Wrapf(err, "failed to remove basedir %s from %s", basedir, name)
				}
			}
			if outputPath != "" {
				toName = filepath.Join(outputPath, toName)
			}
			data, err := ioutil.ReadFile(name)
			if err != nil {
				return errors.Wrapf(err, "failed to read file %s", name)
			}
			var url string
			if c.provider == nil {
				url, err = c.performLegacyUpload(bucket, name, ctx, toName, data)
				if err != nil {
					return err
				}
			} else {
				url, err = c.provider.UploadFileToBucket(data, toName, c.bucketURL)
				if err != nil {
					return err
				}
			}
			urls = append(urls, url)
			return nil
		}

		err := util.GlobAllFiles("", p, fn)
		if err != nil {
			return urls, err
		}
	}
	return urls, nil
}

// CollectData collects the data storing it at the given output path and returning the URL
// to access it
func (c *BucketCollector) CollectData(data []byte, outputName string) (string, error) {
	if c.provider == nil {
		opts := &blob.WriterOptions{
			ContentType: util.ContentTypeForFileName(outputName),
			Metadata: map[string]string{
				"classification": c.classifier,
			},
		}
		u := ""
		ctx := c.createContext()
		err := c.bucket.WriteAll(ctx, outputName, data, opts)
		if err != nil {
			return u, errors.Wrapf(err, "failed to write to bucket %s", outputName)
		}

		u = util.UrlJoin(c.bucketURL, outputName)
		return u, nil
	}
	log.Logger().Warn("Uploading using provider")
	url, err := c.provider.UploadFileToBucket(data, outputName, c.bucketURL)
	if err != nil {
		return "", err
	}
	return url, nil
}

func (c *BucketCollector) performLegacyUpload(bucket *blob.Bucket, name string, ctx context.Context, toName string, data []byte) (string, error) {
	opts := &blob.WriterOptions{
		ContentType: util.ContentTypeForFileName(name),
		Metadata: map[string]string{
			"classification": c.classifier,
		},
	}
	err := bucket.WriteAll(ctx, toName, data, opts)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write to bucket %s", toName)
	}

	u := util.UrlJoin(c.bucketURL, toName)
	return u, nil
}

func (c *BucketCollector) createContext() context.Context {
	ctx, _ := context.WithTimeout(context.Background(), c.Timeout)
	return ctx
}
