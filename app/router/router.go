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
	// OSS 文件访问（公开，前端可直接用 URL 展示图片等）
	api.GET("/oss/*path", controller.OSSGet)
}

func PrivateRouter(api *gin.RouterGroup) {
	api.Use(middleware.JWTAuth())
	userGroup := api.Group("/user")
	{
		userGroup.GET("/info", controller.UserInfo)
		userGroup.GET("/logout", controller.UserLogout)
		userGroup.POST("/update", controller.UserUpdate)
		userGroup.POST("/bind/school", controller.UserBindSchool)
		userGroup.POST("/avatar", controller.UserUploadAvatar)
		userGroup.POST("/background", controller.UserUploadBackground)
	}
	// 帖子（type=1）、提问（type=2）、回答（type=3），三类接口数据隔离+学校隔离
	api.Use(middleware.LoadUserSchool())
	postGroup := api.Group("/post")
	{
		postGroup.GET("", controller.PostHandlers.List)
		postGroup.GET("/search", controller.PostHandlers.Search)
		postGroup.POST("", controller.PostHandlers.Create)
		postGroup.GET("/:id", controller.PostHandlers.Get)
		postGroup.PUT("/:id", controller.PostHandlers.Update)
		postGroup.POST("/:id/images", controller.PostHandlers.UploadImages)
		postGroup.PUT("/:id/images", controller.PostHandlers.UpdateImages)
		postGroup.DELETE("/:id", controller.PostHandlers.Delete)
	}
	questionGroup := api.Group("/question")
	{
		questionGroup.GET("", controller.QuestionHandlers.List)
		questionGroup.GET("/search", controller.QuestionHandlers.Search)
		questionGroup.POST("", controller.QuestionHandlers.Create)
		questionGroup.GET("/:id/answers", controller.QuestionListAnswers) // 须在 /:id 之前
		questionGroup.GET("/:id", controller.QuestionHandlers.Get)
		questionGroup.PUT("/:id", controller.QuestionHandlers.Update)
		questionGroup.POST("/:id/images", controller.QuestionHandlers.UploadImages)
		questionGroup.PUT("/:id/images", controller.QuestionHandlers.UpdateImages)
		questionGroup.DELETE("/:id", controller.QuestionHandlers.Delete)
	}
	answerGroup := api.Group("/answer")
	{
		answerGroup.GET("", controller.AnswerHandlers.List)
		answerGroup.GET("/search", controller.AnswerHandlers.Search)
		answerGroup.POST("", controller.AnswerHandlers.Create)
		answerGroup.GET("/:id", controller.AnswerHandlers.Get)
		answerGroup.PUT("/:id", controller.AnswerHandlers.Update)
		answerGroup.POST("/:id/images", controller.AnswerHandlers.UploadImages)
		answerGroup.PUT("/:id/images", controller.AnswerHandlers.UpdateImages)
		answerGroup.DELETE("/:id", controller.AnswerHandlers.Delete)
	}
	// 共通模块：评论、收藏、点赞，由前端传 extType 区分
	// extType: 1帖子 2提问 3回答 4商品(仅收藏)
	commentGroup := api.Group("/comments")
	{
		commentGroup.GET("/:extType/:id", controller.CommentList)
		commentGroup.POST("/:extType/:id", controller.CommentCreate)
		commentGroup.GET("/:extType/:id/:commentId/replies", controller.CommentListReplies)
	}
	collectGroup := api.Group("/collect")
	{
		collectGroup.POST("/folders", controller.CreateCollectFolder)
		collectGroup.GET("/folders", controller.ListCollectFolders)
		collectGroup.GET("/folders/:id/items", controller.ListCollectItems)
		collectGroup.POST("/:extType/:id", controller.CollectAdd)
		collectGroup.DELETE("/:extType/:id", controller.CollectRemove)
	}
	likeGroup := api.Group("/like")
	{
		likeGroup.POST("/:extType/:id", controller.LikeAdd)
		likeGroup.DELETE("/:extType/:id", controller.LikeRemove)
	}
	// OSS 上传、删除（需 JWT）
	api.POST("/oss/*path", controller.OSSUpload)
	api.DELETE("/oss/*path", controller.OSSDelete)
}
