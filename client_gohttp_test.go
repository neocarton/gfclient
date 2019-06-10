package glclient

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
)

const (
	testClientName = "mockedClient"
	testBaseURL    = "http://testurl/root/"
)

type (
	mockedAPIClient struct {
		GoHTTPClient
	}

	getUserParams struct {
		ID int `gfclient:"path:id"`
	}

	getUserResult struct {
		ID       int    `json:"id"`
		UserName string `json:"username"`
	}
)

var testConfig = &DefaultConfig{}
var mockedResponseBody []byte
var mockedResponseError error

func newMockedAPIClient() *mockedAPIClient {
	c := &mockedAPIClient{*NewGoHTTPClient(testClientName, testBaseURL, testConfig)}
	// TODO use reflection to mock
	return c
}

// doSend is a mock function
func (api *mockedAPIClient) doSend(req *http.Request) (*http.Response, error) {
	logger.Debugf("Executing mocked doSend()")
	body := ioutil.NopCloser(bytes.NewReader([]byte(mockedResponseBody)))
	res := &http.Response{Body: body}
	return res, mockedResponseError
}

// getUser sample implementation of API call to /user/<id>
func (api *mockedAPIClient) getUser(id int) (*getUserResult, error) {
	params := &getUserParams{ID: id}
	result := &getUserResult{}
	err := api.Do(result, MethodGet, "/user/<id>", params, MimeJSON, MimeJSON)
	return result, err
}

func TestDoWithPathParam(t *testing.T) {
	// Mock result
	expUser := &getUserResult{ID: 1, UserName: "test_user"}
	json.Unmarshal(mockedResponseBody, expUser)
	// Test
	testClient := newMockedAPIClient()
	user, err := testClient.getUser(expUser.ID)
	if err != nil {
		t.Errorf("Test failed with error: %+v", err)
	}
	if user != expUser {
		t.Errorf("Expecting %+v, Actual was %+v", expUser, user)
	}
}
