package glclient

import (
	"bytes"
	"fmt"
	"net/http"

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
)

// NewGoHTTPClient Create GoHTTPClient
func NewGoHTTPClient(name string, baseURL string, config Config) *GoHTTPClient {
	c := &GoHTTPClient{
		name:       name,
		baseURL:    baseURL,
		config:     config,
		httpClient: &http.Client{Timeout: config.Timeout()},
	}
	return c
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
	logger.Debugf("Start to call API with base-URL '%s', path '%s', method '%s' and parameter %+v", client.baseURL, path, method, params)
	// Prepare request
	req, err := BuildRequest(method, client.baseURL, path, params, consumeContentType, produceContentType)
	if err != nil {
		message := fmt.Sprintf("Failed to build request for API with base-URL '%s', path '%s', method '%s' and parameter %+v", client.baseURL, path, method, params)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	httpReq, err := toHTTPRequest(req)
	if err != nil {
		message := fmt.Sprintf("Failed to create HTTP request for API %+v", req)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	// Send HTTP request
	res, err := client.doSend(httpReq)
	if err != nil {
		message := fmt.Sprintf("Failed to call API %+v", req)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	logger.Debugf("API %+v responded: %+v", req, res)
	// Check status
	status := res.StatusCode
	if status != 200 {
		message := fmt.Sprintf("API %+v responded with status '%d'", req, status)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
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
	err = toObject(result, data, req.ProduceContentType)
	if err != nil {
		message := fmt.Sprintf("Failed to parse response from API %+v", req)
		logger.Errorf(message, err)
		err = gsin.InitError(&gsin.Error{}, message, err, nil)
		return err
	}
	return err
}

func (client *GoHTTPClient) doSend(req *http.Request) (*http.Response, error) {
	return client.httpClient.Do(req)
}

func toHTTPRequest(req *Request) (*http.Request, error) {
	// Create HTTP request
	httpReq, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer(req.Data))
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
