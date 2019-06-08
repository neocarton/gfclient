package glclient

import (
	"bytes"
	"fmt"
	"net/http"
	urlUtil "net/url"
	"reflect"
	"regexp"
	"strings"
	"time"

	"github.com/neocarton/glog"
	"github.com/neocarton/gsin"
)

// Tag reflection tag
const (
	Tag          = "gfclient"
	TagSeparator = ":"
	TagPath      = "path"
	TagQuery     = "query"
	TagHeader    = "header"
	TagCookie    = "cookie"
	TagBody      = "body"
)

// Mime types used when doing request data reading and response data writing.
const (
	MimeText             = "text/plain"
	MimeJSON             = "application/json"
	MimeXML              = "application/xml"
	MimeHTML             = "text/html"
	MimeForm             = "application/x-www-form-urlencoded"
	MimeFormMultiplePart = "multipart/form-data"

	HeaderContentType = "Content-Type"
)

const (
	urlParamPattern      = "(.*)(<[%s]+>)(.*)" // http://host/api/<param1>/something?param2=<param2>
	urlParamValuePattern = "$1%s$2"
)

type (
	thisPackage struct{}

	// Config config
	Config interface {
		Timeout() time.Duration
	}

	// Client API client interface
	Client interface {
		Name() string
		Config() Config
		BaseURL() string
		ConsumeContentType() string
		ProduceContentType() string
		Do(result interface{}, method string, path string, params interface{}) error
	}

	// BaseClient base client
	BaseClient struct {
		Client
		name               string
		config             Config
		baseURL            string
		consumeContentType string
		produceContentType string
	}

	// SampleParams struct {
	// 	id          string      `gfclient:"path:id"`
	// 	name        string      `gfclient:"path"`
	// 	limit       int32       `gfclient:"query"`
	// 	offset      int32       `gfclient:"query"`
	// 	accessToken string      `gfclient:"header:Authorization"`
	// 	session     string      `gfclient:"cookie:session"`
	// 	data        interface{} `gfclient:"body"`
	// }
	request struct {
		Paths        map[string]string
		Queries      map[string]string
		Headers      map[string]string
		Cookies      map[string]string
		DataAsObject interface{}
	}
)

var (
	logger = glog.GetLoggerByPackage(&thisPackage{})
)

// NewAPI Create API client
func NewAPI(c Client, name string, config Config, baseURL string, consumeContentType string, produceContentType string) Client {
	baseClient := c.(BaseClient)
	baseClient.name = name
	baseClient.config = config
	baseClient.baseURL = baseURL
	baseClient.consumeContentType = consumeContentType
	baseClient.produceContentType = produceContentType
	return baseClient
}

// Name return client name
func (api BaseClient) Name() string {
	return api.name
}

// Config return config
func (api BaseClient) Config() Config {
	return api.config
}

// BaseURL return base URL
func (api BaseClient) BaseURL() string {
	return api.baseURL
}

// ConsumeContentType return consume Content-Type
func (api BaseClient) ConsumeContentType() string {
	return api.consumeContentType
}

// ProduceContentType return produce Content-Type
func (api BaseClient) ProduceContentType() string {
	return api.produceContentType
}

// Do do send HTTP request
func (api BaseClient) Do(result interface{}, method string, path string, params interface{}) error {
	logger.Debugf("Start to call API %+v with parameter %+v", api, params)
	// Prepare request
	req, err := api.toHTTPRequest(method, path, params)
	if err != nil {
		message := fmt.Sprintf("Failed to create HTTP request for API %+v with parameter %+v", api, params)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	// Send HTTP request
	httpClient := &http.Client{Timeout: api.Config().Timeout()}
	res, err := httpClient.Do(req)
	if err != nil {
		message := fmt.Sprintf("Failed to call API: %+v", req)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	logger.Debugf("API %+v with parameter %+v responds %+v", api, params, res)
	// Parse response
	defer res.Body.Close()
	var data []byte
	_, err = res.Body.Read(data)
	if err != nil {
		message := fmt.Sprintf("Failed to read response from API %+v: %+v", req, res)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	contentType := api.ProduceContentType()
	err = toObject(result, data, contentType)
	if err != nil {
		message := fmt.Sprintf("Failed to parse response from API %+v", req)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	return err
}

func (api *BaseClient) toHTTPRequest(method string, path string, param interface{}) (*http.Request, error) {
	// Refine method
	if method == "" {
		method = http.MethodGet
	}
	// Build URL
	req := toRequest(param)
	url := joinURL(api.BaseURL(), path)
	url = buildURL(url, req.Paths, req.Queries)
	// Convert post data (if any) to byte array
	contentType := api.ConsumeContentType()
	hasData := (req.DataAsObject != nil)
	var data []byte
	var err error
	if hasData {
		data, err = toBytes(req.DataAsObject, contentType)
		if err != nil {
			return nil, err
		}
	}
	// Create HTTP request
	httpReq, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	// Set headers
	header := &httpReq.Header
	for key, value := range req.Headers {
		header.Set(key, value)
	}
	if hasData && header.Get(contentType) == "" {
		header.Set(HeaderContentType, contentType)
	}
	// Set cookies
	for key, value := range req.Cookies {
		httpReq.AddCookie(&http.Cookie{Name: key, Value: value})
	}
	return httpReq, nil
}

func joinURL(baseURL string, path string) string {
	url := baseURL
	if strings.HasSuffix(url, "/") {
		url = url[:len(url)-1]
	}
	if strings.HasPrefix(path, "/") {
		path = path[1:]
	}
	url = url + "/" + path
	return url
}

func toRequest(param interface{}) *request {
	req := &request{}
	typ := reflect.TypeOf(param)
	val := reflect.ValueOf(param)
	for i := 0; i < val.NumField(); i++ {
		parseTag(req, typ.Field(i), val.Field(i))
	}
	return req
}

func parseTag(req *request, field reflect.StructField, fieldValue reflect.Value) {
	// Parse tag
	tagValue, found := field.Tag.Lookup(Tag)
	if !found {
		return
	}
	tagValueArray := strings.Split(tagValue, TagSeparator)
	tagName := tagValueArray[0]
	var param string
	if len(tagValueArray) > 1 {
		param = tagValueArray[1]
	}
	if len(param) == 0 {
		param = field.Name
	}
	// Save information
	switch tagName {
	case TagPath:
		req.Paths[param] = fieldValue.String()
	case TagQuery:
		req.Queries[param] = fieldValue.String()
	case TagHeader:
		req.Headers[param] = fieldValue.String()
	case TagCookie:
		req.Cookies[param] = fieldValue.String()
	case TagBody:
		req.DataAsObject = fieldValue.Interface()
	}
}

func buildURL(urlPattern string, pathParams map[string]string, queries map[string]string) string {
	url := urlPattern
	for key, value := range pathParams {
		pattern := fmt.Sprintf(urlParamPattern, key)
		valueParttern := fmt.Sprintf(urlParamValuePattern, urlUtil.PathEscape(value))
		var regex = regexp.MustCompile(pattern)
		url = regex.ReplaceAllString(url, valueParttern)
	}
	queryString := toQueryString(queries)
	if len(queryString) > 0 {
		url += "?" + queryString
	}
	return url
}
