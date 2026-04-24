package controller

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
)

// stampArticlesViewedBatch 批量给文章（帖/问/答）列表加「已看过」标记。
// 仅登录用户生效；数据源是 user_behaviors 表（跨设备一致），不依赖 Redis / refresh_token，
// 查询是单条「WHERE ext_id IN (这一页 ID)」，走索引扫描，O(page_size)。
func stampArticlesViewedBatch(ctx context.Context, userID uint, extType int, list []response.ArticleWithAuthor) {
	if userID == 0 || len(list) == 0 || extType <= 0 {
		return
	}
	ids := make([]int, 0, len(list))
	for i := range list {
		ids = append(ids, int(list[i].ID))
	}
	viewed, err := dao.UserBehavior().ViewedInIDs(ctx, userID, extType, ids)
	if err != nil || len(viewed) == 0 {
		return
	}
	set := make(map[int]struct{}, len(viewed))
	for _, id := range viewed {
		set[id] = struct{}{}
	}
	for i := range list {
		if _, ok := set[int(list[i].ID)]; ok {
			list[i].IsViewed = true
		}
	}
}

// stampArticlesViewedBatchMixed 混合类型（帖/问/答）列表按各自 Type 分组批量打「已看过」；
// 用于 SearchArticles 这种聚合接口：一次查询按 (user_id, ext_type) 拆多次，规模都很小。
func stampArticlesViewedBatchMixed(ctx context.Context, userID uint, list []response.ArticleWithAuthor) {
	if userID == 0 || len(list) == 0 {
		return
	}
	byType := map[int][]int{}
	for i := range list {
		t := list[i].Type
		if t <= 0 {
			continue
		}
		byType[t] = append(byType[t], int(list[i].ID))
	}
	seenByType := make(map[int]map[int]struct{}, len(byType))
	for t, ids := range byType {
		viewed, err := dao.UserBehavior().ViewedInIDs(ctx, userID, t, ids)
		if err != nil || len(viewed) == 0 {
			continue
		}
		set := make(map[int]struct{}, len(viewed))
		for _, id := range viewed {
			set[id] = struct{}{}
		}
		seenByType[t] = set
	}
	for i := range list {
		if set, ok := seenByType[list[i].Type]; ok {
			if _, hit := set[int(list[i].ID)]; hit {
				list[i].IsViewed = true
			}
		}
	}
}

// stampAnswersViewedBatch AnswerWithAuthor 切片的「已看过」批量打标
func stampAnswersViewedBatch(ctx context.Context, userID uint, list []response.AnswerWithAuthor) {
	if userID == 0 || len(list) == 0 {
		return
	}
	ids := make([]int, 0, len(list))
	for i := range list {
		ids = append(ids, int(list[i].ID))
	}
	viewed, err := dao.UserBehavior().ViewedInIDs(ctx, userID, constant.ExtTypeAnswer, ids)
	if err != nil || len(viewed) == 0 {
		return
	}
	set := make(map[int]struct{}, len(viewed))
	for _, id := range viewed {
		set[id] = struct{}{}
	}
	for i := range list {
		if _, ok := set[int(list[i].ID)]; ok {
			list[i].IsViewed = true
		}
	}
}

// stampGoodsViewedBatch GoodList 返回的 []map 批量写入 is_viewed
func stampGoodsViewedBatch(ctx context.Context, userID uint, list []map[string]interface{}) {
	if len(list) == 0 {
		return
	}
	for _, m := range list {
		if _, ok := m["is_viewed"]; !ok {
			m["is_viewed"] = false
		}
	}
	if userID == 0 {
		return
	}
	ids := make([]int, 0, len(list))
	for _, m := range list {
		if v, ok := m["id"]; ok {
			switch x := v.(type) {
			case uint:
				ids = append(ids, int(x))
			case int:
				ids = append(ids, x)
			case int64:
				ids = append(ids, int(x))
			case uint32:
				ids = append(ids, int(x))
			}
		}
	}
	viewed, err := dao.UserBehavior().ViewedInIDs(ctx, userID, constant.ExtTypeGoods, ids)
	if err != nil || len(viewed) == 0 {
		return
	}
	set := make(map[int]struct{}, len(viewed))
	for _, id := range viewed {
		set[id] = struct{}{}
	}
	for _, m := range list {
		if v, ok := m["id"]; ok {
			var iid int
			switch x := v.(type) {
			case uint:
				iid = int(x)
			case int:
				iid = x
			case int64:
				iid = int(x)
			case uint32:
				iid = int(x)
			default:
				continue
			}
			if _, hit := set[iid]; hit {
				m["is_viewed"] = true
			}
		}
	}
}

// enrichArticleWithAuthorForViewer 单篇详情：作者 + 当前用户是否已赞/已藏（extType：1帖 2问 3答）
func enrichArticleWithAuthorForViewer(ctx context.Context, userID uint, extType int, a *model.Article) response.ArticleWithAuthor {
	out := enrichArticleWithAuthor(ctx, a)
	if userID == 0 {
		return out
	}
	aid := int(a.ID)
	if ok, err := dao.Like().Exists(ctx, userID, aid, extType); err == nil {
		out.IsLiked = ok
	}
	if ok, err := dao.CollectItem().ExistsByUserExt(ctx, userID, aid, extType); err == nil {
		out.IsCollected = ok
	}
	return out
}

// enrichArticlesWithAuthor 为文章列表填充作者信息
func enrichArticlesWithAuthor(ctx *gin.Context, list []*model.Article) []response.ArticleWithAuthor {
	if len(list) == 0 {
		return nil
	}
	ids := make([]uint, 0, len(list))
	for _, a := range list {
		if a.UserID != nil && *a.UserID > 0 {
			ids = append(ids, uint(*a.UserID))
		}
	}
	userMap, _ := dao.User().GetByIDsIfValid(ctx.Request.Context(), ids)
	result := make([]response.ArticleWithAuthor, len(list))
	for i, a := range list {
		result[i] = response.ArticleWithAuthor{Article: *a}
		if a.UserID != nil {
			if u := userMap[uint(*a.UserID)]; u != nil {
				result[i].Author = &response.AuthorProfile{
					ID:       u.ID,
					Username: u.Username,
					Avatar:   oss.ToFullURL(u.Avatar),
				}
			}
		}
	}
	return result
}

// enrichArticleWithAuthor 为单篇文章填充作者信息
func enrichArticleWithAuthor(ctx context.Context, a *model.Article) response.ArticleWithAuthor {
	out := response.ArticleWithAuthor{Article: *a}
	if a.UserID != nil && *a.UserID > 0 {
		u, err := dao.User().GetByIDIfValid(ctx, uint(*a.UserID))
		if err == nil && u != nil {
			out.Author = &response.AuthorProfile{
				ID:       u.ID,
				Username: u.Username,
				Avatar:   oss.ToFullURL(u.Avatar),
			}
		}
	}
	return out
}

// enrichCommentsWithAuthor 为评论列表填充作者信息、回复目标作者信息、回复数、点赞状态及热门预览回复
func enrichCommentsWithAuthor(ctx *gin.Context, list []*model.Comment) []response.CommentWithAuthor {
	if len(list) == 0 {
		return nil
	}
	viewerID := middleware.GetUserID(ctx)

	userIDs := make([]uint, 0, len(list))
	for _, c := range list {
		if c.UserID != nil && *c.UserID > 0 {
			userIDs = append(userIDs, uint(*c.UserID))
		}
	}

	// 收集回复的目标评论 ID（reply_id 优先，否则 parent_id）
	targetCmtIDSet := make(map[uint]bool)
	for _, c := range list {
		if c.Type == constant.CommentTypeReply {
			if c.ReplyID != nil && *c.ReplyID > 0 {
				targetCmtIDSet[uint(*c.ReplyID)] = true
			} else if c.ParentID != nil && *c.ParentID > 0 {
				targetCmtIDSet[uint(*c.ParentID)] = true
			}
		}
	}
	targetCmtIDs := make([]uint, 0, len(targetCmtIDSet))
	for id := range targetCmtIDSet {
		targetCmtIDs = append(targetCmtIDs, id)
	}

	targetCmtMap := make(map[uint]*model.Comment)
	if len(targetCmtIDs) > 0 {
		targets, _ := dao.Comment().GetByIDs(ctx.Request.Context(), targetCmtIDs)
		for _, t := range targets {
			targetCmtMap[t.ID] = t
			if t.UserID != nil && *t.UserID > 0 {
				userIDs = append(userIDs, uint(*t.UserID))
			}
		}
	}

	// 顶层评论：统计回复数 + 取 top3 热门回复
	var topIDs []uint
	for _, c := range list {
		if c.Type == constant.CommentTypeTop {
			topIDs = append(topIDs, c.ID)
		}
	}
	replyCountMap := make(map[uint]int64)
	topRepliesMap := make(map[uint][]*model.Comment)
	if len(topIDs) > 0 {
		replyCountMap, _ = dao.Comment().CountRepliesByParentIDs(ctx.Request.Context(), topIDs)
		topRepliesMap, _ = dao.Comment().TopRepliesByParentIDs(ctx.Request.Context(), topIDs, 3)
	}

	// 收集 top replies 的 user_id + 它们的 reply target
	var allPreviewReplies []*model.Comment
	for _, replies := range topRepliesMap {
		allPreviewReplies = append(allPreviewReplies, replies...)
		for _, r := range replies {
			if r.UserID != nil && *r.UserID > 0 {
				userIDs = append(userIDs, uint(*r.UserID))
			}
			if r.ReplyID != nil && *r.ReplyID > 0 {
				if !targetCmtIDSet[uint(*r.ReplyID)] {
					targetCmtIDSet[uint(*r.ReplyID)] = true
					targetCmtIDs = append(targetCmtIDs, uint(*r.ReplyID))
				}
			} else if r.ParentID != nil && *r.ParentID > 0 {
				if !targetCmtIDSet[uint(*r.ParentID)] {
					targetCmtIDSet[uint(*r.ParentID)] = true
					targetCmtIDs = append(targetCmtIDs, uint(*r.ParentID))
				}
			}
		}
	}
	// 补充获取 preview replies 引用的目标评论
	if newIDs := targetCmtIDs; len(newIDs) > 0 {
		extras, _ := dao.Comment().GetByIDs(ctx.Request.Context(), newIDs)
		for _, t := range extras {
			if targetCmtMap[t.ID] == nil {
				targetCmtMap[t.ID] = t
				if t.UserID != nil && *t.UserID > 0 {
					userIDs = append(userIDs, uint(*t.UserID))
				}
			}
		}
	}

	userMap, _ := dao.User().GetByIDsIfValid(ctx.Request.Context(), userIDs)

	// 批量检查当前用户对所有评论（含 preview）的点赞状态
	allCommentIDs := make([]uint, 0, len(list)+len(allPreviewReplies))
	for _, c := range list {
		allCommentIDs = append(allCommentIDs, c.ID)
	}
	for _, r := range allPreviewReplies {
		allCommentIDs = append(allCommentIDs, r.ID)
	}
	likedSet := make(map[uint]bool)
	if viewerID > 0 && len(allCommentIDs) > 0 {
		for _, cid := range allCommentIDs {
			if ok, _ := dao.Like().Exists(ctx.Request.Context(), viewerID, int(cid), constant.ExtTypeComment); ok {
				likedSet[cid] = true
			}
		}
	}

	makeAuthor := func(uid int) *response.AuthorProfile {
		if u := userMap[uint(uid)]; u != nil {
			return &response.AuthorProfile{
				ID: u.ID, Username: u.Username, Avatar: oss.ToFullURL(u.Avatar),
			}
		}
		return nil
	}

	replyToAuthor := func(c *model.Comment) *response.AuthorProfile {
		var tid uint
		if c.ReplyID != nil && *c.ReplyID > 0 {
			tid = uint(*c.ReplyID)
		} else if c.ParentID != nil && *c.ParentID > 0 {
			tid = uint(*c.ParentID)
		}
		if tc := targetCmtMap[tid]; tc != nil && tc.UserID != nil {
			return makeAuthor(*tc.UserID)
		}
		return nil
	}

	result := make([]response.CommentWithAuthor, len(list))
	for i, c := range list {
		result[i] = response.CommentWithAuthor{Comment: *c, IsLiked: likedSet[c.ID]}
		if c.UserID != nil {
			result[i].Author = makeAuthor(*c.UserID)
		}
		if c.Type == constant.CommentTypeTop {
			result[i].ReplyCount = replyCountMap[c.ID]
			if previews := topRepliesMap[c.ID]; len(previews) > 0 {
				pr := make([]response.CommentWithAuthor, len(previews))
				for j, r := range previews {
					pr[j] = response.CommentWithAuthor{Comment: *r, IsLiked: likedSet[r.ID]}
					if r.UserID != nil {
						pr[j].Author = makeAuthor(*r.UserID)
					}
					pr[j].ReplyToAuthor = replyToAuthor(r)
				}
				result[i].TopReplies = pr
			}
		}
		if c.Type == constant.CommentTypeReply {
			result[i].ReplyToAuthor = replyToAuthor(c)
		}
	}
	return result
}

// enrichAnswersWithParentQuestion 回答列表附带所属提问标题与正文（社区流展示用）
func enrichAnswersWithParentQuestion(ctx *gin.Context, viewerSchoolID uint, list []*model.Article) []response.AnswerWithAuthor {
	base := enrichArticlesWithAuthor(ctx, list)
	if len(base) == 0 {
		return nil
	}
	seen := make(map[uint]bool)
	var parentIDs []uint
	for _, a := range list {
		if a.ParentID != nil && *a.ParentID > 0 {
			pid := uint(*a.ParentID)
			if !seen[pid] {
				seen[pid] = true
				parentIDs = append(parentIDs, pid)
			}
		}
	}
	parentMap := make(map[uint]*model.Article)
	for _, pid := range parentIDs {
		q, err := dao.Article().GetByIDWithSchoolOrPublicAndType(ctx.Request.Context(), pid, viewerSchoolID, constant.ArticleTypeQuestion)
		if err == nil && q != nil {
			parentMap[pid] = q
		}
	}
	out := make([]response.AnswerWithAuthor, len(base))
	for i := range base {
		out[i].ArticleWithAuthor = base[i]
		if list[i].ParentID != nil {
			pid := uint(*list[i].ParentID)
			if pq, ok := parentMap[pid]; ok {
				out[i].ParentQuestion = &response.ParentQuestionBrief{
					ID:       pq.ID,
					Title:    pq.Title,
					Content:  pq.Content,
					SchoolID: pq.SchoolID,
				}
			}
		}
	}
	return out
}

// enrichAnswerWithParent 单条回答详情附带提问摘要
func enrichAnswerWithParent(ctx *gin.Context, viewerSchoolID uint, art *model.Article) response.AnswerWithAuthor {
	userID := middleware.GetUserID(ctx)
	base := enrichArticleWithAuthorForViewer(ctx.Request.Context(), userID, constant.ExtTypeAnswer, art)
	out := response.AnswerWithAuthor{ArticleWithAuthor: base}
	if art.ParentID != nil && *art.ParentID > 0 {
		pid := uint(*art.ParentID)
		q, err := dao.Article().GetByIDWithSchoolOrPublicAndType(ctx.Request.Context(), pid, viewerSchoolID, constant.ArticleTypeQuestion)
		if err == nil && q != nil {
			out.ParentQuestion = &response.ParentQuestionBrief{
				ID:       q.ID,
				Title:    q.Title,
				Content:  q.Content,
				SchoolID: q.SchoolID,
			}
		}
	}
	return out
}
