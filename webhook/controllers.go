package webhook

import (
	"fmt"
	"net/http"
	"net/url"

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
	divorceLawyers         = "DIVORCE_LAWYERS"
	adoptionLawyers        = "ADOPTION_LAWYERS"
	testamentLawyers       = "TESTAMENT_LAWYERS"
	corruptionLawyers      = "CORRUPTION_LAWYERS"
	askLegalQuestion       = "ASK_LEGAL_QUESTION"
)

var legalData = [][]string{
	[]string{"Divorce", divorceLegalProcess},
	[]string{"Adoption", adoptionLegalProcess},
	[]string{"Testament", testamentLegalProcess},
	[]string{"Corruption", corruptionLegalProcess},
	[]string{"Ask question", askLegalQuestion},
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
	// TODO: Refactor
	if message.HasPath("optin") {
		fmt.Println("Implement authentication")
	} else if message.HasPath("message") {
		sendMessageBackToUser(message)
	} else if message.HasPath("delivery") {
		fmt.Println("Implement delivery")
	} else if message.HasPath("postback") {
		handlePostback(message)
	} else if message.HasPath("read") {
		fmt.Println("Implement read")
	} else if message.HasPath("account_linking") {
		fmt.Println("Implement account_linking")
	} else {
		GoLogger.LogInfo("Webhook received unknown event", nil)
	}
}

func sendMessageBackToUser(message *GoJSON.JSONWrapper) {
	// TODO: Refactor
	senderID, _ := message.GetStringFromPath("sender.id")
	text, _ := message.GetStringFromPath("message.text")
	GoLogger.LogInfo("Message received", map[string]interface{}{
		"user":    senderID,
		"message": text,
	})
	if message.HasPath("message.quick_reply") {
		payload, _ := message.GetStringFromPath("message.quick_reply.payload")
		process := ""
		payloadPostback := ""
		switch payload {
		case divorceLegalProcess:
			process = "Divorce"
			payloadPostback = divorceLawyers
		case adoptionLegalProcess:
			process = "Adoption"
			payloadPostback = adoptionLawyers
		case testamentLegalProcess:
			process = "Testament"
			payloadPostback = testamentLawyers
		case corruptionLegalProcess:
			process = "Corruption"
			payloadPostback = corruptionLawyers
		case askLegalQuestion:
			sendTextMessage("Type your question, I'll give you the best results I find", senderID)
			return
		case "subscribe":
			sendTextMessage("You have been subscribed", senderID)
			return
		case "nosubscribe":
			sendTextMessage("OK! come back soon!", senderID)
			return
		}
		sendLegalProcessMsg(process, payloadPostback, senderID)
		return
	}
	if text == "start" {
		sendStartMessage(senderID)
	} else if text == "help" {
		sendHelpMessage(senderID)
	} else if text == "subscribe" {
		sendSubscribeMessage(senderID)
	} else {
		// TODO: Train nlp model
		sendTextMessage("The answer is 42, don't look anymore", senderID)
	}
}

func sendLegalProcessMsg(process, payloadPostback, senderID string) {
	body := createLegalProcessBody(process, payloadPostback)
	callSendAPI(senderID, body)
}

func createLegalProcessBody(process, payloadPostback string) *GoJSON.JSONWrapper {
	// TODO: refactor
	legalBody, _ := GoJSON.New("{}")
	legalBody.SetValueAtPath("attachment.type", "template")
	legalBody.SetValueAtPath("attachment.payload.template_type", "button")
	legalBody.SetValueAtPath("attachment.payload.text", "Select the option you want")
	button, _ := GoJSON.New("{}")
	button.SetValueAtPath("type", "web_url")
	button.SetValueAtPath("url", "http://www.google.com")
	button.SetValueAtPath("title", "FAQ")

	button2, _ := GoJSON.New("{}")
	button2.SetValueAtPath("type", "postback")
	button2.SetValueAtPath("title", process+" lawyers")
	button2.SetValueAtPath("payload", payloadPostback)
	legalBody.CreateJSONArrayAtPathWithArray("attachment.payload.buttons", []*GoJSON.JSONWrapper{button, button2})

	return legalBody
}

func sendStartMessage(senderID string) {
	startBody := createStartBody()
	callSendAPI(senderID, startBody)
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

func sendHelpMessage(senderID string) {
	sendTextMessage("Hi, I'm your legal assistant and I'm gonna help you with everything you need", senderID)
	helpBody := createHelpBody()
	callSendAPI(senderID, helpBody)
}

func createHelpBody() *GoJSON.JSONWrapper {
	// TODO: Refactor
	helpBody, _ := GoJSON.New("{}")
	helpBody.SetValueAtPath("attachment.type", "template")
	helpBody.SetValueAtPath("attachment.payload.template_type", "button")
	helpBody.SetValueAtPath("attachment.payload.text", "Are you ready to start?")
	button, _ := GoJSON.New("{}")
	button.SetValueAtPath("type", "postback")
	button.SetValueAtPath("title", "Start")
	button.SetValueAtPath("payload", "start")
	helpBody.CreateJSONArrayAtPathWithArray("attachment.payload.buttons", []*GoJSON.JSONWrapper{button})
	return helpBody
}

func sendSubscribeMessage(senderID string) {
	subscribeBody := createSubscribeBody()
	callSendAPI(senderID, subscribeBody)
}

func createSubscribeBody() *GoJSON.JSONWrapper {
	subscribeBody, _ := GoJSON.New("{}")
	subscribeBody.SetValueAtPath("text", "Do you want to subscribe to daily legal tips?")
	subscribeBody.CreateJSONArrayAtPath("quick_replies")

	element, _ := GoJSON.New("{}")
	element.SetValueAtPath("content_type", "text")
	element.SetValueAtPath("title", "yes")
	element.SetValueAtPath("payload", "subscribe")
	subscribeBody.ArrayAppendInPath("quick_replies", element)
	element, _ = GoJSON.New("{}")
	element.SetValueAtPath("content_type", "text")
	element.SetValueAtPath("title", "no")
	element.SetValueAtPath("payload", "nosubscribe")
	subscribeBody.ArrayAppendInPath("quick_replies", element)
	return subscribeBody
}

func sendTextMessage(text, senderID string) {
	body, _ := GoJSON.New("{}")
	body.SetValueAtPath("text", text)
	callSendAPI(senderID, body)
}

func handlePostback(message *GoJSON.JSONWrapper) {
	payload, _ := message.GetStringFromPath("postback.payload")
	senderID, _ := message.GetStringFromPath("sender.id")
	// process := ""
	switch payload {
	case divorceLawyers:
		// process = "Divorce"
	case adoptionLawyers:
		// process = "Adoption"
	case testamentLawyers:
		// process = "Testament"
	case corruptionLawyers:
		// process = "Corruption"
	case "start":
		callSendAPI(senderID, createStartBody())
		return
	}
	body := createExpertsLawyersBody()
	callSendAPI(senderID, body)
}

func createExpertsLawyersBody() *GoJSON.JSONWrapper {
	expertsBody, _ := GoJSON.New("{}")
	expertsBody.SetValueAtPath("attachment.type", "template")
	expertsBody.SetValueAtPath("attachment.payload.template_type", "generic")
	var elements []*GoJSON.JSONWrapper
	for i := 0; i < 5; i++ {
		elements = append(elements, createExpertElement())
	}
	expertsBody.CreateJSONArrayAtPathWithArray("attachment.payload.elements", elements)
	return expertsBody
}

func createExpertElement() *GoJSON.JSONWrapper {
	// TODO: Refactor
	element, _ := GoJSON.New("{}")
	element.SetValueAtPath("title", "Robert Dobbins")
	element.SetValueAtPath("image_url", "https://cdndata.bigfooty.com/2016/08/282084_47459072e9490ce8ddfdc9c29b6adc2a.jpg")

	button, _ := GoJSON.New("{}")
	button.SetValueAtPath("type", "web_url")
	button.SetValueAtPath("url", "http://www.google.com")
	button.SetValueAtPath("title", "Website")

	button2, _ := GoJSON.New("{}")
	button2.SetValueAtPath("type", "web_url")
	button2.SetValueAtPath("url", "http://www.google.com")
	button2.SetValueAtPath("title", "Make appointment")

	button3, _ := GoJSON.New("{}")
	button3.SetValueAtPath("type", "phone_number")
	button3.SetValueAtPath("title", "Phone number")
	button3.SetValueAtPath("payload", "+15105551234")

	element.CreateJSONArrayAtPathWithArray("buttons", []*GoJSON.JSONWrapper{
		button, button2, button3})
	return element
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
