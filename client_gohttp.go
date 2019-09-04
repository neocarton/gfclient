package glclient

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"git.veep.tech/veep/2/common/convertor"

	"github.com/neocarton/glog"
	"github.com/neocarton/gsin"
)

type (
	// GoHTTPClient client implemented using net/http
	GoHTTPClient struct {
		Client
		name       string
		baseURL    string
		config     Config
		httpClient *http.Client
	}

	// Contents request content
	Contents struct {
		Body interface{} `gfclient:"body"`
	}
)

// NewGoHTTPClient Create GoHTTPClient
func NewGoHTTPClient(name string, baseURL string, config Config) *GoHTTPClient {
	client := &GoHTTPClient{}
	client.Init(name, baseURL, config)
	return client
}

// Init initialize GoHTTPClient
func (client *GoHTTPClient) Init(name string, baseURL string, config Config) {
	client.name = name
	client.baseURL = baseURL
	client.config = config
	client.httpClient = &http.Client{Timeout: config.Timeout()}
}

// Name return client name
func (client *GoHTTPClient) Name() string {
	return client.name
}

// BaseURL return base URL
func (client *GoHTTPClient) BaseURL() string {
	return client.baseURL
}

// Config return config
func (client *GoHTTPClient) Config() Config {
	return client.config
}

// Do do send HTTP request
func (client *GoHTTPClient) Do(result interface{}, method string, path string, params interface{},
	consumeContentType string, produceContentType string) error {
	logger.Debugf("Start to call API with method '%s', base-URL '%s', path '%s'", method, client.baseURL, path)
	// Prepare request
	req, err := BuildRequest(method, client.baseURL, path, params, consumeContentType, produceContentType)
	if err != nil {
		message := fmt.Sprintf("Failed to build request for API with base-URL '%s', path '%s', method '%s' and parameter %s",
			client.baseURL, path, method, convertor.ToJSON(params))
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	httpReq, err := toHTTPRequest(req)
	if err != nil {
		message := fmt.Sprintf("Failed to create HTTP request for API %s", convertor.ToJSON(req))
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	// Send HTTP request
	logger.Debugf("Sending request to API: %s", glog.AsJSON(req))
	res, err := client.httpClient.Do(httpReq)
	if err != nil {
		message := fmt.Sprintf("Failed to call API %s", convertor.ToJSON(req))
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	// Check status
	status := res.StatusCode
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	logger.Debugf("API %s '%s' '%s' responded with status '%d', headers %s and body:\n%s",
		method, client.baseURL, path, res.StatusCode, glog.AsJSON(res.Header), string(data))
	if status != 200 {
		message := fmt.Sprintf("API %s responded with status '%d'", convertor.ToJSON(req), status)
		logger.Errorf(message, nil)
		err = gsin.InitError(&gsin.Error{}, message, nil, nil)
		return err
	}
	// Parse response
	if err != nil {
		message := fmt.Sprintf("Failed to read response from API %s: %+v", convertor.ToJSON(req), res)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	err = toObject(result, data, req.ProduceContentType)
	if err != nil {
		message := fmt.Sprintf("Failed to parse response from API %s", convertor.ToJSON(req))
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	return nil
}

func toHTTPRequest(req *Request) (*http.Request, error) {
	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, req.URL, strings.NewReader(req.Data))
	if err != nil {
		return nil, err
	}
	// Set headers
	header := &httpReq.Header
	for key, value := range req.Headers {
		header.Set(key, value)
	}
	// Set cookies
	for key, value := range req.Cookies {
		httpReq.AddCookie(&http.Cookie{Name: key, Value: value})
	}
	return httpReq, nil
}
