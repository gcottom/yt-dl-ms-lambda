package main

import (
	//"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	//"github.com/gcottom/yt-dl-ms-lambda/pkg/conf"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/handlers"
)

func main() {
	//os.Setenv("AWS_ACCESS_KEY_ID", conf.AccessKeyId)
	//os.Setenv("AWS_SECRET_ACCESS_KEY", conf.SecretAccessKey)
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		return handlers.GetTrackConvertedHandler(req)
	default:
		return handlers.UnhandledMethod()
	}
}
