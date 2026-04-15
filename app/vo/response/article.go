package response

import (
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
)

// ArticleWithAuthor 文章详情（含作者信息）
type ArticleWithAuthor struct {
	model.Article
	Author      *AuthorProfile `json:"author,omitempty"`
	IsLiked     bool           `json:"is_liked"`
	IsCollected bool           `json:"is_collected"`
}

// ParentQuestionBrief 回答列表/详情中附带的提问摘要
type ParentQuestionBrief struct {
	ID       uint   `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	SchoolID *int   `json:"school_id,omitempty"`
}

// AnswerWithAuthor 回答（含作者与所属提问摘要，供社区流与详情）
type AnswerWithAuthor struct {
	ArticleWithAuthor
	ParentQuestion *ParentQuestionBrief `json:"parent_question,omitempty"`
}

// CommentWithAuthor 评论（含作者信息）
type CommentWithAuthor struct {
	model.Comment
	Author        *AuthorProfile      `json:"author,omitempty"`
	ReplyToAuthor *AuthorProfile      `json:"reply_to_author,omitempty"`
	ReplyCount    int64               `json:"reply_count"`
	IsLiked       bool                `json:"is_liked"`
	TopReplies    []CommentWithAuthor `json:"top_replies,omitempty"`
}
