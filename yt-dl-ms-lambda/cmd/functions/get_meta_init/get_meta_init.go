package main

import (
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/gcottom/yt-dl-ms-lambda/pkg/handlers"
)

func main() {
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		return handlers.GetMetaInitHandler(req)
	default:
		return handlers.UnhandledMethod()
	}
}
