// Package internal imports dependencies to ensure they are retained in go.mod
package internal

import (
	_ "github.com/gin-gonic/gin"
	_ "github.com/golang-jwt/jwt/v5"
	_ "github.com/leanovate/gopter"
	_ "golang.org/x/crypto/bcrypt"
	_ "golang.org/x/time/rate"
)
