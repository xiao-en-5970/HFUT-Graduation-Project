package controller

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/schools"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

func UserRegister(ctx *gin.Context) {
	type Register struct {
		Username   string `json:"username"`
		Password   string `json:"password"`
		RePassword string `json:"re_password"`
	}
	var user Register
	if err := ctx.BindJSON(&user); err != nil {
		logger.Error(ctx, "用户注册参数错误", zap.Error(err))
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if user.RePassword != user.Password {
		reply.ReplyErrWithMessage(ctx, "两次密码不一致！")
		return
	}
	userId, err := service.User().Register(ctx, user.Username, user.Password)
	if err != nil {
		reply.ReplyErr(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"user_id": userId})
}

// UserSchoolLogin 学校端登录：对接学校 CAS，需验证码的学校传 captcha+captcha_token
// POST /api/v1/user/school-login
// Body: { "school_code": "hfut", "username": "学号", "password": "密码", "captcha": "验证码", "captcha_token": "xxx" }
func UserSchoolLogin(ctx *gin.Context) {
	type Req struct {
		SchoolCode   string `json:"school_code" binding:"required"`
		Username     string `json:"username" binding:"required"`
		Password     string `json:"password" binding:"required"`
		Captcha      string `json:"captcha"`
		CaptchaToken string `json:"captcha_token"`
	}
	var req Req
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	opts := &schools.LoginOptions{}
	if school, err := dao.School().GetByCode(ctx.Request.Context(), req.SchoolCode); err == nil && school != nil {
		if school.LoginURL != nil {
			opts.LoginURL = *school.LoginURL
		}
		if school.CaptchaURL != nil {
			opts.CaptchaURL = *school.CaptchaURL
		}
	}
	res, err := schools.Login(ctx.Request.Context(), req.SchoolCode, req.Username, req.Password, req.Captcha, req.CaptchaToken, opts)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	if !res.Success {
		reply.ReplyErrWithMessage(ctx, res.Message)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{
		"student_id": res.StudentID,
		"name":       res.Name,
	})
}

// Login 用户登录
//
// 双 token 模型：
//
//	{
//	  "access_token":  "...",   // 5 分钟（前端放 storage）
//	  "refresh_token": "...",   // 30 天
//	  "expires_in":     300,     // access 剩余秒数
//	  "token_type":    "Bearer"
//	}
//
// 同时通过 Set-Cookie 下发 refresh token：
//
//	HttpOnly + Secure + SameSite=Strict + Path=/api/v1/user/refresh + Max-Age=30d
//
// 浏览器（admin web）应当**只读 access_token**，让 refresh_token 完全留在 cookie 里——
// JS 读不到，XSS 也读不到，最大化保护长期登录态。
//
// React Native 客户端没有原生 cookie，需要继续从 JSON 里取 refresh_token；推荐写到
// iOS Keychain / Android EncryptedSharedPreferences，而不是 AsyncStorage 明文。
func UserLogin(ctx *gin.Context) {
	type Login struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	var user Login
	if err := ctx.BindJSON(&user); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	pair, err := service.User().Login(ctx, user.Username, user.Password)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	setRefreshCookie(ctx, pair.RefreshToken)
	reply.ReplyOKWithMessageAndData(ctx, "登录成功", pair)
}

// UserRefreshToken POST /user/refresh
//
// 优先从 HttpOnly cookie 读 refresh token；没有 cookie 时再 fallback 到 JSON body
// `{refresh_token}`——保证 React Native（无 cookie）也能用同一接口。
//
// 错误码：
//
//	200    刷新成功，返回 TokenPair；同时 Set-Cookie 写一对新的 refresh
//	401    refresh 过期 / 无效 / 类型错（拿 access 来调本接口）
//	500    其它内部错（DB 不可用等）
func UserRefreshToken(ctx *gin.Context) {
	refresh := readRefreshToken(ctx)
	if refresh == "" {
		ctx.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "未提供 refresh token"})
		return
	}
	pair, err := service.User().RefreshToken(refresh)
	if err != nil {
		// refresh 失败：同时清掉旧 cookie 防止后续仍带回来
		clearRefreshCookie(ctx)
		ctx.AbortWithStatusJSON(401, gin.H{"code": 401, "message": "refresh token 无效或已过期：" + err.Error()})
		return
	}
	setRefreshCookie(ctx, pair.RefreshToken)
	reply.ReplyOKWithMessageAndData(ctx, "刷新成功", pair)
}

// UserLogoutCookie POST /user/logout
//
// 公开接口（不需要 access token）。仅做一件事：让浏览器丢弃 HttpOnly refresh cookie。
// admin web 的"退出登录"按钮调用本接口，避免本机仍残留有效 refresh，下次别人复用此机器
// 仍能 refresh 出 access。
//
// 不引入 refresh 黑名单——若想强制让某个用户 30 天前所有 refresh 立即失效，应通过
// rotate JWT_SECRET 或下线该用户（status=invalid）实现，本接口仅清浏览器侧凭证。
func UserLogoutCookie(ctx *gin.Context) {
	clearRefreshCookie(ctx)
	reply.ReplyOK(ctx)
}

// readRefreshToken 从 cookie / JSON body 里取 refresh token；都没有返回 ""。
//
// 顺序很重要：cookie 优先——浏览器一旦把 refresh 放进 HttpOnly cookie，body 就拿不到，
// 客户端代码也就**没必要**再传 refresh 到任何 JS 可见的地方。RN 客户端走 body 兜底。
func readRefreshToken(ctx *gin.Context) string {
	if v, err := ctx.Cookie(refreshCookieName); err == nil && v != "" {
		return v
	}
	type Req struct {
		RefreshToken string `json:"refresh_token"`
	}
	var body Req
	_ = ctx.ShouldBindJSON(&body)
	return body.RefreshToken
}

// refreshCookieName / refreshCookiePath / refreshCookieMaxAge 集中常量。
//
// Path 限定到 /api/v1/user/refresh：浏览器仅在请求 refresh 接口时携带，
// 业务接口不带，最小化 cookie 在网络上的曝光面。
const (
	refreshCookieName   = "refresh_token"
	refreshCookiePath   = "/api/v1/user/refresh"
	refreshCookieMaxAge = 30 * 24 * 60 * 60 // 30d，与 util.RefreshTokenTTL 对齐
)

// setRefreshCookie 把 refresh token 写进 HttpOnly + Secure + SameSite=Strict 的 cookie。
//
// 安全配置说明：
//
//   - HttpOnly：JS 读不到，XSS 偷不走
//   - Secure：仅 HTTPS 下发送；线上 api.xiaoen.xyz 是 HTTPS
//   - SameSite=Strict：第三方站点不会带这条 cookie，CSRF 缓解
//   - Path=/api/v1/user/refresh：只在调 refresh 接口时被自动带
//
// 本地开发若没 HTTPS：浏览器收到 Secure cookie 会**直接丢弃**，这种情况下 refresh
// 走不到，前端可以临时降级为传 body（旧路径仍然支持）。
func setRefreshCookie(ctx *gin.Context, refresh string) {
	ctx.SetSameSite(http.SameSiteStrictMode)
	ctx.SetCookie(refreshCookieName, refresh, refreshCookieMaxAge, refreshCookiePath, "", true /*Secure*/, true /*HttpOnly*/)
}

// clearRefreshCookie 让浏览器立刻丢弃 refresh cookie（设 maxAge=-1）。
func clearRefreshCookie(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteStrictMode)
	ctx.SetCookie(refreshCookieName, "", -1, refreshCookiePath, "", true, true)
}

func UserInfo(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyErrWithMessage(ctx, "用户不存在")
		return
	}
	info, err := service.User().Info(ctx, userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, info)
}

// UserChatUnreadSummary GET /user/chat/unread 订单聊天未读汇总（总条数 + 按订单）
func UserChatUnreadSummary(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	total, byOrder, err := service.Order().ChatUnreadSummary(ctx, userID)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	byOrderJSON := make(map[string]uint, len(byOrder))
	for k, v := range byOrder {
		byOrderJSON[strconv.FormatUint(uint64(k), 10)] = v
	}
	reply.ReplyOKWithData(ctx, gin.H{"total": total, "by_order": byOrderJSON})
}

// UserProfile 获取任意非删用户的公开身份信息（需 JWT）。viewerID 来自 JWT 解出来，
// 用来计算 is_following / is_followed_by / is_self（详见 service.GetProfile）。
func UserProfile(ctx *gin.Context) {
	idStr := ctx.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || id == 0 {
		reply.ReplyErrWithMessage(ctx, "用户ID无效")
		return
	}
	viewerID := middleware.GetUserID(ctx)
	profile, err := service.User().GetProfile(ctx, uint(id), viewerID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, profile)
}

func UserUpdate(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req service.UpdateProfileReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	if err := service.User().UpdateProfile(ctx, userID, req); err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOK(ctx)
}

func UserBindSchool(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	var req service.BindSchoolReq
	if err := ctx.BindJSON(&req); err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	err := service.User().BindSchool(ctx, userID, req)
	if err != nil {
		logger.Error(ctx, "用户绑定学校失败", zap.Error(err))
		msg := err.Error()
		if msg == "" {
			msg = "操作失败，请稍后重试"
		}
		reply.ReplyErrWithMessage(ctx, msg)
		return
	}
	reply.ReplyOK(ctx)
}

func UserLogout(ctx *gin.Context) {
	return
}

func UserUploadAvatar(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	url, err := service.User().UploadAvatar(ctx, userID, file)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"url": url})
}

func UserUploadBackground(ctx *gin.Context) {
	userID := middleware.GetUserID(ctx)
	if userID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	file, err := ctx.FormFile("file")
	if err != nil {
		reply.ReplyInvalidParams(ctx, err)
		return
	}
	url, err := service.User().UploadBackground(ctx, userID, file)
	if err != nil {
		reply.ReplyInternalError(ctx, err)
		return
	}
	reply.ReplyOKWithData(ctx, gin.H{"url": url})
}

// UserListPosts 按用户列出帖子 GET /user/:id/posts
func UserListPosts(ctx *gin.Context) {
	UserListArticlesByType(ctx, constant.ArticleTypeNormal)
}

// UserListQuestions 按用户列出提问 GET /user/:id/questions
func UserListQuestions(ctx *gin.Context) {
	UserListArticlesByType(ctx, constant.ArticleTypeQuestion)
}

// UserListAnswers 按用户列出回答 GET /user/:id/answers
func UserListAnswers(ctx *gin.Context) {
	UserListArticlesByType(ctx, constant.ArticleTypeAnswer)
}

// UserListArticlesByType 按用户分页列出指定类型文章，自己看自己含私密，看别人仅公开
func UserListArticlesByType(ctx *gin.Context, articleType int) {
	viewerID := middleware.GetUserID(ctx)
	if viewerID == 0 {
		reply.ReplyUnauthorized(ctx)
		return
	}
	idStr := ctx.Param("id")
	targetID, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil || targetID == 0 {
		reply.ReplyErrWithMessage(ctx, "用户ID无效")
		return
	}
	schoolID := middleware.GetSchoolID(ctx)
	page, _ := strconv.Atoi(ctx.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(ctx.DefaultQuery("pageSize", "20"))
	sort := ""
	if ctx.Query("sort") == dao.SortUpdatedAt {
		sort = dao.SortUpdatedAt
	}
	list, total, err := service.Article().ListByUser(ctx, uint(targetID), viewerID, schoolID, articleType, page, pageSize, sort)
	if err != nil {
		if errors.Is(err, errno.ErrArticleNotFoundOrNoPermission) {
			reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)
			return
		}
		reply.ReplyInternalError(ctx, err)
		return
	}
	for _, a := range list {
		a.Images = oss.TransformImageURLs(a.Images)
	}
	reply.ReplyOKWithData(ctx, gin.H{"list": enrichArticlesWithAuthor(ctx, list), "total": total, "page": page, "page_size": pageSize})
}
