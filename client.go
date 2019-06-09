package glclient

import (
	"github.com/neocarton/glog"
)

type (
	thisPackage struct{}

	// Client API client interface
	Client interface {
		Name() string
		BaseURL() string
		Config() Config
		Do(result interface{}, method string, path string, params interface{},
			consumeContentType string, produceContentType string) error
	}
)

var (
	logger = glog.GetLoggerByPackage(&thisPackage{})
)
