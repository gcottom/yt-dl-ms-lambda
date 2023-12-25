package main

import (
	"encoding/base64"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"

	"time"

	"github.com/Knetic/govaluate"
	"github.com/gcottom/yt-dl-ms-lambda/pkg/handlers"
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func main() {
	lambda.Start(handler)
}

func handler(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	switch req.HTTPMethod {
	case "GET":
		return getToken(req)
	default:
		return handlers.UnhandledMethod()
	}
}

type TokenResponse struct {
	Token string `json:"token"`
}

func getToken(req events.APIGatewayProxyRequest) (*events.APIGatewayProxyResponse, error) {
	secret := os.Getenv("JWT_SECRET")
	alg, err := base64.StdEncoding.DecodeString(os.Getenv("ALG"))
	if err != nil {
		return nil, err
	}
	t := req.QueryStringParameters["t"]
	v := req.QueryStringParameters["v"]
	expression, err := govaluate.NewEvaluableExpression(string(alg))
	if err != nil {
		return nil, err
	}
	parameters := make(map[string]interface{}, 8)
	parameters["t"] = t
	parameters["v"] = v

	result, err := expression.Evaluate(parameters)
	if result == true {
		claims := jwt.MapClaims{}
		now := time.Now()
		claims["exp"] = jwt.NewNumericDate(now.Add(300 * time.Second))
		claims["iat"] = jwt.NewNumericDate(now)
		claims["nbf"] = jwt.NewNumericDate(now.Add(-60 * time.Second))
		claims["authorized"] = true
		claims["user"] = "yt-dl-ui"
		nonce, err := uuid.NewRandom()
		if err != nil {
			return handlers.ApiResponse(http.StatusBadRequest, handlers.ErrorBody{aws.String(err.Error())})
		}
		claims["nonce"] = nonce.String()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(secret)
		if err != nil {
			return handlers.ApiResponse(http.StatusBadRequest, handlers.ErrorBody{aws.String(err.Error())})
		}
		return handlers.ApiResponse(http.StatusOK, TokenResponse{tokenString})
	}
	return handlers.ApiResponse(http.StatusBadRequest, handlers.ErrorBody{aws.String(err.Error())})
}
