package routes

import (
	"github.com/gin-gonic/gin"
	"go-chat-redis-pub-sub/controllers"
	"go-chat-redis-pub-sub/services"
)

type UserRouteController struct {
	userController controllers.UserController
}

func NewRouteUserController(userController controllers.UserController) UserRouteController {
	return UserRouteController{userController}
}

func (uc *UserRouteController) UserRoute(rg *gin.RouterGroup, userService services.UserService) {

	router := rg.Group("users")
	router.GET("/me", uc.userController.GetMe)
}
