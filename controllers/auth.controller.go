package controllers

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"go-chat-redis-pub-sub/config"
	"go-chat-redis-pub-sub/models"
	"go-chat-redis-pub-sub/services"
	"go-chat-redis-pub-sub/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"net/http"
	"strings"
)

type AuthController struct {
	authService services.AuthService
	userService services.UserService
}

func NewAuthController(authService services.AuthService, userService services.UserService) AuthController {
	return AuthController{authService, userService}
}

func (ac *AuthController) SignUpUser(c *gin.Context) {

	var user *models.SignUpInput

	if err := c.ShouldBindJSON(&user); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	newUser, err := ac.authService.SignUpUser(user)

	if err != nil {
		if strings.Contains(err.Error(), "user already exist") {
			c.JSON(http.StatusConflict, gin.H{"status": "error", "message": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"status": "error", "message": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"status": "success", "data": gin.H{"user": newUser}})
}

func (ac *AuthController) SignInUser(c *gin.Context) {
	var cred *models.SignInInput

	if err := c.ShouldBindJSON(&cred); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	user, err := ac.userService.FindUserByUsername(cred.Username)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": "Invalid email or password"})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	if err = utils.VerifyPassword(user.Password, cred.Password); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": "Invalid email or Password"})
		return
	}

	cfg, _ := config.LoadConfig(".")

	// Generate Tokens
	accessToken, err := utils.CreateToken(cfg.AccessTokenExpiresIn, user.ID, cfg.AccessTokenPrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	refreshToken, err := utils.CreateToken(cfg.RefreshTokenExpiresIn, user.ID, cfg.RefreshTokenPrivateKey)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	c.SetCookie("accessToken", accessToken, cfg.AccessTokenMaxAge*60, "/", "localhost", false, true)
	c.SetCookie("refreshToken", refreshToken, cfg.RefreshTokenMaxAge*60, "/", "localhost", false, true)
	c.SetCookie("loggedIn", "true", cfg.AccessTokenMaxAge*60, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{"status": "success", "accessToken": accessToken})

}

func (ac *AuthController) RefreshAccessToken(c *gin.Context) {
	message := "could not refresh access token"

	cookie, err := c.Cookie("refreshToken")

	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": message})
		return
	}

	cfg, _ := config.LoadConfig(".")

	sub, err := utils.ValidateToken(cookie, cfg.RefreshTokenPublicKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	user, err := ac.userService.FindUserById(fmt.Sprint(sub))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": "User belonged to this token no logger exists"})
		return
	}

	accessToken, err := utils.CreateToken(cfg.AccessTokenExpiresIn, user.ID, cfg.AccessTokenPrivateKey)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"status": "fail", "message": err.Error()})
		return
	}

	c.SetCookie("access_token", accessToken, cfg.AccessTokenMaxAge*60, "/", "localhost", false, true)
	c.SetCookie("logged_in", "true", cfg.AccessTokenMaxAge*60, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{"status": "success", "access_token": accessToken})
}

func (ac *AuthController) LogoutUser(c *gin.Context) {
	c.SetCookie("accessToken", "", -1, "/", "localhost", false, true)
	c.SetCookie("refreshToken", "", -1, "/", "localhost", false, true)
	c.SetCookie("loggedIn", "", -1, "/", "localhost", false, true)

	c.JSON(http.StatusOK, gin.H{"status": "success"})
}
