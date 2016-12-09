package webhook

import (
	"github.com/reivaj05/GoServer"
)

var Endpoints = []*GoServer.Endpoint{
	&GoServer.Endpoint{
		Method:  "GET",
		Path:    "/webhook/",
		Handler: getWebhookHandler,
	},
	&GoServer.Endpoint{
		Method:  "POST",
		Path:    "/webhook/",
		Handler: postWebhookHandler,
	},
}
