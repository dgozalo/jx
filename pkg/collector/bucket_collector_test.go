package collector

import (
	"fmt"
	"github.com/jenkins-x/jx/pkg/cloud/buckets/mocks"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/petergtz/pegomock"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestBucketCollector_CollectData(t *testing.T) {
	bytes := []byte("This is a test")
	outputName := "destination"
	bucketURL := "bucket://bucketName"

	mp := buckets_test.NewMockProvider()
	pegomock.When(mp.UploadFileToBucket(bytes, outputName, bucketURL)).ThenReturn(fmt.Sprintf("%s/%s", bucketURL, outputName), nil)

	collector := BucketCollector{
		bucketURL: bucketURL,
		provider:  mp,
	}

	finalURL, err := collector.CollectData(bytes, outputName)
	assert.NoError(t, err)
	assert.Equal(t, fmt.Sprintf("%s/%s", collector.bucketURL, outputName), finalURL)
}

func TestBucketCollector_CollectFiles(t *testing.T) {
	bucketURL := "bucket://bucketName"
	pwd, err := os.Getwd()
	assert.NoError(t, err)
	collectFilesPath := path.Join(pwd, "test_data", "collect_files")
	testFilePath := path.Join(collectFilesPath, "example.txt")
	exists, err := util.FileExists(testFilePath)
	assert.NoError(t, err)
	assert.True(t, exists)

	bytes, err := ioutil.ReadFile(testFilePath)
	assert.NoError(t, err)

	mp := buckets_test.NewMockProvider()
	pegomock.When(mp.UploadFileToBucket(bytes, "example/example.txt", bucketURL)).ThenReturn(fmt.Sprintf("%s/%s", bucketURL, "example/example.txt"), nil)

	collector := BucketCollector{
		bucketURL: bucketURL,
		provider:  mp,
	}

	urls, err := collector.CollectFiles([]string{path.Join(collectFilesPath, "*.txt")}, "example", collectFilesPath)
	assert.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, fmt.Sprintf("%s/%s", bucketURL, "example/example.txt"), urls[0], "There needs to be a URL pointing to the only file uploaded")
}
