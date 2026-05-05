// Package service 的 order_urge.go 实现 P3.3 "QQ 加急"——把订单聊天里的某条
// 消息一键推到对方 QQ 私聊。
//
// 设计契约（详见 QQ-bot/skill/bot/SKILL.md "P3.3 QQ 加急"段）：
//
//  1. **谁能加急**：caller 必须是订单参与方（买方 / 卖方 / 求助发布者 / 接单者，
//     都通过账号集 resolveOrderParticipant 校验）；且消息**必须是 caller 自己发的**——
//     加急别人的消息没有业务语义。
//  2. **接收人路由**：caller 是 buyer → 接收人 = seller，反之亦然。接收人 user 经
//     resolveRecipientQQ 解析为目标 QQ：
//     - 普通账号 + 有绑定 QQ child → 用 child.qq_number 私聊
//     - 孤儿 QQ child（本身就是 QQ 用户）→ 用 user.qq_number 私聊
//     - 普通账号 + 没绑 QQ → 拒绝（前端提示让用户先在 app 内绑 QQ）
//     **不做群里 @ 兜底**——加急是私聊提醒，群发会泄露订单信息且偏离设计语义。
//  3. **限流**：同一 (order, caller) 5 分钟内只能加急 1 次；redis key
//     `order_urge_throttle:{order_id}:{caller_id}`，TTL=5min。
//  4. **幂等**：同一条消息只能加急 1 次（DB 列 urgent + WHERE urgent=false）。
//  5. **审计**：成功加急后写一条 official 类型的订单消息进 order_messages
//     ("已加急提醒对方查看消息（HH:mm）") 让对话双方都能看到事件——
//     避免"对方 QQ 收到加急但 app 里看起来啥都没发生"的体验割裂。
package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service/errno"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/botinternal"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	commonredis "github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// =============================================================================
// 限流
// =============================================================================

// orderUrgeThrottleTTL 同一 (order, caller) 在多长时间内只允许加急 1 次。
//
// 5 分钟：足够避免 spam（典型场景：用户点了加急但对方还没看到，5min 内不该再点）；
// 又比验证码限流（60s）长得多——加急是真去打扰对方私聊的"重型"动作，节奏要更克制。
const orderUrgeThrottleTTL = 5 * time.Minute

func orderUrgeThrottleKey(orderID uint, callerID uint) string {
	return fmt.Sprintf("order_urge_throttle:%d:%d", orderID, callerID)
}

// =============================================================================
// 接收人 QQ 解析
// =============================================================================

// resolveRecipientQQ 给定 hfut user_id，找出加急时该私聊的 QQ 号。
//
// 业务语义（严格按"对方绑了 QQ → bot 私聊"的口径）：
//
//	user 是 normal + 名下挂 QQ child → 返回 child.qq_number
//	user 是 orphan QQ child（无主账号但本身就是 QQ 用户）→ 返回 user.qq_number
//	user 是 normal + 没绑 QQ → ErrOrderUrgeRecipientNoQQ
//
// 故意**不再做 group 兜底**——加急是私聊提醒，群里 @ 会泄露订单信息且偏离设计语义；
// SendPrivate 失败时直接报"机器人不可达"。
//
// 返回 0 永远表示"没法加急"，调用方应当返回 ErrOrderUrgeRecipientNoQQ。
func resolveRecipientQQ(ctx context.Context, userID uint) (int64, error) {
	if userID == 0 {
		return 0, errno.ErrOrderUrgeRecipientNoQQ
	}
	var u model.User
	if err := pgsql.DB.WithContext(ctx).
		Select("id, account_type, parent_user_id, qq_number, status").
		Where("id = ? AND status = ?", userID, constant.StatusValid).
		First(&u).Error; err != nil {
		return 0, errno.ErrOrderUrgeRecipientNoQQ
	}

	// 接收人本身是普通账号 → 找它名下的 QQ child（绑过的 QQ）
	if !u.IsQQChild() {
		var child model.User
		parentID := int(u.ID)
		err := pgsql.DB.WithContext(ctx).
			Select("id, qq_number").
			Where("parent_user_id = ? AND account_type = ? AND status = ?",
				parentID, model.AccountTypeQQChild, constant.StatusValid).
			First(&child).Error
		if err != nil {
			return 0, errno.ErrOrderUrgeRecipientNoQQ
		}
		if child.QQNumber == nil || *child.QQNumber == "" {
			return 0, errno.ErrOrderUrgeRecipientNoQQ
		}
		qqInt, perr := strconv.ParseInt(*child.QQNumber, 10, 64)
		if perr != nil {
			return 0, errno.ErrOrderUrgeRecipientNoQQ
		}
		return qqInt, nil
	}

	// 接收人是 QQ child——本身就是 QQ 用户（孤儿 / 已绑都直接用它的 qq_number）。
	// 已绑场景理论被 ResolveTargetUserID 重定向到 parent 了，走到这里几乎只剩孤儿。
	if u.QQNumber == nil || *u.QQNumber == "" {
		return 0, errno.ErrOrderUrgeRecipientNoQQ
	}
	qqInt, perr := strconv.ParseInt(*u.QQNumber, 10, 64)
	if perr != nil {
		return 0, errno.ErrOrderUrgeRecipientNoQQ
	}
	return qqInt, nil
}

// =============================================================================
// 业务函数
// =============================================================================

// OrderMessageUrge 把指定订单消息推到对方 QQ。
//
// 错误返回（按优先级）：
//   - 限流命中 → *ThrottledError
//   - 订单 / 消息不存在 → ErrOrderNotFound / ErrOrderMessageNotFound
//   - 不是参与方 / 不是自己的消息 → ErrOrderNotParticipant / ErrOrderMessageNotMine
//   - 已加急过 → ErrOrderMessageAlreadyUrgent
//   - 对方未绑 QQ → ErrOrderUrgeRecipientNoQQ
//   - bot 不可达 → ErrOrderUrgeBotUnavailable
func (s *orderService) OrderMessageUrge(ctx *gin.Context, orderID uint, msgID uint, callerUserID uint) error {
	rctx := ctx.Request.Context()

	// 1) 校验 caller 是订单参与方 + 拿到 order/good
	o, g, isBuyer, isSeller, err := s.resolveOrderParticipant(ctx, orderID, callerUserID)
	if err != nil {
		return err
	}

	// 2) 找消息 + 校验归属（必须是 caller 自己发的且属于该订单）
	msg, err := dao.OrderMessage().GetByID(rctx, msgID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errno.ErrOrderMessageNotFound
		}
		return fmt.Errorf("查询订单消息失败: %w", err)
	}
	if msg.OrderID != orderID {
		return errno.ErrOrderMessageNotInOrder
	}
	// 通过账号集判断"我发的"——caller 主账号或其旗下号 send_id 都算
	ids, ierr := GetAccountIDsForOps(rctx, callerUserID)
	if ierr != nil {
		return errno.ErrOrderMessageNotMine
	}
	if !ids.IsOwnedByOneOf(uint(msg.SenderID)) {
		return errno.ErrOrderMessageNotMine
	}
	if msg.Urgent {
		return errno.ErrOrderMessageAlreadyUrgent
	}
	if msg.MsgType == constant.OrderMsgTypeOfficial {
		// 官方消息不允许加急——本来就是系统提示，没有"提醒对方"的意义
		return errno.ErrOrderMessageNotMine
	}

	// 3) 限流（基于 redis SetNX，跟 qq_bind throttle 同一套思路）
	throttleKey := orderUrgeThrottleKey(orderID, callerUserID)
	ok, terr := commonredis.Client.SetNX(rctx, throttleKey, "1", orderUrgeThrottleTTL).Result()
	if terr != nil {
		return fmt.Errorf("加急限流锁失败: %w", terr)
	}
	if !ok {
		retryAfter := 0
		if d, derr := commonredis.Client.TTL(rctx, throttleKey).Result(); derr == nil && d > 0 {
			retryAfter = int(d.Seconds())
			if retryAfter == 0 {
				retryAfter = 1
			}
		}
		return &ThrottledError{RetryAfterSeconds: retryAfter}
	}
	// 注：拿到锁后即便后面失败也不释放锁——避免用户疯狂重试拖死 bot；
	// 跟 qq_bind RequestCode 同样的设计。

	// 4) 解析接收人 → QQ
	var recipientUserID uint
	switch {
	case isBuyer && g.UserID != nil:
		recipientUserID = uint(*g.UserID)
	case isSeller && o.UserID != nil:
		recipientUserID = uint(*o.UserID)
	default:
		return errno.ErrOrderNotParticipant
	}
	recipientQQ, err := resolveRecipientQQ(rctx, recipientUserID)
	if err != nil {
		return err
	}

	// 5) bot 必备
	if botinternal.Default == nil {
		logger.Warnf(rctx, "order_urge: botinternal.Default == nil（BOT_INTERNAL_API_URL 没配 / URL 不合法）")
		return errno.ErrOrderUrgeBotUnavailable
	}

	// 6) DB 标记 urgent（条件 update，确保幂等）
	now := time.Now()
	rows, uerr := dao.OrderMessage().MarkUrgent(rctx, msgID, now)
	if uerr != nil {
		return fmt.Errorf("标记 urgent 失败: %w", uerr)
	}
	if rows == 0 {
		// 跟前面 msg.Urgent 检查 race——并发加急同一条消息时另一条已经赢
		return errno.ErrOrderMessageAlreadyUrgent
	}

	// 7) 发 QQ：严格私聊。失败 → 直接报 bot 不可达，**不做群里 @ 兜底**
	// （群里 @ 会把订单内容暴露给整个群，偏离"私下提醒对方"的设计语义）。
	text := buildUrgeText(o, msg, isBuyer)
	if perr := botinternal.Default.SendPrivate(rctx, recipientQQ, text); perr != nil {
		logger.Warnf(rctx, "order_urge: SendPrivate 失败 qq=%d err=%v", recipientQQ, perr)
		// QQ 没发出去，但 DB 已经标记 urgent + 限流锁也已经占——刻意保留这两个状态：
		// urgent 标记代表"已尝试加急"，前端可在气泡上区分；限流锁防止 spam 重试。
		return errno.ErrOrderUrgeBotUnavailable
	}

	// 8) 写一条 official 消息让对话双方都能看到加急事件（不阻塞主流程；失败仅 log）
	official := &model.OrderMessage{
		OrderID:  orderID,
		SenderID: 0, // 0 = 系统
		MsgType:  constant.OrderMsgTypeOfficial,
		Content:  fmt.Sprintf("已加急提醒对方查看消息（%s）", now.Format("15:04")),
	}
	if oerr := dao.OrderMessage().Create(rctx, official); oerr != nil {
		logger.Warnf(rctx, "order_urge: 写 official 消息失败（不影响主流程） err=%v", oerr)
	}

	return nil
}

// buildUrgeText 构造发往对方 QQ 的私聊提醒；区分 isBuyer 让接收方一眼看清"是哪边"找他。
//
// 文案精简（跟 P3.1 前端文案口径一致）：不引用消息原文太长，给关键摘要 + 跳转引导。
func buildUrgeText(o *model.Order, msg *model.OrderMessage, callerIsBuyer bool) string {
	role := "卖家"
	if callerIsBuyer {
		role = "买家"
	}
	// 引用消息：取前 60 字防 QQ 私聊单条字数限制
	excerpt := msg.Content
	if msg.MsgType == constant.OrderMsgTypeImage {
		excerpt = "[图片]"
	}
	excerpt = truncateChinese(excerpt, 60)
	if excerpt == "" {
		excerpt = "(空消息)"
	}
	return fmt.Sprintf("【HFUT 校园平台】订单 #%d 的%s加急了一条消息：%s\n请及时打开 app 查看",
		o.ID, role, excerpt)
}

// truncateChinese 按 rune 截断到 max 长度，超出加省略号；max <= 0 时直接返回原串。
func truncateChinese(s string, max int) string {
	if max <= 0 {
		return s
	}
	rs := []rune(s)
	if len(rs) <= max {
		return s
	}
	return string(rs[:max]) + "…"
}
