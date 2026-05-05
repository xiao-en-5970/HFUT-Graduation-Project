// Package service 的 user_account.go 实现"账号集"权限模型。
//
// 核心抽象：一个登录的主账号实际能 access 的资源 = 主账号自己 + 它名下挂的旗下账号。
// 所有"我的 X"接口的权限校验、列表查询、数据聚合都基于这个集合。
//
// 详见 QQ-bot/skill/bot/SKILL.md "数据聚合 / 操作权限" 段。
package service

import (
	"context"
	"errors"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// AccountIDSet 一个 caller 在做"我的"操作时能 access 的所有 user_id 集合。
//
// 字段：
//
//	Caller       发起操作的主账号 id（前端 JWT 解出来的）
//	ChildID      旗下账号 id；若 caller 没绑 QQ 旗下号则为 0
//	AllIDs       去重的全集 = [Caller, ChildID(若 != 0)]，给 dao 层 IN 查询直接用
//
// 用法：
//
//	ids, err := GetAccountIDsForOps(ctx, callerUserID)
//	if err != nil { ... }
//	dao.Good().IsOwnedByOneOf(ctx, goodID, ids.AllIDs)
//
// 边界：
//   - 如果 caller 自己是个旗下账号（理论上不该走到这里，因为旗下号不能登录），
//     IsQQChild 返 true 时直接返回 {Caller}（不递归找它的 parent，避免角色错乱）
//   - caller user_id == 0：返 ErrInvalidCaller
type AccountIDSet struct {
	Caller  uint
	ChildID uint   // 0 表示没绑
	AllIDs  []uint // 去重 + 至少含 Caller
}

// IsAggregated 是否真的聚合了多账号（caller + 旗下号 != caller 自己）。
//
// false 时表示这个 caller 只有自己一个 user_id；上层可以走更简单的"==" 比较跳过 IN 查询。
// （只是优化，不走也不影响结果。）
func (s AccountIDSet) IsAggregated() bool {
	return len(s.AllIDs) > 1
}

// IsOwnedByOneOf 给定一个资源的 owner_user_id（来自 goods/articles 表的 user_id 字段），
// 判断这个 caller 是否有权操作。
//
// 是 caller 自己 || 是 caller 的旗下账号 → true
func (s AccountIDSet) IsOwnedByOneOf(ownerUserID uint) bool {
	for _, id := range s.AllIDs {
		if id == ownerUserID {
			return true
		}
	}
	return false
}

// IsFromChild 这个 owner_user_id 是不是 caller 的旗下账号（用来给前端打 "来自 QQ" tag）。
//
// owner_user_id == ChildID 时 true；其它情况（owner 是 caller 自己 / 不属于这个集合）false。
func (s AccountIDSet) IsFromChild(ownerUserID uint) bool {
	return s.ChildID != 0 && ownerUserID == s.ChildID
}

// ErrInvalidCaller caller user_id 为 0 或非法。
var ErrInvalidCaller = errors.New("无效的 caller user_id")

// GetAccountIDsForOps 拿 caller 的"账号集"。
//
// 实现：1 次 SQL 查 caller，2 次 SQL 查它名下挂的旗下账号；旗下账号最多 1 个（业务约束 1:1）。
// 如果以后要支持"主账号挂多个旗下号"，这里改成查询返回 []，AllIDs 把全部塞进去就行——上层
// 接口（IsOwnedByOneOf/IsFromChild）已经按集合语义实现，无需变化。
//
// 性能：一次绑定决策结果可在请求生命周期内缓存（gin context 加 key 就行），但 P2b 阶段
// 这俩 SQL 走主键索引每次几百微秒，不优化也无所谓。
func GetAccountIDsForOps(ctx context.Context, callerUserID uint) (AccountIDSet, error) {
	if callerUserID == 0 {
		return AccountIDSet{}, ErrInvalidCaller
	}

	// 1) 校验 caller 存在 + active；同时拿到 account_type 决定要不要查旗下号
	var caller model.User
	if err := pgsql.DB.WithContext(ctx).
		Where("id = ? AND status = ?", callerUserID, constant.StatusValid).
		First(&caller).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return AccountIDSet{}, ErrUserNotFound
		}
		return AccountIDSet{}, err
	}

	// caller 自己就是旗下账号——按设计旗下号无法登录走到这里（password 空）；
	// 但万一有 admin 直接 SQL 改了什么导致旗下号能登录，安全起见**只**返回它自己，
	// 不递归向上找 parent（避免攻击面：旗下号一旦能登录就能 access parent 的资源）。
	if caller.IsQQChild() {
		return AccountIDSet{
			Caller: caller.ID,
			AllIDs: []uint{caller.ID},
		}, nil
	}

	// 2) 找 caller 名下挂的旗下账号（业务上 1:1，但 SQL 上 First 安全）
	var child model.User
	parentID := int(caller.ID)
	cerr := pgsql.DB.WithContext(ctx).
		Where("parent_user_id = ? AND account_type = ? AND status = ?",
			parentID, model.AccountTypeQQChild, constant.StatusValid).
		First(&child).Error
	if cerr != nil && !errors.Is(cerr, gorm.ErrRecordNotFound) {
		return AccountIDSet{}, cerr
	}

	if errors.Is(cerr, gorm.ErrRecordNotFound) {
		// caller 没绑过 QQ
		return AccountIDSet{
			Caller: caller.ID,
			AllIDs: []uint{caller.ID},
		}, nil
	}

	return AccountIDSet{
		Caller:  caller.ID,
		ChildID: child.ID,
		AllIDs:  []uint{caller.ID, child.ID},
	}, nil
}

// isSelfTrade 判断"caller 在跟 ownerUserID 交易时，是不是变相跟自己交易"——
// 也就是 caller 跟 ownerUserID **在同一个账号集**里。用于：
//
//   - 下单时拒绝 buyer 购买自己挂的商品（直接关系）
//   - 拒绝 buyer 购买**自己旗下号**挂的商品（账号集关系）——主账号在 app 给自己旗下号商品下单
//     等同自己跟自己交易，可被滥用刷单 / 凑订单数 / 测评分等
//   - 接单求助时同理：拒绝 taker 接自己的有偿求助 / 自己旗下号的有偿求助
//
// 实现走 GetAccountIDsForOps + IsOwnedByOneOf；账号集查不到时回退到直接相等。
//
// 取签名 (gin.Context) 是为了跟 service 层调用方风格对齐——内部用 ctx.Request.Context()。
func isSelfTrade(ctx *gin.Context, callerUserID, ownerUserID uint) bool {
	if callerUserID == 0 || ownerUserID == 0 {
		return false
	}
	if callerUserID == ownerUserID {
		return true
	}
	ids, err := GetAccountIDsForOps(ctx.Request.Context(), callerUserID)
	if err != nil {
		return callerUserID == ownerUserID // fallback：账号集查不到时只比直接相等
	}
	return ids.IsOwnedByOneOf(ownerUserID)
}

// InboundTargetKind 接收人种类——决定 inbound 通知的处理路径。
type InboundTargetKind int

const (
	// InboundNormal 普通账号（含已 CAS 绑学校或未绑学校的 normal）：通知正常入库。
	InboundNormal InboundTargetKind = iota + 1
	// InboundBoundChild 已绑主账号的 QQ 旗下号：重定向到 parent 后正常入库（详见 P2b 设计）。
	InboundBoundChild
	// InboundOrphan 孤儿 QQ 旗下号（parent_user_id IS NULL）：不入库，bot 转发到原创建群。
	InboundOrphan
	// InboundInvalid target 不存在 / 已禁用 / user_id<=0：静默丢弃。
	InboundInvalid
)

// InboundTarget 接收人状态详情，给 ResolveInboundTarget 返回。
//
// 字段含义按 Kind 不同有效性也不同——见各 Kind 注释。
type InboundTarget struct {
	Kind InboundTargetKind

	// EffectiveUserID 入库时应该写的 user_id。
	//   Normal / BoundChild / Orphan：分别是 自己 / parent / 旗下号自己
	//   Invalid：0
	EffectiveUserID uint

	// OrphanGroupID 孤儿场景：bot 转发回的 QQ 群号；0 表示历史数据没记录创建群（fallback：跳过转发 + log）
	OrphanGroupID int64

	// OrphanQQNumber 孤儿场景：旗下号的 QQ 号——bot SendGroup 用它做 @ 字段
	OrphanQQNumber string
}

// ResolveInboundTarget 给 inbound 通知 / 评论 / 私信等"被动接收"操作分类 target user。
//
// 用于：notificationService.emit / emitAggregatedLike 入库前分流。
//
// 设计语义详见 SKILL.md "孤儿旗下账号特殊行为" + "数据聚合 / 操作权限"段。
func ResolveInboundTarget(ctx context.Context, targetUserID uint) InboundTarget {
	if targetUserID == 0 {
		return InboundTarget{Kind: InboundInvalid}
	}
	var u model.User
	err := pgsql.DB.WithContext(ctx).
		Select("id, account_type, parent_user_id, qq_number, created_in_group_id, status").
		Where("id = ? AND status = ?", targetUserID, constant.StatusValid).
		First(&u).Error
	if err != nil {
		return InboundTarget{Kind: InboundInvalid}
	}
	// 普通账号（即便绑了学校 / 没绑学校都算）
	if !u.IsQQChild() {
		return InboundTarget{Kind: InboundNormal, EffectiveUserID: u.ID}
	}
	// 已绑主账号的旗下号 → 重定向 parent
	if u.ParentUserID != nil && *u.ParentUserID > 0 {
		return InboundTarget{Kind: InboundBoundChild, EffectiveUserID: uint(*u.ParentUserID)}
	}
	// 孤儿
	out := InboundTarget{
		Kind:            InboundOrphan,
		EffectiveUserID: u.ID,
	}
	if u.CreatedInGroupID != nil && *u.CreatedInGroupID != 0 {
		out.OrphanGroupID = *u.CreatedInGroupID
	}
	if u.QQNumber != nil {
		out.OrphanQQNumber = *u.QQNumber
	}
	return out
}

// ResolveTargetUserID "接收人重定向"——把指向某 user_id 的"被动接收"操作（被通知 / 被评论 /
// 被聊天 / 被回复）的 target user_id 转成它的"实际持有者"。
//
// 规则：
//   - target 是 **绑定了主账号** 的 QQ 旗下账号 → 返回 parent_user_id（主账号）
//   - target 是 **孤儿** 旗下账号 / 不存在 / 普通账号 → 原样返回 target
//
// 设计动机（详见 QQ-bot/skill/bot/SKILL.md "数据聚合 / 操作权限"）：
//
// 用户的"账号集"重新定位旗下账号——它仅是"通过 QQ 渠道发布"的标签，**不是真正的账号**。
// 所以所有"对旗下号的 inbound"（通知 / 评论 / 私信等）应该直接落到主账号身上，不让旗下号
// 持有 inbox。这样：
//
//  1. 主账号在 app 的"我的通知" 不需要再聚合查询（DB 里 user_id 直接就是主账号 id）
//  2. 旗下号没有"对话"概念——所有通过 QQ 发的资源被回复时，主账号在 app 内直接处理
//  3. 跟孤儿旗下号天然区分：孤儿没主账号接管，回复仍由 bot 转发回 QQ 群（P2c）
//
// 性能：一次 SQL 命中索引，加 ~ 1ms；高频路径可在 ctx 内缓存。
func ResolveTargetUserID(ctx context.Context, targetUserID uint) uint {
	if targetUserID == 0 {
		return 0
	}
	var u model.User
	err := pgsql.DB.WithContext(ctx).
		Select("id, account_type, parent_user_id, status").
		Where("id = ? AND status = ?", targetUserID, constant.StatusValid).
		First(&u).Error
	if err != nil {
		return targetUserID
	}
	if u.IsQQChild() && u.ParentUserID != nil && *u.ParentUserID > 0 {
		return uint(*u.ParentUserID)
	}
	return targetUserID
}
