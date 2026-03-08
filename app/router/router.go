package router

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/controller"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// SetupRouter 设置路由，接收 gin.Engine 作为参数
func SetupRouter(engine *gin.Engine) {
	// 管理平台前端静态页（/admin 及子路径）
	// 注：Static 通配符会与 /admin/login 冲突，登录页请直接访问 /admin/login.html
	engine.GET("/admin", func(c *gin.Context) { c.Redirect(302, "/admin/") })
	engine.Static("/admin", "package/web/admin")

	// API 路由组
	api := engine.Group("/api/v1")
	// 应用 zap 日志中间件到所有 API 路由
	api.Use(middleware.ZapLogger())
	PublicRouter(api)
	PrivateRouter(api)
}

func PublicRouter(api *gin.RouterGroup) {
	// 管理员登录（公开，无需 JWT）
	api.POST("/admin/login", controller.AdminLogin)

	userGroup := api.Group("/user")
	{
		userGroup.POST("/login", controller.UserLogin)
		userGroup.POST("/school-login", controller.UserSchoolLogin) // 学校端登录，仅需账号密码
		userGroup.POST("/register", controller.UserRegister)
	}
	// OSS 文件访问（公开，前端可直接用 URL 展示图片等）
	api.GET("/oss/*path", controller.OSSGet)
}

func PrivateRouter(api *gin.RouterGroup) {
	api.Use(middleware.JWTAuth())
	// 学校列表与验证码（绑定学校时用）
	api.GET("/schools", controller.SchoolListForBind)
	api.GET("/schools/:id/captcha", controller.SchoolCaptcha)
	userGroup := api.Group("/user")
	{
		userGroup.GET("/info", controller.UserInfo)
		userGroup.GET("/logout", controller.UserLogout)
		userGroup.GET("/:id/posts", controller.UserListPosts)
		userGroup.GET("/:id/questions", controller.UserListQuestions)
		userGroup.GET("/:id/answers", controller.UserListAnswers)
		userGroup.GET("/:id", controller.UserProfile)
		userGroup.POST("/update", controller.UserUpdate)
		userGroup.POST("/bind/school", controller.UserBindSchool)
		userGroup.POST("/avatar", controller.UserUploadAvatar)
		userGroup.POST("/background", controller.UserUploadBackground)
	}
	// 帖子（type=1）、提问（type=2）、回答（type=3），三类接口数据隔离+学校隔离
	api.Use(middleware.LoadUserSchool())
	// 聚合搜索：帖子+提问+回答，支持筛选与排序
	api.GET("/search/articles", controller.SearchArticles)
	postGroup := api.Group("/post")
	{
		postGroup.GET("/drafts", controller.PostHandlers.ListDrafts)
		postGroup.GET("", controller.PostHandlers.List)
		postGroup.GET("/search", controller.PostHandlers.Search)
		postGroup.POST("", controller.PostHandlers.Create)
		postGroup.GET("/:id", controller.PostHandlers.Get)
		postGroup.PUT("/:id", controller.PostHandlers.Update)
		postGroup.POST("/:id/images", controller.PostHandlers.UploadImages)
		postGroup.POST("/:id/publish", controller.PostHandlers.Publish)
		postGroup.DELETE("/:id", controller.PostHandlers.Delete)
	}
	questionGroup := api.Group("/question")
	{
		questionGroup.GET("/drafts", controller.QuestionHandlers.ListDrafts)
		questionGroup.GET("", controller.QuestionHandlers.List)
		questionGroup.GET("/search", controller.QuestionHandlers.Search)
		questionGroup.POST("", controller.QuestionHandlers.Create)
		questionGroup.GET("/:id/answers", controller.QuestionListAnswers) // 须在 /:id 之前
		questionGroup.GET("/:id", controller.QuestionHandlers.Get)
		questionGroup.PUT("/:id", controller.QuestionHandlers.Update)
		questionGroup.POST("/:id/images", controller.QuestionHandlers.UploadImages)
		questionGroup.POST("/:id/publish", controller.QuestionHandlers.Publish)
		questionGroup.DELETE("/:id", controller.QuestionHandlers.Delete)
	}
	answerGroup := api.Group("/answer")
	{
		answerGroup.GET("/drafts", controller.AnswerHandlers.ListDrafts)
		answerGroup.GET("", controller.AnswerHandlers.List)
		answerGroup.GET("/search", controller.AnswerHandlers.Search)
		answerGroup.POST("", controller.AnswerHandlers.Create)
		answerGroup.GET("/:id", controller.AnswerHandlers.Get)
		answerGroup.PUT("/:id", controller.AnswerHandlers.Update)
		answerGroup.POST("/:id/images", controller.AnswerHandlers.UploadImages)
		answerGroup.POST("/:id/publish", controller.AnswerHandlers.Publish)
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

	// 管理员接口：需 JWT + 管理员权限(role>=2)
	adminGroup := api.Group("/admin")
	adminGroup.Use(middleware.AdminAuth())
	{
		// 用户管理
		adminGroup.GET("/users", controller.AdminUserList)
		adminGroup.POST("/users", controller.AdminUserCreate)
		adminGroup.PUT("/users/:id", controller.AdminUserUpdate)
		adminGroup.DELETE("/users/:id", controller.AdminUserDisable)
		adminGroup.POST("/users/:id/restore", controller.AdminUserRestore)
		adminGroup.PUT("/users/:id/role", controller.AdminUserUpdateRole)
		adminGroup.PUT("/users/:id/status", controller.AdminUserUpdateStatus)

		// 帖子管理
		adminGroup.GET("/posts", func(c *gin.Context) { controller.AdminArticleList(c, constant.ArticleTypeNormal) })
		adminGroup.POST("/posts", func(c *gin.Context) { controller.AdminArticleCreate(c, constant.ArticleTypeNormal) })
		adminGroup.PUT("/posts/:id", func(c *gin.Context) { controller.AdminArticleUpdate(c, constant.ArticleTypeNormal) })
		adminGroup.DELETE("/posts/:id", controller.AdminPostDisable)
		adminGroup.POST("/posts/:id/restore", controller.AdminPostRestore)
		// 提问管理
		adminGroup.GET("/questions", func(c *gin.Context) { controller.AdminArticleList(c, constant.ArticleTypeQuestion) })
		adminGroup.POST("/questions", func(c *gin.Context) { controller.AdminArticleCreate(c, constant.ArticleTypeQuestion) })
		adminGroup.PUT("/questions/:id", func(c *gin.Context) { controller.AdminArticleUpdate(c, constant.ArticleTypeQuestion) })
		adminGroup.DELETE("/questions/:id", controller.AdminQuestionDisable)
		adminGroup.POST("/questions/:id/restore", controller.AdminQuestionRestore)
		// 回答管理
		adminGroup.GET("/answers", func(c *gin.Context) { controller.AdminArticleList(c, constant.ArticleTypeAnswer) })
		adminGroup.POST("/answers", func(c *gin.Context) { controller.AdminArticleCreate(c, constant.ArticleTypeAnswer) })
		adminGroup.PUT("/answers/:id", func(c *gin.Context) { controller.AdminArticleUpdate(c, constant.ArticleTypeAnswer) })
		adminGroup.DELETE("/answers/:id", controller.AdminAnswerDisable)
		adminGroup.POST("/answers/:id/restore", controller.AdminAnswerRestore)

		// 学校管理
		adminGroup.GET("/schools", controller.AdminSchoolList)
		adminGroup.POST("/schools", controller.AdminSchoolCreate)
		adminGroup.PUT("/schools/:id", controller.AdminSchoolUpdate)
		adminGroup.DELETE("/schools/:id", controller.AdminSchoolDisable)
		adminGroup.POST("/schools/:id/restore", controller.AdminSchoolRestore)
	}
}
