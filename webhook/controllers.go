package webhook

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/reivaj05/GoConfig"
	"github.com/reivaj05/GoJSON"
	"github.com/reivaj05/GoLogger"
	"github.com/reivaj05/GoRequester"
	"github.com/reivaj05/GoServer"
)

const (
	divorceLegalProcess    = "DIVORCE_LEGAL_PROCESS"
	adoptionLegalProcess   = "ADOPTION_LEGAL_PROCESS"
	testamentLegalProcess  = "TESTAMENT_LEGAL_PROCESS"
	corruptionLegalProcess = "CORRUPTION_LEGAL_PROCESS"
	otherLegalProcess      = "OTHER_LEGAL_PROCESS"
)

var legalData = [][]string{
	[]string{"Divorce", divorceLegalProcess},
	[]string{"Adoption", adoptionLegalProcess},
	[]string{"Testament", testamentLegalProcess},
	[]string{"Corruption", corruptionLegalProcess},
	[]string{"Other", otherLegalProcess},
}

func getWebhookHandler(rw http.ResponseWriter, req *http.Request) {
	verifyToken, hubMode, hubChallenge := getRequestParams(req)
	if hubMode == "subscribe" && verifyToken == "verify_token" {
		GoServer.SendResponseWithStatus(rw, hubChallenge, http.StatusOK)
	} else {
		GoLogger.LogError("Validation failed", map[string]interface{}{
			"verify_token": verifyToken,
		})
		GoServer.SendResponseWithStatus(rw, "", http.StatusForbidden)
	}
}

func getRequestParams(req *http.Request) (string, string, string) {
	return req.FormValue("hub.verify_token"), req.FormValue("hub.mode"),
		req.FormValue("hub.challenge")
}

func postWebhookHandler(rw http.ResponseWriter, req *http.Request) {
	data := parseRequestBody(req)
	if object, _ := data.GetStringFromPath("object"); object == "page" {
		processData(data)
		GoServer.SendResponseWithStatus(rw, "", http.StatusOK)
		return
	}
	GoServer.SendResponseWithStatus(rw, "", http.StatusBadRequest)
}

func parseRequestBody(req *http.Request) *GoJSON.JSONWrapper {
	body, _ := GoServer.ReadBodyRequest(req)
	data, _ := GoJSON.New(body)
	return data
}

func processData(data *GoJSON.JSONWrapper) {
	entries := data.GetArrayFromPath("entry")
	processEntries(entries)
}

func processEntries(entries []*GoJSON.JSONWrapper) {
	for _, entry := range entries {
		messages := entry.GetArrayFromPath("messaging")
		processMessages(messages)
	}
}

func processMessages(messages []*GoJSON.JSONWrapper) {
	for _, msg := range messages {
		processMessage(msg)
	}
}

func processMessage(message *GoJSON.JSONWrapper) {
	if message.HasPath("optin") {
		fmt.Println("Implement authentication")
	} else if message.HasPath("message") {
		sendMessageBackToUser(message)
	} else if message.HasPath("delivery") {
		fmt.Println("Implement delivery")
	} else if message.HasPath("postback") {
		fmt.Println("Implement postback")
	} else if message.HasPath("read") {
		fmt.Println("Implement read")
	} else if message.HasPath("account_linking") {
		fmt.Println("Implement account_linking")
	} else {
		GoLogger.LogInfo("Webhook received unknown event", nil)
	}
}

func sendMessageBackToUser(message *GoJSON.JSONWrapper) {
	fmt.Println("message")
	senderID, _ := message.GetStringFromPath("sender.id")
	text, _ := message.GetStringFromPath("message.text")
	GoLogger.LogInfo("Message received", map[string]interface{}{
		"user":    senderID,
		"message": text,
	})
	if text == "start" {
		sendStartMessage(senderID)
	} else if strings.HasPrefix(text, "weather") {
		sendWeatherMessage(text, senderID)
	} else if strings.HasPrefix(text, "image me") {
		sendImageMessage(text, senderID)
	} else {
		sendTextMessage(text, senderID)
	}
}

func sendStartMessage(senderID string) {
	// TODO: Clean and refactor, no sensitive
	startBody := createStartBody()
	callSendAPI(senderID, startBody)
}

func sendWeatherMessage(text, senderID string) {
	city := strings.Split(text, " ")[1]
	requesterObj := requester.New()
	values := url.Values{}
	values.Add("APPID", GoConfig.GetConfigStringValue("weatherAPIKey"))
	values.Add("q", city)
	config := &requester.RequestConfig{
		Method: "GET",
		URL:    GoConfig.GetConfigStringValue("weatherEndpoint"),
		Values: values,
	}
	response, _, _ := requesterObj.MakeRequest(config)
	sendTextMessage(createWeatherResponse(response), senderID)
}

func sendImageMessage(text, senderID string) {
	imageQuery := strings.Split(text, " ")[2]
	requesterObj := requester.New()
	values := url.Values{}
	values.Add("key", GoConfig.GetConfigStringValue("imageAPIKEY"))
	values.Add("q", imageQuery)
	values.Add("per_page", "3")
	config := &requester.RequestConfig{
		Method: "GET",
		URL:    GoConfig.GetConfigStringValue("imageEndpoint"),
		Values: values,
	}
	response, _, _ := requesterObj.MakeRequest(config)

	imageBody := createImageBody(response)
	callSendAPI(senderID, imageBody)
}

func createImageBody(response string) *GoJSON.JSONWrapper {
	imageBody, _ := GoJSON.New("{}")
	imageBody.SetValueAtPath("attachment.type", "template")
	imageBody.SetValueAtPath("attachment.payload.template_type", "generic")
	jsonImage, _ := GoJSON.New(response)
	elements := createImageElementsArray(jsonImage)
	imageBody.CreateJSONArrayAtPathWithArray("attachment.payload.elements", elements)
	return imageBody
}

func createImageElementsArray(jsonImage *GoJSON.JSONWrapper) (elements []*GoJSON.JSONWrapper) {

	for _, image := range jsonImage.GetArrayFromPath("hits") {
		previewURL, _ := image.GetStringFromPath("pageURL")
		pageURL, _ := image.GetStringFromPath("previewURL")
		element, _ := GoJSON.New("{}")
		element.SetValueAtPath("title", "Image")
		element.SetValueAtPath("item_url", previewURL)
		element.SetValueAtPath("image_url", pageURL)
		elements = append(elements, element)
	}
	return elements
}

func createWeatherResponse(weatherResponse string) string {
	jsonWeather, _ := GoJSON.New(weatherResponse)
	text := "The weather for today in %s is %s"
	weather, _ := jsonWeather.GetArrayFromPath("weather")[0].GetStringFromPath("main")
	city, _ := jsonWeather.GetStringFromPath("name")
	return fmt.Sprintf(text, city, weather)
}

func createStartBody() *GoJSON.JSONWrapper {
	startBody, _ := GoJSON.New("{}")
	startBody.SetValueAtPath("text", "Pick a legal process I can help you with:")
	startBody.CreateJSONArrayAtPath("quick_replies")
	startBody = addQuickRepliesToStartBody(startBody)
	return startBody
}

func addQuickRepliesToStartBody(startBody *GoJSON.JSONWrapper) *GoJSON.JSONWrapper {
	for _, item := range legalData {
		element, _ := GoJSON.New("{}")
		element.SetValueAtPath("content_type", "text")
		element.SetValueAtPath("title", item[0])
		element.SetValueAtPath("payload", item[1])
		startBody.ArrayAppendInPath("quick_replies", element)
	}
	return startBody
}

func sendTextMessage(text, senderID string) {
	body, _ := GoJSON.New("{}")
	body.SetValueAtPath("text", text)
	callSendAPI(senderID, body)
}

func callSendAPI(senderID string, body *GoJSON.JSONWrapper) {
	requesterObj := requester.New()
	config := createRequestConfig(senderID, body)
	response, status, err := requesterObj.MakeRequest(config)
	GoLogger.LogInfo("Request sent", map[string]interface{}{
		"response": response,
		"status":   status,
		"err":      err,
	})
}

func createRequestConfig(
	senderID string, body *GoJSON.JSONWrapper) *requester.RequestConfig {

	return &requester.RequestConfig{
		Method:  "POST",
		URL:     GoConfig.GetConfigStringValue("messengerPostURL"),
		Values:  createRequestValues(),
		Body:    createRequestBody(senderID, body),
		Headers: createRequestHeaders(),
	}
}

func createRequestValues() url.Values {
	values := url.Values{}
	values.Add("access_token", GoConfig.GetConfigStringValue("pageAccessToken"))
	return values
}

func createRequestBody(senderID string, body *GoJSON.JSONWrapper) []byte {
	replyBody, _ := GoJSON.New("{}")
	replyBody.SetValueAtPath("recipient.id", senderID)
	replyBody.SetObjectAtPath("message", body)
	return []byte(replyBody.ToString())
}

func createRequestHeaders() map[string]string {
	return map[string]string{
		"Content-Type": "application/json",
	}
}

func gitWebhookHandler(rw http.ResponseWriter, req *http.Request) {
	data := parseRequestBody(req)
	repoName, _ := data.GetStringFromPath("repository.name")
	repoURL, _ := data.GetStringFromPath("repository.url")
	pusherName, _ := data.GetStringFromPath("pusher.name")

	gitBody, _ := GoJSON.New("{}")
	gitBody.SetValueAtPath("attachment.type", "template")
	gitBody.SetValueAtPath("attachment.payload.template_type", "generic")
	element, _ := GoJSON.New("{}")
	element.SetValueAtPath("title", pusherName+" has pushed to repo: "+repoName)
	button, _ := GoJSON.New("{}")
	button.SetValueAtPath("type", "web_url")
	button.SetValueAtPath("url", repoURL)
	button.SetValueAtPath("title", "View repo")
	element.CreateJSONArrayAtPathWithArray("buttons", []*GoJSON.JSONWrapper{button})
	gitBody.CreateJSONArrayAtPathWithArray("attachment.payload.elements", []*GoJSON.JSONWrapper{element})
	callSendAPI("1137104706416635", gitBody)
	GoServer.SendResponseWithStatus(rw, data.ToString(), http.StatusOK)
}
