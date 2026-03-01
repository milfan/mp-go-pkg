// Package pkgminio is general utils for minio storage
package pkgminio

type configTransportMinio struct {
	iddleConn           int
	iddleConnHost       int
	iddleConnTimeout    int
	iddleConnAlive      int
	tlsHandShakeTimeout int
	tlsSkipVerify       bool
}

func SetIddleConn(val int) func(*configMinio) {
	return func(cm *configMinio) {
		cm.iddleConn = val
	}
}

func SetIddleConnHost(val int) func(*configMinio) {
	return func(cm *configMinio) {
		cm.iddleConnHost = val
	}
}

func SetIddleConnTimeout(val int) func(*configMinio) {
	return func(cm *configMinio) {
		cm.iddleConnTimeout = val
	}
}

func SetIddleConnAlive(val int) func(*configMinio) {
	return func(cm *configMinio) {
		cm.iddleConnAlive = val
	}
}

func SetTLSHandShakeTimeout(val int) func(*configMinio) {
	return func(cm *configMinio) {
		cm.tlsHandShakeTimeout = val
	}
}

func SetTLSSkipVerify(val bool) func(*configMinio) {
	return func(cm *configMinio) {
		cm.tlsSkipVerify = val
	}
}
