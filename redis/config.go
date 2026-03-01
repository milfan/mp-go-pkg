package pkgredis

import (
	"strconv"

	"github.com/sirupsen/logrus"
)

type redisConfig struct {
	hostAddress  string
	password     string
	dbNumber     int
	retryConnect bool
}

type RedisOption func(*redisConfig)

func SetHostAddress(host string) RedisOption {
	return func(o *redisConfig) {
		o.hostAddress = host
	}
}

func SetDBNumber(s string) RedisOption {
	return func(o *redisConfig) {
		dbNum, err := strconv.Atoi(s)
		if err != nil {
			logrus.Fatalf("err: %s", err)
		}

		o.dbNumber = dbNum
	}
}

func WithPassword(s string) RedisOption {
	return func(o *redisConfig) {
		o.password = s
	}
}

func WithRetryConnect(val bool) RedisOption {
	return func(o *redisConfig) {
		o.retryConnect = val
	}
}
