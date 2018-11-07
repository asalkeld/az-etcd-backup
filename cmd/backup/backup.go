package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"
	"github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	"github.com/coreos/etcd-operator/pkg/backup"
	"github.com/coreos/etcd-operator/pkg/backup/writer"
	"github.com/coreos/etcd-operator/pkg/util/azureutil/absfactory"
	"github.com/coreos/etcd-operator/pkg/util/k8sutil"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"

	"github.com/openshift/backup/pkg/log"
)

var (
	logLevel    = flag.String("loglevel", "Debug", "Valid values are Debug, Info, Warning, Error")
	interval    = flag.Duration("interval", 5*time.Second, "How often to retry the download.")
	blobName    = flag.String("blobname", "", "Name of the blob to download")
	destination = flag.String("destination", "", "Where to place the blob on the filesystem")
	gitCommit   = "unknown"
)

type backupCmd struct {
	abs *storage.BlobStorageClient
}

func handleBackup(ctx context.Context, kubecli kubernetes.Interface, s *v1beta2.ABSBackupSource, endpoints []string, clientTLSSecret, namespace string) error {
	cli, err := absfactory.NewClientFromSecret(kubecli, namespace, s.ABSSecret)
	if err != nil {
		return err
	}

	var tlsConfig *tls.Config
	if tlsConfig, err = generateTLSConfig(kubecli, clientTLSSecret, namespace); err != nil {
		return err
	}

	bm := backup.NewBackupManagerFromWriter(kubecli, writer.NewABSWriter(cli.ABS), tlsConfig, endpoints, namespace)

	_, _, err = bm.SaveSnap(ctx, s.Path)
	if err != nil {
		return fmt.Errorf("failed to save snapshot (%v)", err)
	}
	return nil
}

func handleBackupPrune(ctx context.Context) error {
	// TODO implement

	return nil
}

func handleRetrieve(ctx context.Context, kubecli kubernetes.Interface, absSecret, namespace string) error {
	cli, err := absfactory.NewClientFromSecret(kubecli, namespace, absSecret)
	if err != nil {
		return fmt.Errorf("Cannot get storage account backup: %v", err)
	}
	r := new(backupCmd)
	r.abs = cli.ABS

	for i := 1; i <= 10; i++ {
		err = r.copy(ctx, *blobName, *destination)
		if err != nil {
			logrus.Warnf("Error while getting az blob: %v->%v %v", *blobName, *destination, err)
			<-time.After(*interval)
		} else {
			return nil
		}
	}
	return fmt.Errorf("tried 10 times to copy backup file - still failed")

}

func main() {
	flag.Parse()
	logrus.SetLevel(log.SanitizeLogLevel(*logLevel))
	logrus.SetFormatter(&logrus.TextFormatter{FullTimestamp: true})
	logrus.Printf("backup starting, git commit %s", gitCommit)

	ctx := context.Background()
	clientset := k8sutil.MustNewKubeClient()
	var err error
	if false {
		err = handleRetrieve(ctx, clientset, "etcd-backup-abs-credentials", "openshift-etcd")
	} else {
		absBackupSource := &v1beta2.ABSBackupSource{Path: "etcd/backup-now", ABSSecret: "etcd-backup-abs-credentials"}
		etcdEndpoints := []string{"https://master-000000:2380", "https://master-000001:2380", "https://master-000002:2380"}
		err = handleBackup(ctx, clientset, absBackupSource, etcdEndpoints, "etcd-client-tls", "openshift-etcd")
		if err != nil {
			// TODO delete the bad backup
		} else {
			handleBackupPrune(ctx)
		}
	}
	if err != nil {
		logrus.Fatal(err)
	}
}
