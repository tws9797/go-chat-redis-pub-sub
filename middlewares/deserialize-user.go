package middlewares

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-chat-redis-pub-sub/config"
	"go-chat-redis-pub-sub/services"
	"go-chat-redis-pub-sub/utils"
	"net/http"
	"strings"
)

func DeserializeUser(userService services.UserService) gin.HandlerFunc {
	return func(c *gin.Context) {
		var accessToken string
		cookie, err := c.Cookie("accessToken")

		authorizationHeader := c.Request.Header.Get("Authorization")
		fields := strings.Fields(authorizationHeader)

		if len(fields) != 0 && fields[0] == "Bearer" {
			accessToken = fields[1]
		} else if err == nil {
			accessToken = cookie
		}

		if accessToken == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "You are not logged in"})
			return
		}

		cfg, _ := config.LoadConfig(".")
		sub, err := utils.ValidateToken(accessToken, cfg.AccessTokenPublicKey)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": err.Error()})
			return
		}

		user, err := userService.FindUserById(fmt.Sprint(sub))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"status": "fail", "message": "The user belonging to this token no logger exists"})
			return
		}

		c.Set("currentUser", user)
		c.Next()
	}
}
