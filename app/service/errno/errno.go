// Package errno 定义 service 层可比较的哨兵错误，供业务与 controller 用 errors.Is 判断。
package errno

import "errors"

// 帖子 / 文章
var (
	ErrArticleNotFoundOrNoPermission = errors.New("帖子不存在或无权限")
	ErrSchoolNotBound                = errors.New("请先绑定学校")
	ErrParentQuestionRequired        = errors.New("回答必须指定父提问 parent_id")
	ErrParentQuestionNotFound        = errors.New("父提问不存在或非本校")
	ErrDraftNotFoundOrNoPermission   = errors.New("草稿不存在或无权限")
)

// 商品
var ErrGoodNotFoundOrNoPermission = errors.New("商品不存在或无权限")

// 订单
var (
	ErrOrderGoodNotFound      = errors.New("商品不存在或已下架")
	ErrOrderGoodNotOnSale     = errors.New("商品未上架")
	ErrOrderInsufficientStock = errors.New("库存不足")
	ErrOrderNotFound          = errors.New("订单不存在")
)

// 评论
var (
	ErrCommentArticleNotFound = errors.New("文章不存在或无权限")
	ErrCommentParentNotFound  = errors.New("父评论不存在")
)

// 点赞
var (
	ErrLikeArticleNotFound = errors.New("文章不存在")
	ErrLikeAlreadyLiked    = errors.New("已点赞")
	ErrLikeNotLiked        = errors.New("未点赞")
)

// 收藏
var (
	ErrCollectFolderNotFound   = errors.New("收藏夹不存在")
	ErrCollectFolderNotOwned   = errors.New("无权限操作该收藏夹")
	ErrCollectArticleNotFound  = errors.New("文章不存在")
	ErrCollectAlreadyCollected = errors.New("已收藏")
	ErrCollectNotCollected     = errors.New("未收藏")
)
