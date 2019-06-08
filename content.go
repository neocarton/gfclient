package glclient

import (
	"encoding/json"
	"errors"
	"fmt"
	urlUtil "net/url"
	"strings"

	"github.com/neocarton/gsin"
)

func toObject(object interface{}, data []byte, contentType string) error {
	// Parse body
	var err error
	switch contentType {
	case MimeJSON:
		err = json.Unmarshal(data, &object)
	default:
		message := fmt.Sprintf("Unknown content-type '%s'", contentType)
		err = errors.New(message)
	}
	if err != nil {
		message := fmt.Sprintf("Failed to parse data as content-type '%s'", contentType)
		logger.Error(message)
		context := map[string]interface{}{"data": data}
		err = gsin.InitError(&gsin.Error{}, message, err, context)
	}
	return err
}

func toBytes(object interface{}, contentType string) ([]byte, error) {
	// Parse body
	var data []byte
	var err error
	switch contentType {
	case MimeJSON:
		data, err = json.Marshal(object)
	default:
		message := fmt.Sprintf("Unknown content-type '%s'", contentType)
		err = errors.New(message)
	}
	if err != nil {
		message := fmt.Sprintf("Failed to convert object to content-type '%s'", contentType)
		logger.Error(message)
		context := map[string]interface{}{"object": object}
		err = gsin.InitError(&gsin.Error{}, message, err, context)
	}
	return data, err
}

func toQueryString(queries map[string]string) string {
	var queryList []string
	for key, value := range queries {
		query := urlUtil.QueryEscape(key) + "=" + urlUtil.QueryEscape(value)
		queryList = append(queryList, query)
	}
	return strings.Join(queryList, "&")
}
