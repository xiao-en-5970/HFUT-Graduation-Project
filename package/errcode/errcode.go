package errcode

// 通用错误码 (1000-1999)
const (
	Success             = 200 // 成功
	InvalidParams       = 400 // 参数错误
	Unauthorized        = 401 // 未认证
	Forbidden           = 403 // 无权限
	NotFound            = 404 // 资源不存在
	InternalServerError = 500 // 服务器内部错误
)

// 用户模块错误码 (2000-2099)
const (
	ErrUserNotFound      = 2001 // 用户不存在
	ErrUserAlreadyExists = 2002 // 用户已存在
	ErrUserPasswordWrong = 2003 // 密码错误
	ErrUserDisabled      = 2004 // 用户已禁用
	ErrUserNoPermission  = 2005 // 用户无权限
)

// 文章模块错误码 (2100-2199)
const (
	ErrArticleNotFound     = 2101 // 文章不存在
	ErrArticleNoPermission = 2102 // 无权限操作文章
	ErrArticleCreateFailed = 2103 // 文章创建失败
	ErrArticleUpdateFailed = 2104 // 文章更新失败
	ErrArticleDeleteFailed = 2105 // 文章删除失败
)

// 评论模块错误码 (2200-2299)
const (
	ErrCommentNotFound     = 2201 // 评论不存在
	ErrCommentNoPermission = 2202 // 无权限操作评论
	ErrCommentCreateFailed = 2203 // 评论创建失败
)

// 点赞模块错误码 (2300-2399)
const (
	ErrLikeNotFound      = 2301 // 点赞不存在
	ErrLikeAlreadyExists = 2302 // 已点赞
)

// 商品模块错误码 (2400-2499)
const (
	ErrGoodNotFound     = 2401 // 商品不存在
	ErrGoodNoPermission = 2402 // 无权限操作商品
	ErrGoodCreateFailed = 2403 // 商品创建失败
	ErrGoodUpdateFailed = 2404 // 商品更新失败
	ErrGoodDeleteFailed = 2405 // 商品删除失败
	ErrGoodOutOfStock   = 2406 // 商品库存不足
	ErrGoodNotOnSale    = 2407 // 商品不在售
)

// 收藏模块错误码 (2500-2599)
const (
	ErrCollectNotFound      = 2501 // 收藏不存在
	ErrCollectAlreadyExists = 2502 // 已收藏
	ErrCollectNoPermission  = 2503 // 无权限操作收藏
)

// 关注模块错误码 (2600-2699)
const (
	ErrFollowNotFound      = 2601 // 关注关系不存在
	ErrFollowAlreadyExists = 2602 // 已关注
	ErrFollowSelf          = 2603 // 不能关注自己
	ErrFollowNoPermission  = 2604 // 无权限操作关注
)

// 订单模块错误码 (2700-2799)
const (
	ErrOrderNotFound     = 2701 // 订单不存在
	ErrOrderNoPermission = 2702 // 无权限操作订单
	ErrOrderCreateFailed = 2703 // 订单创建失败
	ErrOrderUpdateFailed = 2704 // 订单更新失败
)

// 学校模块错误码 (2800-2899)
const (
	ErrSchoolNotFound     = 2801 // 学校不存在
	ErrSchoolCreateFailed = 2802 // 学校创建失败
)

// 标签模块错误码 (2900-2999)
const (
	ErrTagNotFound     = 2901 // 标签不存在
	ErrTagCreateFailed = 2902 // 标签创建失败
)

// 错误码对应的消息
var errMsgMap = map[int]string{
	// 通用错误
	Success:             "成功",
	InvalidParams:       "参数错误",
	Unauthorized:        "未认证",
	Forbidden:           "无权限",
	NotFound:            "资源不存在",
	InternalServerError: "服务器内部错误",

	// 用户模块
	ErrUserNotFound:      "用户不存在",
	ErrUserAlreadyExists: "用户已存在",
	ErrUserPasswordWrong: "密码错误",
	ErrUserDisabled:      "用户已禁用",
	ErrUserNoPermission:  "用户无权限",

	// 文章模块
	ErrArticleNotFound:     "文章不存在",
	ErrArticleNoPermission: "无权限操作文章",
	ErrArticleCreateFailed: "文章创建失败",
	ErrArticleUpdateFailed: "文章更新失败",
	ErrArticleDeleteFailed: "文章删除失败",

	// 评论模块
	ErrCommentNotFound:     "评论不存在",
	ErrCommentNoPermission: "无权限操作评论",
	ErrCommentCreateFailed: "评论创建失败",

	// 点赞模块
	ErrLikeNotFound:      "点赞不存在",
	ErrLikeAlreadyExists: "已点赞",

	// 商品模块
	ErrGoodNotFound:     "商品不存在",
	ErrGoodNoPermission: "无权限操作商品",
	ErrGoodCreateFailed: "商品创建失败",
	ErrGoodUpdateFailed: "商品更新失败",
	ErrGoodDeleteFailed: "商品删除失败",
	ErrGoodOutOfStock:   "商品库存不足",
	ErrGoodNotOnSale:    "商品不在售",

	// 收藏模块
	ErrCollectNotFound:      "收藏不存在",
	ErrCollectAlreadyExists: "已收藏，不能重复收藏",
	ErrCollectNoPermission:  "无权限删除此收藏",

	// 关注模块
	ErrFollowNotFound:      "关注关系不存在",
	ErrFollowAlreadyExists: "已关注，不能重复关注",
	ErrFollowSelf:          "不能关注自己",
	ErrFollowNoPermission:  "未关注该用户",

	// 订单模块
	ErrOrderNotFound:     "订单不存在",
	ErrOrderNoPermission: "无权限操作订单",
	ErrOrderCreateFailed: "订单创建失败",
	ErrOrderUpdateFailed: "订单更新失败",

	// 学校模块
	ErrSchoolNotFound:     "学校不存在",
	ErrSchoolCreateFailed: "学校创建失败",

	// 标签模块
	ErrTagNotFound:     "标签不存在",
	ErrTagCreateFailed: "标签创建失败",
}

// GetMsg 根据错误码获取错误消息
func GetMsg(code int) string {
	if msg, ok := errMsgMap[code]; ok {
		return msg
	}
	return "未知错误"
}

// IsSuccess 判断是否成功
func IsSuccess(code int) bool {
	return code == Success
}
