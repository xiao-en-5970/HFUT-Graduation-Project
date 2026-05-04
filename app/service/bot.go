// Package service 的 bot.go 提供 /api/v1/bot/* 路由组的业务逻辑。
//
// 这些接口只能通过 middleware.BotServiceAuth 校验过的服务（如 QQ-bot）调用。
// 设计文档：QQ-bot 仓库 skill/bot/SKILL.md。
package service

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// ErrBotGroupNoSchool 群没有配学校时返回；bot 收到这个错应该完全静默忽略，
// 不要在群里给任何提示——本群压根不在白名单里。
var ErrBotGroupNoSchool = errors.New("bot: 当前 QQ 群没有任何学校配置 qq_groups，请管理员先在 schools 表里关联")

// ErrBotUserNotFound 旗下账号或主账号不存在 / 被禁用。
var ErrBotUserNotFound = errors.New("bot: 用户不存在或已禁用")

// ErrBotGoodNotFound 商品不存在 / 被禁用。
var ErrBotGoodNotFound = errors.New("bot: 商品不存在或已禁用")

// ErrBotGoodNotOwner 调用方不是商品所有者，无权操作。
var ErrBotGoodNotOwner = errors.New("bot: 仅商品所有者或其主账号可操作")

// =============================================================================
// QQ 旗下账号 upsert
// =============================================================================

// BotUpsertQQChildReq 由 QQ-bot 调用：按 (qq_number, group_id) 创建/复用旗下账号。
type BotUpsertQQChildReq struct {
	QQNumber string `json:"qq_number"` // 必填
	GroupID  int64  `json:"group_id"`  // 必填，用于推断 school_id
	Nickname string `json:"nickname"`  // 可选，群名片，会写到 username 之外的展示字段
}

// BotUpsertQQChildResp upsert 结果。Created=true 表示这次新建了一个旗下账号。
type BotUpsertQQChildResp struct {
	UserID   uint   `json:"user_id"`
	Created  bool   `json:"created"`
	SchoolID uint   `json:"school_id"`
	Username string `json:"username"`
	Nickname string `json:"nickname,omitempty"`
}

// BotUpsertQQChild 找到则返回；找不到则按 group_id 反查 school 后创建。
//
// 注意 QQ 旗下账号是"严格 1:1"——一个 qq_number 全局只能有一个有效旗下账号。
// uq_users_qq_number_active 唯一索引兜底防并发重复创建。
func BotUpsertQQChild(ctx context.Context, req BotUpsertQQChildReq) (*BotUpsertQQChildResp, error) {
	if req.QQNumber == "" {
		return nil, errors.New("qq_number 不能为空")
	}
	if req.GroupID == 0 {
		return nil, errors.New("group_id 不能为空")
	}

	// 1) 命中现有旗下账号？
	existing, err := findActiveQQChild(ctx, req.QQNumber)
	if err != nil {
		return nil, fmt.Errorf("查找现有旗下账号失败: %w", err)
	}
	if existing != nil {
		// 旗下账号已存在：可能 group_id 跟当时创建的不同（用户跨群发消息），
		// 不报错——一个 QQ 用户的旗下账号是全局的，群只决定第一次创建时归属哪所学校。
		// 群名片有变化时顺手更新（"实时跟群名片同步"）。
		if req.Nickname != "" && req.Nickname != existing.Username {
			_ = pgsql.DB.WithContext(ctx).Model(existing).
				UpdateColumn("avatar", existing.Avatar) // no-op 占位，保留扩展点
			// TODO: 若以后加 nickname 列，在这里 update
		}
		return &BotUpsertQQChildResp{
			UserID:   existing.ID,
			Created:  false,
			SchoolID: existing.SchoolID,
			Username: existing.Username,
		}, nil
	}

	// 2) 没找到——按 group_id 反查归属学校
	schoolID, err := findSchoolIDByQQGroup(ctx, req.GroupID)
	if err != nil {
		return nil, fmt.Errorf("查 group_id→school 失败: %w", err)
	}
	if schoolID == 0 {
		return nil, ErrBotGroupNoSchool
	}

	// 3) 创建新旗下账号
	username := fmt.Sprintf("qq%s", req.QQNumber)
	qqNum := req.QQNumber
	newU := &model.User{
		Username:    username,
		Password:    "", // 不可登录
		SchoolID:    schoolID,
		AccountType: model.AccountTypeQQChild,
		QQNumber:    &qqNum,
		Status:      constant.StatusValid,
		Role:        constant.RoleUser,
	}
	if _, err := dao.User().Create(ctx, newU); err != nil {
		// 并发冲突：另一个请求已经抢先创建过同一 qq_number 的旗下账号——重新查一次返回它
		again, err2 := findActiveQQChild(ctx, req.QQNumber)
		if err2 == nil && again != nil {
			return &BotUpsertQQChildResp{
				UserID:   again.ID,
				Created:  false,
				SchoolID: again.SchoolID,
				Username: again.Username,
			}, nil
		}
		return nil, fmt.Errorf("创建旗下账号失败: %w", err)
	}
	return &BotUpsertQQChildResp{
		UserID:   newU.ID,
		Created:  true,
		SchoolID: newU.SchoolID,
		Username: newU.Username,
		Nickname: req.Nickname,
	}, nil
}

// findActiveQQChild 按 qq_number 找一条 status=valid 的旗下账号；没找到返回 (nil, nil)。
func findActiveQQChild(ctx context.Context, qqNumber string) (*model.User, error) {
	var u model.User
	err := pgsql.DB.WithContext(ctx).
		Where("qq_number = ? AND account_type = ? AND status = ?",
			qqNumber, model.AccountTypeQQChild, constant.StatusValid).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

// findSchoolIDByQQGroup 按 group_id 在 schools.qq_groups 数组里反查归属学校。
//
// 同一 group_id 出现在多所学校的 qq_groups 里时取 id 最小的（理论上不该出现，
// 但 schema 不强制；遇到时 log 一下 warning 等管理员清理）。
//
// 找不到返回 (0, nil)，调用方按 ErrBotGroupNoSchool 处理。
func findSchoolIDByQQGroup(ctx context.Context, groupID int64) (uint, error) {
	var schools []model.School
	// pgsql 的 ARRAY 包含查询用 ANY()
	err := pgsql.DB.WithContext(ctx).
		Where("? = ANY(qq_groups) AND status = ?", groupID, constant.StatusValid).
		Order("id ASC").
		Limit(1).
		Find(&schools).Error
	if err != nil {
		return 0, err
	}
	if len(schools) == 0 {
		return 0, nil
	}
	return schools[0].ID, nil
}

// =============================================================================
// 商品上架 / 下架 / 在售列表
// =============================================================================

// BotPublishGoodReq bot 上架商品入参。
type BotPublishGoodReq struct {
	UserID     uint     `json:"user_id"`    // 必填，旗下账号或主账号
	Title      string   `json:"title"`      // 必填
	Content    string   `json:"content"`    // 商品描述
	Category   int16    `json:"category"`   // 1=二手 2=有偿求助
	Negotiable bool     `json:"negotiable"` // true 时 price 字段被忽略
	Price      int      `json:"price"`      // 单位：分
	Location   string   `json:"location"`   // 地点（goods_addr）
	Images     []string `json:"images"`     // OSS URL 列表
}

// BotPublishGoodResp 上架成功返回 good_id。
type BotPublishGoodResp struct {
	GoodID uint `json:"good_id"`
}

// BotPublishGood 落库一个商品。
//
// 鉴权层假设 bot 只可能传**对应旗下账号**或**该旗下账号的主账号**的 user_id；
// 严格的"主账号能不能替旗下账号上架"语义在 controller 那一层把控。
func BotPublishGood(ctx context.Context, req BotPublishGoodReq) (*BotPublishGoodResp, error) {
	user, err := getActiveUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if req.Title == "" {
		return nil, errors.New("title 不能为空")
	}
	if req.Category != 1 && req.Category != 2 {
		return nil, errors.New("category 仅支持 1=二手 / 2=有偿求助")
	}

	uid := int(user.ID)
	sid := int(user.SchoolID)
	g := &model.Good{
		UserID:        &uid,
		SchoolID:      &sid,
		Title:         req.Title,
		Content:       req.Content,
		Status:        constant.StatusValid,
		GoodStatus:    1, // 在售
		GoodsType:     2, // 自提（默认；bot 上架场景以自提为主）
		GoodsAddr:     req.Location,
		PickupAddr:    req.Location,
		GoodsCategory: req.Category,
		Negotiable:    req.Negotiable,
		Price:         req.Price,
		MarkedPrice:   req.Price,
		Stock:         1, // 默认 1 件
		Images:        req.Images,
		ImageCount:    len(req.Images),
	}
	if _, err := dao.Good().Create(ctx, g); err != nil {
		return nil, fmt.Errorf("创建商品失败: %w", err)
	}
	return &BotPublishGoodResp{GoodID: g.ID}, nil
}

// BotOffShelfGood 下架商品；调用方 user_id 必须是 owner 或 owner 的主账号。
func BotOffShelfGood(ctx context.Context, goodID uint, callerUserID uint) error {
	g := &model.Good{}
	err := pgsql.DB.WithContext(ctx).
		Where("id = ? AND status = ?", goodID, constant.StatusValid).
		First(g).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrBotGoodNotFound
		}
		return err
	}
	if !canOperateAsOwner(ctx, g.UserID, callerUserID) {
		return ErrBotGoodNotOwner
	}
	if g.GoodStatus == 2 { // 已下架
		return nil // 幂等
	}
	return pgsql.DB.WithContext(ctx).
		Model(&model.Good{}).
		Where("id = ?", g.ID).
		UpdateColumn("good_status", 2).Error
}

// BotActiveGoodOfUser 单条在售商品的精简表示，给 bot 消歧用。
type BotActiveGoodOfUser struct {
	ID         uint   `json:"id"`
	Title      string `json:"title"`
	Negotiable bool   `json:"negotiable"`
	Price      int    `json:"price"`
	Category   int16  `json:"category"`
	CreatedAt  string `json:"created_at"` // ISO8601 字符串，给 bot 比对时间窗用
}

// BotListActiveGoodsOfUser 列出某用户当前在售（status=valid + good_status=1）商品；
// bot 在做"已出消歧"或"重发去重"时调。
func BotListActiveGoodsOfUser(ctx context.Context, userID uint, limit int) ([]*BotActiveGoodOfUser, error) {
	if _, err := getActiveUser(ctx, userID); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	var goods []*model.Good
	err := pgsql.DB.WithContext(ctx).
		Where("user_id = ? AND status = ? AND good_status = ?", int(userID), constant.StatusValid, 1).
		Order("id DESC").
		Limit(limit).
		Find(&goods).Error
	if err != nil {
		return nil, err
	}
	out := make([]*BotActiveGoodOfUser, 0, len(goods))
	for _, g := range goods {
		out = append(out, &BotActiveGoodOfUser{
			ID:         g.ID,
			Title:      g.Title,
			Negotiable: g.Negotiable,
			Price:      g.Price,
			Category:   g.GoodsCategory,
			CreatedAt:  g.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return out, nil
}

// =============================================================================
// 提问 / 回答 / 关闭提问 / 群内开放提问列表
// =============================================================================

// BotPublishArticleReq 创建提问/回答。type=2 提问、type=3 回答。
type BotPublishArticleReq struct {
	UserID   uint     `json:"user_id"`
	Type     int      `json:"type"`             // 2=提问 3=回答
	Title    string   `json:"title"`            // 提问必填；回答可空
	Content  string   `json:"content"`          // 必填
	ParentID *int     `json:"parent_id"`        // 回答必填，对应提问的 article id
	Images   []string `json:"images,omitempty"` // OSS URL
}

// BotPublishArticleResp 创建成功返回 article id。
type BotPublishArticleResp struct {
	ArticleID uint `json:"article_id"`
}

// BotPublishArticle 创建提问/回答。
func BotPublishArticle(ctx context.Context, req BotPublishArticleReq) (*BotPublishArticleResp, error) {
	user, err := getActiveUser(ctx, req.UserID)
	if err != nil {
		return nil, err
	}
	if req.Type != 2 && req.Type != 3 {
		return nil, errors.New("type 仅支持 2=提问 / 3=回答")
	}
	if req.Type == 3 && (req.ParentID == nil || *req.ParentID == 0) {
		return nil, errors.New("回答必须带 parent_id")
	}
	if req.Content == "" {
		return nil, errors.New("content 不能为空")
	}

	uid := int(user.ID)
	sid := int(user.SchoolID)
	a := &model.Article{
		UserID:        &uid,
		SchoolID:      &sid,
		ParentID:      req.ParentID,
		Title:         req.Title,
		Content:       req.Content,
		Status:        model.ArticleStatusNormal,
		PublishStatus: 2, // 公开
		Type:          req.Type,
		Images:        req.Images,
		ImageCount:    len(req.Images),
	}
	if err := pgsql.DB.WithContext(ctx).Create(a).Error; err != nil {
		return nil, fmt.Errorf("创建文章失败: %w", err)
	}
	return &BotPublishArticleResp{ArticleID: a.ID}, nil
}

// BotCloseArticle 关闭提问（status 改 4）。
func BotCloseArticle(ctx context.Context, articleID uint, callerUserID uint) error {
	a := &model.Article{}
	err := pgsql.DB.WithContext(ctx).Where("id = ?", articleID).First(a).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("bot: 文章不存在")
		}
		return err
	}
	if a.Type != 2 {
		return errors.New("bot: 仅提问类（type=2）支持关闭")
	}
	if !canOperateAsOwner(ctx, a.UserID, callerUserID) {
		return errors.New("bot: 仅文章作者或其主账号可关闭")
	}
	if a.Status == model.ArticleStatusClosed {
		return nil // 幂等
	}
	return pgsql.DB.WithContext(ctx).
		Model(&model.Article{}).
		Where("id = ?", a.ID).
		UpdateColumn("status", model.ArticleStatusClosed).Error
}

// BotOpenQuestion 群内开放提问的精简表示。
type BotOpenQuestion struct {
	ID        uint   `json:"id"`
	UserID    uint   `json:"user_id"`
	Title     string `json:"title"`
	Content   string `json:"content"`
	CreatedAt string `json:"created_at"`
}

// BotListOpenQuestionsByGroup 列出某 QQ 群对应学校下的开放提问，按 created_at desc。
//
// bot 想"给某条群里别人的提问发回答"时调，按 hint 文本匹配 title/content 找到 parent_id。
func BotListOpenQuestionsByGroup(ctx context.Context, groupID int64, limit int) ([]*BotOpenQuestion, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	schoolID, err := findSchoolIDByQQGroup(ctx, groupID)
	if err != nil {
		return nil, err
	}
	if schoolID == 0 {
		return nil, ErrBotGroupNoSchool
	}
	var arts []*model.Article
	err = pgsql.DB.WithContext(ctx).
		Where("school_id = ? AND type = 2 AND status = ? AND publish_status = 2",
			int(schoolID), model.ArticleStatusNormal).
		Order("id DESC").
		Limit(limit).
		Find(&arts).Error
	if err != nil {
		return nil, err
	}
	out := make([]*BotOpenQuestion, 0, len(arts))
	for _, a := range arts {
		uid := uint(0)
		if a.UserID != nil {
			uid = uint(*a.UserID)
		}
		out = append(out, &BotOpenQuestion{
			ID:        a.ID,
			UserID:    uid,
			Title:     a.Title,
			Content:   a.Content,
			CreatedAt: a.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	return out, nil
}

// =============================================================================
// 内部 helper
// =============================================================================

// getActiveUser 拿一个 status=valid 的用户；不存在 / 禁用 → ErrBotUserNotFound。
func getActiveUser(ctx context.Context, userID uint) (*model.User, error) {
	if userID == 0 {
		return nil, ErrBotUserNotFound
	}
	var u model.User
	err := pgsql.DB.WithContext(ctx).
		Where("id = ? AND status = ?", userID, constant.StatusValid).
		First(&u).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrBotUserNotFound
		}
		return nil, err
	}
	return &u, nil
}

// canOperateAsOwner 判断 callerUserID 是不是资源 owner_id 本人，或者 owner 的主账号。
//
// "主账号操作旗下账号资源"是 SKILL.md 里明确允许的——所以 caller 是 owner 的 parent_user_id 也算 OK。
func canOperateAsOwner(ctx context.Context, ownerUserIDPtr *int, callerUserID uint) bool {
	if ownerUserIDPtr == nil || callerUserID == 0 {
		return false
	}
	ownerID := uint(*ownerUserIDPtr)
	if ownerID == callerUserID {
		return true
	}
	// owner 是 caller 的旗下账号？
	owner, err := getActiveUser(ctx, ownerID)
	if err != nil || owner == nil {
		return false
	}
	if owner.ParentUserID != nil && uint(*owner.ParentUserID) == callerUserID {
		return true
	}
	return false
}
