package webhook

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/gorilla/mux"
	"github.com/reivaj05/GoConfig"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ControllersTestSuite struct {
	suite.Suite
	assert              *assert.Assertions
	router              *mux.Router
	bodyMockRequest     string
	messengerMockServer *httptest.Server
}

func (suite *ControllersTestSuite) SetupSuite() {
	suite.assert = assert.New(suite.T())
	suite.router = createTestRouter()
	GoConfig.Init(&GoConfig.ConfigOptions{
		ConfigType: "json",
		ConfigFile: "config",
		ConfigPath: "..",
	})
	suite.createBodyMockRequest()
	suite.messengerMockServer = httptest.NewServer(http.HandlerFunc(
		suite.messengerMockHandler))
	GoConfig.SetConfigValue("messengerPostURL", suite.messengerMockServer.URL)
}

func (suite *ControllersTestSuite) messengerMockHandler(
	rw http.ResponseWriter, req *http.Request) {
	rw.WriteHeader(http.StatusOK)
}

func (suite *ControllersTestSuite) createBodyMockRequest() {
	suite.bodyMockRequest = `{
		"entry":[{
			"id":"1",
			"messaging":[{
				"message":{"mid":"","seq":30,"text":"testText"},
				"recipient":{"id":"2"},
				"sender":{"id":"3"},
				"timestamp":1.479938087162e+12
			}],
			"time":1.479938204128e+12
		}],
		"object":"page"
	}`
}

func createTestRouter() *mux.Router {
	router := mux.NewRouter()
	router.Methods("GET").Path("/webhook/").Handler(http.HandlerFunc(getWebhookHandler))
	router.Methods("POST").Path("/webhook/").Handler(http.HandlerFunc(postWebhookHandler))
	return router
}

func (suite *ControllersTestSuite) TestGetWebhookHandler() {
	values := url.Values{}
	values.Add("hub.mode", "subscribe")
	values.Add("hub.verify_token", "verify_token")
	URL := "http://localhost/webhook/?" + values.Encode()
	_, status := suite.makeRequest("GET", URL, "")
	suite.assert.Equal(http.StatusOK, status)
}

func (suite *ControllersTestSuite) TestGetWebhookHandlerValidationFailed() {
	_, status := suite.makeRequest("GET", "http://localhost/webhook/", "")
	suite.assert.Equal(http.StatusForbidden, status)
}

func (suite *ControllersTestSuite) TestPostWebhookHandlerWrongBody() {
	_, status := suite.makeRequest("POST", "http://localhost/webhook/", "{}")
	suite.assert.Equal(http.StatusBadRequest, status)
}

func (suite *ControllersTestSuite) TestPostWebhookHandlerGoodBody() {
	_, status := suite.makeRequest("POST",
		"http://localhost/webhook/", suite.bodyMockRequest)
	suite.assert.Equal(http.StatusOK, status)
}

func (suite *ControllersTestSuite) makeRequest(
	method, url, body string) (string, int) {

	rw := httptest.NewRecorder()
	request, _ := http.NewRequest(method, url, bytes.NewBuffer([]byte(body)))
	suite.router.ServeHTTP(rw, request)
	res, _ := ioutil.ReadAll(rw.Body)
	return string(res), rw.Code
}

func TestControllers(t *testing.T) {
	suite.Run(t, new(ControllersTestSuite))
}
