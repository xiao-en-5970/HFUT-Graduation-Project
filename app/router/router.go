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
		postGroup.GET("/:id/comments/:commentId/replies", controller.PostCommentHandlers.ListReplies)
		postGroup.GET("/:id/comments", controller.PostCommentHandlers.ListComments)
		postGroup.POST("/:id/comments", controller.PostCommentHandlers.Create)
		postGroup.POST("/:id/collect", controller.PostCollectHandlers.Add)
		postGroup.DELETE("/:id/collect", controller.PostCollectHandlers.Remove)
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
		questionGroup.GET("/:id/comments/:commentId/replies", controller.QuestionCommentHandlers.ListReplies)
		questionGroup.GET("/:id/comments", controller.QuestionCommentHandlers.ListComments)
		questionGroup.POST("/:id/comments", controller.QuestionCommentHandlers.Create)
		questionGroup.POST("/:id/collect", controller.QuestionCollectHandlers.Add)
		questionGroup.DELETE("/:id/collect", controller.QuestionCollectHandlers.Remove)
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
		answerGroup.GET("/:id/comments/:commentId/replies", controller.AnswerCommentHandlers.ListReplies)
		answerGroup.GET("/:id/comments", controller.AnswerCommentHandlers.ListComments)
		answerGroup.POST("/:id/comments", controller.AnswerCommentHandlers.Create)
		answerGroup.POST("/:id/collect", controller.AnswerCollectHandlers.Add)
		answerGroup.DELETE("/:id/collect", controller.AnswerCollectHandlers.Remove)
	}
	collectGroup := api.Group("/collect")
	{
		collectGroup.POST("/folders", controller.CreateCollectFolder)
		collectGroup.GET("/folders", controller.ListCollectFolders)
		collectGroup.GET("/folders/:id/items", controller.ListCollectItems)
	}
	// OSS 上传、删除（需 JWT）
	api.POST("/oss/*path", controller.OSSUpload)
	api.DELETE("/oss/*path", controller.OSSDelete)
}
