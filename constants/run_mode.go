// Package pkgconstants is for constant collections
package pkgconstants

import "errors"

type AppRunMode uint8
type MapAppRunMode map[string]AppRunMode

const (
	Development AppRunMode = iota + 1
	Production
)

var mapRunMode = MapAppRunMode{
	"development": Development,
	"production":  Production,
}

func GetAppRunMode(s string) (*AppRunMode, error) {
	if val, found := mapRunMode[s]; found {
		return &val, nil
	}

	return nil, errors.New("unknown run mode value")
}

func (r AppRunMode) GetLabel() string {
	return [...]string{
		"development",
		"production",
	}[r-1]
}
