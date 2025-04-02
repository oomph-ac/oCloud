package jwt

import "github.com/golang-jwt/jwt/v5"

func Validate(tokenString string) (*jwt.Token, bool) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})
	return token, err == nil && token.Valid
}
