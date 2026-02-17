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

// ExtType 关联类型（用于 likes, comments, collects, tags 等）
const (
	ExtTypeArticle = 1 // 文章
	ExtTypeComment = 2 // 评论
	ExtTypeGood    = 3 // 商品
)

// CommentType 评论类型
const (
	CommentTypeTop   = 1 // 顶层评论
	CommentTypeReply = 2 // 评论回复
)
