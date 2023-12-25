package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"os"
	"strconv"

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
		fmt.Println(err)
		return nil, err
	}
	t, err := strconv.Atoi(req.QueryStringParameters["t"])
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	v, err := strconv.Atoi(req.QueryStringParameters["v"])
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	expression, err := govaluate.NewEvaluableExpression(string(alg))
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	parameters := make(map[string]interface{}, 8)
	parameters["t"] = t

	result, err := expression.Evaluate(parameters)
	if err != nil {
		fmt.Println(err)
		return nil, err
	}
	if int(result.(float64)) == v {
		claims := jwt.MapClaims{}
		now := time.Now()
		claims["exp"] = jwt.NewNumericDate(now.Add(300 * time.Second))
		claims["iat"] = jwt.NewNumericDate(now)
		claims["nbf"] = jwt.NewNumericDate(now.Add(-60 * time.Second))
		claims["authorized"] = true
		claims["user"] = "yt-dl-ui"
		nonce, err := uuid.NewRandom()
		if err != nil {
			fmt.Println(err)
			return handlers.ApiResponse(http.StatusBadRequest, handlers.ErrorBody{aws.String(err.Error())})
		}
		claims["nonce"] = nonce.String()
		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(secret)
		if err != nil {
			fmt.Println(err)
			return handlers.ApiResponse(http.StatusBadRequest, handlers.ErrorBody{aws.String(err.Error())})
		}
		return handlers.ApiResponse(http.StatusOK, TokenResponse{tokenString})
	}
	return handlers.ApiResponse(http.StatusBadRequest, handlers.ErrorBody{aws.String(err.Error())})
}
