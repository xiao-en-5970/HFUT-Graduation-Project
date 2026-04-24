package constant

const (
	StatusValid   = 1
	StatusInvalid = 2
	StatusDraft   = 3 // 草稿，仅创建元信息，用于先获取 ID 再做 OSS 上传
)

// ArticleType 文章类型
const (
	ArticleTypeNormal   = 1 // 普通文章
	ArticleTypeQuestion = 2 // 提问
	ArticleTypeAnswer   = 3 // 回答
)

// ExtTypeArticleGood 关联类型（用于 comments, collect_item, likes 等）
// 1:帖子 2:提问 3:回答 4:商品 5:评论（likes 专用）
const (
	ExtTypePost     = 1 // 帖子
	ExtTypeQuestion = 2 // 提问
	ExtTypeAnswer   = 3 // 回答
	ExtTypeGoods    = 4 // 商品
	ExtTypeComment  = 5 // 评论（仅 likes 表使用）
)

// CommentType 评论类型
const (
	CommentTypeTop   = 1 // 顶层评论
	CommentTypeReply = 2 // 评论回复
)

// UserRole 用户角色
const (
	RoleUser      = 1 // 普通用户
	RoleAdmin     = 2 // 管理员
	RoleSuper     = 3 // 超级管理员
	RoleAnonymous = 4 // 匿名用户
)

// BehaviorAction 用户行为类型（user_behaviors.action）
const (
	BehaviorView      = 1 // 浏览 / 点击详情
	BehaviorLike      = 2 // 点赞
	BehaviorUnlike    = 3 // 取消点赞
	BehaviorCollect   = 4 // 收藏
	BehaviorUncollect = 5 // 取消收藏
	BehaviorComment   = 6 // 评论
	BehaviorSearch    = 7 // 搜索（关键词）
)
