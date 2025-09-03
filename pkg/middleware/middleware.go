package middleware

import (
	"os"
	"strings"
	"time"

	"github.com/crm/pkg/state"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

func ClaimIp() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("CurrentIP", c.ClientIP())
		c.Set(state.CurrentUserIP, c.ClientIP())
		c.Next()
	}
}

func Admin() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("admin_key") != os.Getenv("ADMIN_KEY") {
			c.JSON(400, gin.H{"message": "Unauthorized access"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func CheckAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{"error": "Token is required"})
			c.Abort()
			return
		}

		authToken := strings.Split(authHeader, " ")
		if len(authToken) != 2 || authToken[0] != "Bearer" {
			c.JSON(400, gin.H{"error": "Invalid/Malformed auth token"})
			c.Abort()
			return
		}

		myJwt := authToken[1]
		claims := jwt.MapClaims{}
		token, err := jwt.ParseWithClaims(myJwt, claims, func(token *jwt.Token) (interface{}, error) {
			return []byte(os.Getenv("SECRET")), nil
		})

		if err != nil {
			c.JSON(401, gin.H{"error": err.Error()})
			c.Abort()
			return
		}

		if !token.Valid {
			c.JSON(401, gin.H{"error": "Token is not valid"})
			c.Abort()
			return
		}

		if exp, ok := claims["exp"].(float64); !ok || float64(time.Now().Unix()) > exp {
			c.JSON(401, gin.H{"error": "Token expired"})
			c.Abort()
			return
		}

		// Set user ID to context
		if userID, ok := claims["id"].(float64); ok {
			c.Set(state.CurrentUserId, uint(userID))
		}

		c.Next()
	}
}
