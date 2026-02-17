package router

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/controller"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
)

// SetupRouter 设置路由，接收 gin.Engine 作为参数
func SetupRouter(engine *gin.Engine) {

	// API 路由组
	api := engine.Group("/api/v1")
	// 应用 zap 日志中间件到所有 API 路由
	api.Use(middleware.ZapLogger())
	PublicRouter(api)
	PrivateRouter(api)
}

func PublicRouter(api *gin.RouterGroup) {
	userGroup := api.Group("/user")
	{
		userGroup.POST("/login", controller.UserLogin)
		userGroup.POST("/register", controller.UserRegister)
	}

}

func PrivateRouter(api *gin.RouterGroup) {
	api.Use(middleware.JWTAuth())
	userGroup := api.Group("/user")
	{
		userGroup.GET("/info", controller.UserInfo)
		userGroup.GET("/logout", controller.UserLogout)
		userGroup.POST("/update", controller.UserUpdate)
		userGroup.POST("/bind/school", controller.UserBindSchool)
	}

}
