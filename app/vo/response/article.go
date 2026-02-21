package response

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
)

// ArticleWithAuthor 文章详情（含作者信息）
type ArticleWithAuthor struct {
	model.Article
	Author *AuthorProfile `json:"author,omitempty"`
}

// CommentWithAuthor 评论（含作者信息）
type CommentWithAuthor struct {
	model.Comment
	Author *AuthorProfile `json:"author,omitempty"`
}
