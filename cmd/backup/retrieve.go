package main

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/coreos/etcd-operator/pkg/backup/util"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/wait"
)

func (r *backupCmd) open(path string) (io.ReadCloser, error) {
	container, key, err := util.ParseBucketAndKey(path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse abs container and key: %v", err)
	}

	containerRef := r.abs.GetContainerReference(container)
	containerExists, err := containerRef.Exists()
	if err != nil {
		return nil, err
	}

	if !containerExists {
		return nil, fmt.Errorf("container %v does not exist", container)
	}

	blob := containerRef.GetBlobReference(key)
	return blob.Get(&storage.GetBlobOptions{})
}

func (r *backupCmd) copy(ctx context.Context, srcBlob, destPath string) error {
	logrus.Printf("copy blob %v to filesystem %v", srcBlob, destPath)
	var rc io.ReadCloser
	var err error
	err = wait.PollImmediateInfinite(time.Second, func() (bool, error) {
		rc, err = r.open(srcBlob)
		if err, ok := err.(storage.AzureStorageServiceError); ok && err.StatusCode == http.StatusNotFound {
			return false, nil
		}
		return err == nil, err
	})
	if err != nil {
		return err
	}
	defer rc.Close()
	logrus.Print("read blob")

	b, err := ioutil.ReadAll(rc)
	if err != nil {
		return err
	}
	logrus.Printf("creating %s", destPath)
	df, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = df.Write(b)
	return err
}
