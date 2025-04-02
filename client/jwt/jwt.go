package jwt

import (
	"fmt"
	"os"
)

var jwtSecret []byte

func init() {
	if secret := os.Getenv("JWT_SECRET"); secret != "" {
		jwtSecret = []byte(secret)
	} else {
		fmt.Println("JWT_SECRET environment variable not set")
		os.Exit(1)
	}
}
