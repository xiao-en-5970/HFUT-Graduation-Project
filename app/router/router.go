package router

import (
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/controller"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
)

// SetupRouter 设置路由，接收 gin.Engine 作为参数
func SetupRouter(engine *gin.Engine) {

	// 创建控制器实例
	userController := controller.NewUserController()
	articleController := controller.NewArticleController()
	commentController := controller.NewCommentController()
	likeController := controller.NewLikeController()
	goodController := controller.NewGoodController()
	tagController := controller.NewTagController()
	schoolController := controller.NewSchoolController()

	// API 路由组
	api := engine.Group("/api/v1")
	// 应用 zap 日志中间件到所有 API 路由
	api.Use(middleware.ZapLogger())

	{
		// 用户相关路由（注册和登录不需要认证）
		users := api.Group("/users")
		{
			users.POST("/register", userController.Register) // 用户注册
			users.POST("/login", userController.Login)       // 用户登录

			// 需要认证的路由
			usersAuth := users.Group("")
			usersAuth.Use(middleware.JWTAuth())
			{
				usersAuth.GET("/info", userController.Info)   // 获取当前登录用户信息
				usersAuth.GET("/:id", userController.GetByID) // 获取用户信息
				usersAuth.PUT("/:id", userController.Update)  // 更新用户信息
				usersAuth.GET("", userController.List)        // 获取用户列表
			}
		}

		// 文章相关路由（需要认证）
		articles := api.Group("/articles")
		articles.Use(middleware.JWTAuth())
		{
			articles.POST("", articleController.Create)       // 创建文章
			articles.GET("/:id", articleController.GetByID)   // 获取文章详情
			articles.PUT("/:id", articleController.Update)    // 更新文章
			articles.DELETE("/:id", articleController.Delete) // 删除文章
			articles.GET("", articleController.List)          // 获取文章列表
		}

		// 评论相关路由（需要认证）
		comments := api.Group("/comments")
		comments.Use(middleware.JWTAuth())
		{
			comments.POST("", commentController.Create)       // 创建评论
			comments.GET("", commentController.List)          // 获取评论列表
			comments.DELETE("/:id", commentController.Delete) // 删除评论
		}

		// 点赞相关路由（需要认证）
		likes := api.Group("/likes")
		likes.Use(middleware.JWTAuth())
		{
			likes.POST("/toggle", likeController.ToggleLike) // 切换点赞状态
			likes.GET("/check", likeController.IsLiked)      // 检查是否已点赞
		}

		// 商品相关路由（需要认证）
		goods := api.Group("/goods")
		goods.Use(middleware.JWTAuth())
		{
			goods.POST("", goodController.Create)       // 创建商品
			goods.GET("/:id", goodController.GetByID)   // 获取商品详情
			goods.PUT("/:id", goodController.Update)    // 更新商品
			goods.DELETE("/:id", goodController.Delete) // 删除商品
			goods.GET("", goodController.List)          // 获取商品列表
		}

		// 标签相关路由（需要认证）
		tags := api.Group("/tags")
		tags.Use(middleware.JWTAuth())
		{
			tags.POST("", tagController.Create)  // 创建标签
			tags.GET("", tagController.GetByExt) // 根据关联对象获取标签列表
		}

		// 学校相关路由（需要认证）
		schools := api.Group("/schools")
		schools.Use(middleware.JWTAuth())
		{
			schools.POST("", schoolController.Create)     // 创建学校
			schools.GET("/:id", schoolController.GetByID) // 获取学校详情
			schools.GET("", schoolController.List)        // 获取学校列表
		}
	}
}
