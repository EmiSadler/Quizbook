package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/makersacademy/go-react-acebook-template/api/src/controllers"
	"github.com/makersacademy/go-react-acebook-template/api/src/middleware"
)

func setupUserRoutes(baseRouter *gin.RouterGroup) {
	users := baseRouter.Group("/users")

	users.POST("", controllers.CreateUser)
	users.PUT("", middleware.AuthenticationMiddleware, controllers.UpdateUser)
	users.GET("/me", middleware.AuthenticationMiddleware, controllers.GetCurrentUser)
	users.DELETE("/me", middleware.AuthenticationMiddleware, controllers.DeleteUser)
	users.GET("/:id/likes", middleware.AuthenticationMiddleware, controllers.GetLikedPostsByUserID)
	users.GET("/:id", middleware.AuthenticationMiddleware, controllers.GetUserByID)
}
