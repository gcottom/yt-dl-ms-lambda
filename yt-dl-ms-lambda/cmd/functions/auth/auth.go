package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	jwt "github.com/golang-jwt/jwt/v5"
)

// Claims represents the structure of the JWT claims.
type Claims struct {
	jwt.RegisteredClaims
}

// Handler is your Lambda function handler.
func Handler(ctx context.Context, request events.APIGatewayCustomAuthorizerRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	tokenString := extractToken(request.AuthorizationToken)
	if tokenString == "" {
		//pol := generatePolicy("user", "Deny", request.MethodArn)
		return events.APIGatewayCustomAuthorizerResponse{Context: map[string]interface{}{"message": "MISSING_AUTHENTICATION_TOKEN", "error_description": "token missing", "description": "token not provided in authorization header"}}, errors.New("Missing_Authentication_Token")
	}
	secret := os.Getenv("JWT_SECRET")
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil // Replace with your actual secret key
	})

	if err != nil {
		fmt.Println("Error parsing JWT:", err)
		pol := generatePolicy("user", "Deny", request.MethodArn)
		return events.APIGatewayCustomAuthorizerResponse{PrincipalID: pol.PrincipalID, PolicyDocument: pol.PolicyDocument, Context: map[string]interface{}{"message": "UNAUTHORIZED", "error_description": "token invalid", "description": "error parsing jwt"}}, nil
	}
	exp, err := token.Claims.GetExpirationTime()
	if err != nil {
		fmt.Println("Error parsing JWT expiration:", err)
		pol := generatePolicy("user", "Deny", request.MethodArn)
		return events.APIGatewayCustomAuthorizerResponse{PrincipalID: pol.PrincipalID, PolicyDocument: pol.PolicyDocument, Context: map[string]interface{}{"message": "UNAUTHORIZED", "error_description": "token invalid", "description": "error parsing jwt token claims"}}, nil
	}
	fmt.Println(exp.UnixMilli(), ",", time.Now().UnixMilli())
	if exp.UnixMilli() < time.Now().UnixMilli() {
		fmt.Println("Token Expired")
		pol := generatePolicy("user", "Deny", request.MethodArn)
		return events.APIGatewayCustomAuthorizerResponse{PrincipalID: pol.PrincipalID, PolicyDocument: pol.PolicyDocument, Context: map[string]interface{}{"message": "UNAUTHORIZED", "error_description": "token invalid", "messageString": "token expired"}}, nil
	}

	if !token.Valid {
		fmt.Println("Invalid token")
		pol := generatePolicy("user", "Deny", request.MethodArn)
		return events.APIGatewayCustomAuthorizerResponse{PrincipalID: pol.PrincipalID, PolicyDocument: pol.PolicyDocument, Context: map[string]interface{}{"message": "UNAUTHORIZED", "error_description": "token invalid", "description": "token invalid"}}, nil
	}

	// Valid token, generate IAM policy
	return generatePolicy("user", "Allow", request.MethodArn), nil
}

// extractToken extracts the JWT token from the Authorization header.
func extractToken(authorizationToken string) string {
	splitToken := strings.Split(authorizationToken, "Bearer ")
	if len(splitToken) != 2 {
		return ""
	}
	return splitToken[1]
}

// generatePolicy generates the IAM policy based on the specified effect and resource.
func generatePolicy(principalID, effect, resource string) events.APIGatewayCustomAuthorizerResponse {
	return events.APIGatewayCustomAuthorizerResponse{
		PrincipalID: principalID,
		PolicyDocument: events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		},
	}
}

func main() {
	lambda.Start(Handler)
}
