// Package service 的 follow.go 实现关注关系业务层——薄壳，主逻辑全在 dao.Follow()。
//
// 这里只做：
//   - 入参基本校验（不允许关注自己 / 关注禁用账号）
//   - 给前端返回稳定的 "follow_count / fans_count + is_following / is_followed_by"
//     一组字段，方便 UI 关注按钮立即更新而不必再 GET 一次 profile
//
// 设计文档：vo/response.UserProfile、dao/follow.go。
package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/oss"
)

// FollowResult 关注 / 取关后给前端返回的"对象最新状态"快照。
//
// 前端拿到后直接更新本地状态，不需要再 GET /user/:id：
//
//	IsFollowing  操作完之后 viewer 是否仍关注 target
//	FollowCount  target.follow_count（target 关注了多少人——很少改，但一致性好）
//	FansCount    target.fans_count （target 的粉丝数——最常变的字段）
type FollowResult struct {
	IsFollowing bool `json:"is_following"`
	FansCount   int  `json:"fans_count"`
	FollowCount int  `json:"follow_count"`
}

// Follow viewerID 关注 targetID。已关注幂等不报错；自己关自己 / 关禁用用户 → error。
func Follow(ctx context.Context, viewerID, targetID uint) (*FollowResult, error) {
	if err := validateFollowTarget(ctx, viewerID, targetID); err != nil {
		return nil, err
	}
	if _, err := dao.Follow().Follow(ctx, viewerID, targetID); err != nil {
		return nil, err
	}
	return readFollowResult(ctx, viewerID, targetID)
}

// Unfollow viewerID 取消关注 targetID。没关注幂等不报错。
func Unfollow(ctx context.Context, viewerID, targetID uint) (*FollowResult, error) {
	if err := validateFollowTarget(ctx, viewerID, targetID); err != nil {
		return nil, err
	}
	if _, err := dao.Follow().Unfollow(ctx, viewerID, targetID); err != nil {
		return nil, err
	}
	return readFollowResult(ctx, viewerID, targetID)
}

// FollowedUserBrief 关注列表 / 粉丝列表的单行——展示头像、昵称、bio、互关关系。
type FollowedUserBrief struct {
	ID           uint   `json:"id"`
	Username     string `json:"username"`
	Nickname     string `json:"nickname,omitempty"`
	Avatar       string `json:"avatar"`
	Bio          string `json:"bio,omitempty"`
	AccountType  int16  `json:"account_type"`
	IsFollowing  bool   `json:"is_following"` // viewer 是否关注此人（用于"互相关注"提示）
	IsFollowedBy bool   `json:"is_followed_by"`
}

// ListFollowingResp / ListFollowersResp 用同一个结构——分页常量字段。
type FollowListResp struct {
	List     []FollowedUserBrief `json:"list"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
}

// ListFollowing 列出 userID 关注的人；viewerID 用来计算 is_following / is_followed_by（互关展示）。
func ListFollowing(ctx context.Context, userID, viewerID uint, page, pageSize int) (*FollowListResp, error) {
	ids, total, err := dao.Follow().ListFollowing(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	list, err := buildFollowedUserList(ctx, ids, viewerID)
	if err != nil {
		return nil, err
	}
	return &FollowListResp{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

// ListFollowers 列出 userID 的粉丝；viewerID 用来计算 is_following / is_followed_by。
func ListFollowers(ctx context.Context, userID, viewerID uint, page, pageSize int) (*FollowListResp, error) {
	ids, total, err := dao.Follow().ListFollowers(ctx, userID, page, pageSize)
	if err != nil {
		return nil, err
	}
	list, err := buildFollowedUserList(ctx, ids, viewerID)
	if err != nil {
		return nil, err
	}
	return &FollowListResp{List: list, Total: total, Page: page, PageSize: pageSize}, nil
}

// validateFollowTarget 拦截无效场景：自己关自己、target 不存在 / 被禁用。
func validateFollowTarget(ctx context.Context, viewerID, targetID uint) error {
	if viewerID == 0 {
		return ErrInvalidCaller
	}
	if targetID == 0 {
		return errors.New("target_id 不能为空")
	}
	if viewerID == targetID {
		return dao.ErrSelfFollow
	}
	target, err := dao.User().GetByID(ctx, targetID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if target.Status != constant.StatusValid {
		return ErrUserNotFound
	}
	return nil
}

// readFollowResult 关注 / 取关之后再 GET 一次 target 拉计数，组装最终回包。
func readFollowResult(ctx context.Context, viewerID, targetID uint) (*FollowResult, error) {
	target, err := dao.User().GetByID(ctx, targetID)
	if err != nil {
		return nil, err
	}
	isF := false
	if ok, err := dao.Follow().IsFollowing(ctx, viewerID, targetID); err == nil {
		isF = ok
	}
	return &FollowResult{
		IsFollowing: isF,
		FansCount:   target.FansCount,
		FollowCount: target.FollowCount,
	}, nil
}

// buildFollowedUserList 给一组 uid 批量拉用户信息 + 互关状态——一次 SQL 拉用户，一次 SQL 拉
// "viewer→这批 uid" 关注 + 一次 SQL 拉"这批 uid→viewer" 关注。性能 O(1) SQL。
//
// 返回顺序跟入参 ids 一致（用 map 还原），方便分页"按 follow.id DESC" 的序保留。
func buildFollowedUserList(ctx context.Context, ids []uint, viewerID uint) ([]FollowedUserBrief, error) {
	if len(ids) == 0 {
		return []FollowedUserBrief{}, nil
	}
	users, err := dao.User().GetByIDsIfValid(ctx, ids)
	if err != nil {
		return nil, err
	}
	follMap, _ := dao.Follow().BulkIsFollowing(ctx, viewerID, ids)
	// 反向：这批 uid 中哪些关注了 viewer
	fansMap := make(map[uint]bool)
	if viewerID != 0 {
		for _, uid := range ids {
			if ok, err := dao.Follow().IsFollowing(ctx, uid, viewerID); err == nil && ok {
				fansMap[uid] = true
			}
		}
	}
	out := make([]FollowedUserBrief, 0, len(ids))
	for _, uid := range ids {
		u := users[uid]
		if u == nil {
			continue
		}
		brief := buildFollowedUserBrief(u)
		brief.IsFollowing = follMap[uid]
		brief.IsFollowedBy = fansMap[uid]
		out = append(out, brief)
	}
	return out, nil
}

func buildFollowedUserBrief(u *model.User) FollowedUserBrief {
	return FollowedUserBrief{
		ID:          u.ID,
		Username:    u.Username,
		Nickname:    u.DisplayName(),
		Avatar:      oss.ToFullURL(u.DisplayAvatarPath()),
		Bio:         u.Bio,
		AccountType: u.AccountType,
	}
}

// BuildFollowedUserBrief 给外部 controller 复用（如个人主页快速预览）。
func BuildFollowedUserBrief(u *model.User) FollowedUserBrief {
	if u == nil {
		return FollowedUserBrief{}
	}
	return buildFollowedUserBrief(u)
}

// ListResponseUserProfile 已挪走——这里只放关注/列表，profile 走 service.User().GetProfile。
var (
	_ = response.UserProfile{}
)
