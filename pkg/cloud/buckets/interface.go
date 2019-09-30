package buckets

import "bufio"

// Provider represents a bucket provider
type Provider interface {
	// CreateNewBucketForCluster creates a new dynamically named bucket
	CreateNewBucketForCluster(clusterName string, bucketKind string) (string, error)
	EnsureBucketIsCreated(bucketURL string) error
	UploadFileToBucket(bytes []byte, outputName string, bucketURL string) (string, error)
	DownloadFileFromBucket(bucketURL string) (*bufio.Scanner, error)
}
