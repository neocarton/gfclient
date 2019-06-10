package glclient

import (
	"fmt"
	"net/http"
	urlUtil "net/url"
	"reflect"
	"regexp"
	"strings"
)

// Tag reflection tag
//
// SampleParams struct {
// 	id          string      `gfclient:"path:id"`
// 	name        string      `gfclient:"path"`
// 	limit       int32       `gfclient:"query"`
// 	offset      int32       `gfclient:"query"`
// 	accessToken string      `gfclient:"header:Authorization"`
// 	session     string      `gfclient:"cookie:session"`
// 	data        interface{} `gfclient:"body"`
// }
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
	MethodGet    = http.MethodGet
	MethodPost   = http.MethodPost
	MethodPut    = http.MethodPut
	MethodDelete = http.MethodDelete

	MimeText             = "text/plain"
	MimeJSON             = "application/json"
	MimeXML              = "application/xml"
	MimeHTML             = "text/html"
	MimeForm             = "application/x-www-form-urlencoded"
	MimeFormMultiplePart = "multipart/form-data"

	HeaderConsumeContentType = "Content-Type"
	HeaderProduceContentType = "Accept"
)

const (
	urlParamPattern      = "(.*)(<%s>)(.*)" // http://host/api/<param1>/something?param2=<param2>
	urlParamValuePattern = "${1}%s${3}"
)

type (
	requestParameters struct {
		paths   map[string]string
		queries map[string]string
		headers map[string]string
		cookies map[string]string
		input   interface{}
	}

	// Request API request with all information
	Request struct {
		Method             string
		URL                string
		Headers            map[string]string
		Cookies            map[string]string
		Data               []byte
		ConsumeContentType string
		ProduceContentType string
	}
)

// BuildRequest build request from api definition and parameters
func BuildRequest(method string, baseURL string, path string, params interface{},
	consumeContentType string, produceContentType string) (*Request, error) {
	if len(method) == 0 {
		method = MethodGet
	}
	if len(consumeContentType) == 0 {
		consumeContentType = MimeJSON
	}
	if len(produceContentType) == 0 {
		produceContentType = MimeJSON
	}
	reqParams := parseParams(params)
	url := joinURL(baseURL, path)
	url = buildURL(url, reqParams.paths, reqParams.queries)
	// Convert post data (if any) to byte array
	object := reqParams.input
	hasData := (object != nil)
	var data []byte
	var err error
	if hasData {
		// Convert object to byte array
		data, err = toBytes(object, consumeContentType)
		if err != nil {
			return nil, err
		}
		// Set default Content-Type header if not set
		if reqParams.headers[HeaderConsumeContentType] == "" {
			reqParams.headers[HeaderConsumeContentType] = consumeContentType
		}
	}
	// Set default Accept header if not set
	if reqParams.headers[HeaderProduceContentType] == "" {
		reqParams.headers[HeaderProduceContentType] = consumeContentType
	}
	req := &Request{
		Method:             method,
		URL:                url,
		Headers:            reqParams.headers,
		Cookies:            reqParams.cookies,
		Data:               data,
		ConsumeContentType: consumeContentType,
		ProduceContentType: produceContentType,
	}
	return req, nil
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

func parseParams(params interface{}) *requestParameters {
	reqParams := &requestParameters{
		paths:   map[string]string{},
		queries: map[string]string{},
		headers: map[string]string{},
		cookies: map[string]string{},
	}
	typ := reflect.TypeOf(params).Elem()
	val := reflect.ValueOf(params).Elem()
	for i := 0; i < typ.NumField(); i++ {
		parseTag(reqParams, typ.Field(i), val.Field(i))
	}
	return reqParams
}

func parseTag(reqParams *requestParameters, field reflect.StructField, fieldValue reflect.Value) {
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
	value := fieldValue.Interface()
	switch tagName {
	case TagPath:
		reqParams.paths[param] = fmt.Sprintf("%v", value)
	case TagQuery:
		reqParams.queries[param] = fmt.Sprintf("%v", value)
	case TagHeader:
		reqParams.headers[param] = fmt.Sprintf("%v", value)
	case TagCookie:
		reqParams.cookies[param] = fmt.Sprintf("%v", value)
	case TagBody:
		reqParams.input = value
	}
}

func buildURL(urlPattern string, pathParams map[string]string, queries map[string]string) string {
	url := urlPattern
	for key, value := range pathParams {
		pattern := fmt.Sprintf(urlParamPattern, regexp.QuoteMeta(key))
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
