package constant

const (
	StatusValid   = 1
	StatusInvalid = 2
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
