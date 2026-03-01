// Package pkgminio is general utils for minio storage
package pkgminio

import (
	pkgconstants "github.com/milfan/mp-go-pkg/constants"
	"github.com/minio/minio-go/v7"
)

type configMinio struct {
	runMode         pkgconstants.AppRunMode
	isLocal         bool
	endpoint        string
	accessKeyID     string
	secretAccessKey string
	useSSL          bool
	minioClient     *minio.Client
	baseDomain      string
	bucketName      string
	pathURL         string

	configTransportMinio
}

type OptFunc func(*configMinio)

func SetRunMode(runMode pkgconstants.AppRunMode) func(*configMinio) {
	return func(cm *configMinio) {
		cm.runMode = runMode
	}
}

func SetIsLocal(isLocal bool) func(*configMinio) {
	return func(cm *configMinio) {
		cm.isLocal = isLocal
	}
}

func SetEndpoint(endpoint string) func(*configMinio) {
	return func(cm *configMinio) {
		cm.endpoint = endpoint
	}
}

func SetAccessKeyID(accessKeyID string) func(*configMinio) {
	return func(cm *configMinio) {
		cm.accessKeyID = accessKeyID
	}
}

func SetSecretAccessKey(secretAccessKey string) func(*configMinio) {
	return func(cm *configMinio) {
		cm.secretAccessKey = secretAccessKey
	}
}

func SetUseSSL(useSSL bool) func(*configMinio) {
	return func(cm *configMinio) {
		cm.useSSL = useSSL
	}
}

func SetBaseDomain(baseDomain string) func(*configMinio) {
	return func(cm *configMinio) {
		cm.baseDomain = baseDomain
	}
}

func SetBucketName(bucketName string) func(*configMinio) {
	return func(cm *configMinio) {
		cm.bucketName = bucketName
	}
}

func SetPathURL(path string) func(*configMinio) {
	return func(cm *configMinio) {
		cm.pathURL = path
	}
}

type commonMinio struct {
	configMinio
}
