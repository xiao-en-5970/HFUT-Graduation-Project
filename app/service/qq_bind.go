// Package service 的 qq_bind.go 实现"主账号 ↔ QQ 旗下账号"的绑定 / 解绑流程。
//
// 设计契约（详见 QQ-bot/skill/bot/SKILL.md "绑定 QQ 流程"段）：
//
//  1. **前置**：主账号必须已绑学校（user.school_id != 0）；这是身份严肃性的兜底
//  2. **流程**：app 输入 QQ → hfut 调 bot CheckFriend → bot 是好友 → hfut 生成 6 位
//     验证码存 redis（key 含 qq_number + caller user_id）→ hfut 调 bot SendPrivate
//     → app 输入验证码 → hfut 校验 redis 命中 → 挂载旗下账号 + 学校信息覆盖 + tx commit
//  3. **限流**：同一 user 5min 内最多请求 1 次验证码
//  4. **严格 1:1**：一个主账号最多 1 个 QQ 旗下账号；要绑别的 QQ 必须先解绑
//  5. **学校归属**：严格 > 不严格——挂载时主账号的 school_id 直接覆盖旗下账号的旧 school_id
//  6. **解绑**：parent_user_id 设回 NULL，旗下账号变孤儿，**保留所有数据**
package service

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao/model"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/botinternal"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/pgsql"
	commonredis "github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/constant"
)

// =============================================================================
// 错误 sentinel
// =============================================================================

var (
	// ErrUserNotFound 主账号不存在 / 已禁用
	ErrUserNotFound = errors.New("主账号不存在或已禁用")
	// ErrUserNotBoundSchool 主账号还没绑学校；前置不满足
	ErrUserNotBoundSchool = errors.New("请先在'我的'页面绑定学校（CAS 认证）后再绑定 QQ")
	// ErrUserAlreadyBoundQQ 主账号已经绑了 QQ；要绑新 QQ 必须先解绑
	ErrUserAlreadyBoundQQ = errors.New("当前账号已绑定 QQ，要绑定其它 QQ 请先解绑")
	// ErrBotNotFriend 目标 QQ 不是 bot 的好友——让用户先去加 bot 为好友
	ErrBotNotFriend = errors.New("目标 QQ 还不是机器人的好友，请先添加机器人 QQ 为好友再尝试绑定")
	// ErrBotUnavailable bot/NapCat 整体不可达——让前端提示"系统繁忙稍后重试"
	ErrBotUnavailable = errors.New("机器人服务暂时不可达，请稍后重试")
	// ErrCodeInvalid 验证码错（格式 / 不匹配）
	ErrCodeInvalid = errors.New("验证码错误")
	// ErrCodeExpired 验证码过期 / 不存在
	ErrCodeExpired = errors.New("验证码已过期，请重新获取")
	// ErrThrottled 短期内重复请求验证码（限流）——sentinel 用于 errors.Is 简化判别。
	//
	// 实际抛出的错误是 *ThrottledError（实现了 Is 接口指向 ErrThrottled），带剩余
	// 秒数。controller 拿这个剩余秒数放到 response data 里，给前端做按钮倒计时。
	ErrThrottled = errors.New("请求过于频繁")
	// ErrQQNumberInvalid QQ 号格式错
	ErrQQNumberInvalid = errors.New("QQ 号格式错误（应为 5-12 位数字）")
	// ErrUserHasNoQQChild 解绑时找不到当前主账号的旗下账号
	ErrUserHasNoQQChild = errors.New("当前账号未绑定 QQ，无需解绑")
)

// ThrottledError 限流命中时返回的结构化错误，带剩余可重试秒数。
//
// controller 用 errors.As 拿到这个错，把 RetryAfterSeconds 放到 429 response 的
// data 字段里——前端拿这个数做"获取验证码"按钮的倒计时。
//
// 实现 Is(target) 让 errors.Is(err, ErrThrottled) 仍然命中——上层只想粗略分流时
// 不用 errors.As 也能识别。
type ThrottledError struct {
	RetryAfterSeconds int
}

func (e *ThrottledError) Error() string {
	if e.RetryAfterSeconds <= 0 {
		return "请求过于频繁，请稍后再试"
	}
	return fmt.Sprintf("请求过于频繁，请 %d 秒后再试", e.RetryAfterSeconds)
}

// Is 让 errors.Is(err, ErrThrottled) 命中——保持 sentinel 友好。
func (e *ThrottledError) Is(target error) bool {
	return target == ErrThrottled
}

// =============================================================================
// redis key / TTL
// =============================================================================

const (
	// qqBindCodeTTL 验证码有效期；用户输入验证码超过这个时间就失效
	qqBindCodeTTL = 5 * time.Minute
	// qqBindThrottleTTL 同一 user 请求验证码的最小间隔——跟前端"获取验证码"按钮的
	// 倒计时对齐；60s 是常规体验。
	qqBindThrottleTTL = 60 * time.Second
)

// qqBindCodeKey 绑定流程的 redis key `qq_bind_code:{qq_number}`
//
// 故意按 qq_number 而不是 user_id 当 key——避免"同一 user 短时间请求多个 QQ 的验证码"
// 时把别人的覆盖掉；按 qq_number 索引验证码也跟"用户在 app 输入验证码"那一步对得上。
func qqBindCodeKey(qq string) string {
	return "qq_bind_code:" + qq
}

// qqBindThrottleKey 绑定流程的 redis key `qq_bind_throttle:{user_id}`
//
// 限流维度是 caller user_id——同一用户 60s 内只能请求 1 次（不管要绑啥 QQ）。
func qqBindThrottleKey(userID uint) string {
	return "qq_bind_throttle:" + strconv.FormatUint(uint64(userID), 10)
}

// qqUnbindCodeKey 解绑流程的 redis key `qq_unbind_code:{qq_number}`
//
// 跟绑定的 key 不同前缀——避免"用户先触发绑定流程拿到 code A，紧接着触发解绑覆盖成
// code B"这类混淆；两个流程的 code 各自独立。
func qqUnbindCodeKey(qq string) string {
	return "qq_unbind_code:" + qq
}

// qqUnbindThrottleKey 解绑流程的 redis key `qq_unbind_throttle:{user_id}`。
//
// 跟绑定的 throttle key 也不同前缀——绑/解 两套限流互不影响：
// 用户绑定遇到限流时，仍然可以发起解绑流程（虽然没绑过 QQ 的话会被业务层拒绝）。
func qqUnbindThrottleKey(userID uint) string {
	return "qq_unbind_throttle:" + strconv.FormatUint(uint64(userID), 10)
}

// qqBindCodePayload 存进 redis 的结构（json 序列化）。
type qqBindCodePayload struct {
	Code             string `json:"code"`               // 6 位数字
	RequestingUserID uint   `json:"requesting_user_id"` // 当时发起请求的主账号 id
	CreatedAt        int64  `json:"created_at"`         // unix 秒，给前端"剩余有效期"展示
}

// =============================================================================
// 业务函数
// =============================================================================

// QQBindRequestCode 是绑定流程第一步：让 bot 给目标 QQ 发验证码私聊。
//
// 校验 + 限流 + 调 bot 链路全在这里做完；通过后返回 ttl 秒数（让前端展示倒计时）。
//
// 错误：见上面 sentinel；调用方 controller 应该按 errors.Is 分流回不同的 HTTP 状态码。
func QQBindRequestCode(ctx context.Context, callerUserID uint, qqNumber string) (ttlSeconds int, err error) {
	qqNumber = strings.TrimSpace(qqNumber)
	if !isValidQQNumber(qqNumber) {
		return 0, ErrQQNumberInvalid
	}

	// 1) 取主账号 + 校验前置
	user, err := getActiveUser(ctx, callerUserID)
	if err != nil {
		if errors.Is(err, ErrBotUserNotFound) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	// 必须是 normal 账号（不能是旗下号自己发起绑定）
	if user.AccountType != model.AccountTypeNormal {
		return 0, ErrUserNotFound
	}
	if user.SchoolID == 0 {
		return 0, ErrUserNotBoundSchool
	}
	// 已经有绑过的旗下账号？严格 1:1
	if has, err := hasBoundQQChild(ctx, user.ID); err != nil {
		return 0, fmt.Errorf("查询当前 QQ 绑定状态失败: %w", err)
	} else if has {
		return 0, ErrUserAlreadyBoundQQ
	}

	// 2) 限流：同一 user 60s 内只能请求 1 次
	throttleKey := qqBindThrottleKey(user.ID)
	ok, err := commonredis.Client.SetNX(ctx, throttleKey, "1", qqBindThrottleTTL).Result()
	if err != nil {
		return 0, fmt.Errorf("限流锁失败: %w", err)
	}
	if !ok {
		// 拿剩余 TTL 让用户知道还要等多久；TTL 失败时 fallback 到 0（错误信息会变成 "稍后再试"）
		retryAfter := 0
		if d, terr := commonredis.Client.TTL(ctx, throttleKey).Result(); terr == nil && d > 0 {
			retryAfter = int(d.Seconds())
			// redis TTL 向下取整后可能丢精度，给前端的倒计时至少 1s 起步
			if retryAfter == 0 {
				retryAfter = 1
			}
		}
		return 0, &ThrottledError{RetryAfterSeconds: retryAfter}
	}
	// 注意：拿到锁之后即便后面失败也保留锁——避免用户疯狂重试拖死 bot；
	// 锁会按 TTL 自然过期。

	// 3) 调 bot 看 QQ 是不是好友
	if botinternal.Default == nil {
		logger.Warnf(ctx, "qq_bind: botinternal.Default == nil（BOT_INTERNAL_API_URL 没配 / URL 不合法）")
		return 0, ErrBotUnavailable
	}
	qqInt, _ := strconv.ParseInt(qqNumber, 10, 64)
	isFriend, err := botinternal.Default.CheckFriend(ctx, qqInt, true /* noCache */)
	if err != nil {
		// log 原始错——网络层 / DNS / 容器互通问题都会在这里冒头
		logger.Error(ctx, "qq_bind: 调 bot CheckFriend 失败", zap.Int64("qq", qqInt), zap.Error(err))
		return 0, ErrBotUnavailable
	}
	if !isFriend {
		return 0, ErrBotNotFriend
	}

	// 4) 生成 6 位验证码 + 存 redis
	//
	// 显式删旧 code 再 Set 新的——虽然 redis Set 本身就是覆盖语义，但显式 Del 一下让
	// "用户重新发起请求时旧验证码必然失效" 这个意图在代码里更明确，防御后续逻辑改动
	// 无意中改成 SetNX/SetXX 等条件性写入语义。
	_ = commonredis.Client.Del(ctx, qqBindCodeKey(qqNumber)).Err()
	code, err := generateBindCode()
	if err != nil {
		return 0, fmt.Errorf("生成验证码失败: %w", err)
	}
	payload := qqBindCodePayload{
		Code:             code,
		RequestingUserID: user.ID,
		CreatedAt:        time.Now().Unix(),
	}
	raw, _ := json.Marshal(payload)
	if err := commonredis.Client.Set(ctx, qqBindCodeKey(qqNumber), raw, qqBindCodeTTL).Err(); err != nil {
		return 0, fmt.Errorf("缓存验证码失败: %w", err)
	}

	// 5) 调 bot 发私聊验证码
	text := fmt.Sprintf("【HFUT 校园平台】您正在绑定 QQ %s 到 app 账号，验证码：%s（5 分钟内有效，请勿外泄）", qqNumber, code)
	if err := botinternal.Default.SendPrivate(ctx, qqInt, text); err != nil {
		logger.Error(ctx, "qq_bind: 调 bot SendPrivate 失败", zap.Int64("qq", qqInt), zap.Error(err))
		// 删 redis code 避免"验证码已发但其实没发到"的脏状态——重置流程方便用户重试
		_ = commonredis.Client.Del(ctx, qqBindCodeKey(qqNumber)).Err()
		if errors.Is(err, botinternal.ErrBotNotFriend) {
			return 0, ErrBotNotFriend
		}
		return 0, ErrBotUnavailable
	}

	return int(qqBindCodeTTL.Seconds()), nil
}

// QQBindConfirm 绑定流程第二步：用户在 app 输入验证码后，校验 redis + 挂载旗下账号。
//
// 学校归属处理：用主账号 SchoolID 直接覆盖旗下账号的（不严格让位严格）。
func QQBindConfirm(ctx context.Context, callerUserID uint, qqNumber, code string) error {
	qqNumber = strings.TrimSpace(qqNumber)
	code = strings.TrimSpace(code)
	if !isValidQQNumber(qqNumber) {
		return ErrQQNumberInvalid
	}
	if len(code) != 6 {
		return ErrCodeInvalid
	}

	// 1) 主账号校验（同 RequestCode）
	user, err := getActiveUser(ctx, callerUserID)
	if err != nil {
		if errors.Is(err, ErrBotUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if user.AccountType != model.AccountTypeNormal {
		return ErrUserNotFound
	}
	if user.SchoolID == 0 {
		return ErrUserNotBoundSchool
	}
	if has, err := hasBoundQQChild(ctx, user.ID); err != nil {
		return fmt.Errorf("查询当前 QQ 绑定状态失败: %w", err)
	} else if has {
		return ErrUserAlreadyBoundQQ
	}

	// 2) 校验验证码
	raw, err := commonredis.Client.Get(ctx, qqBindCodeKey(qqNumber)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCodeExpired
		}
		return fmt.Errorf("读验证码失败: %w", err)
	}
	var payload qqBindCodePayload
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return ErrCodeExpired
	}
	if payload.RequestingUserID != user.ID {
		// 不同 user 用了同一个 qq 的 code？拒绝；不报"已被别人发起"——避免泄漏信息
		return ErrCodeInvalid
	}
	if payload.Code != code {
		return ErrCodeInvalid
	}

	// 3) 事务：upsert 旗下账号 + 学校信息覆盖 + 删验证码
	if err := pgsql.DB.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 找现有的（孤儿）旗下账号
		var existing model.User
		findErr := tx.Where("qq_number = ? AND account_type = ? AND status = ?",
			qqNumber, model.AccountTypeQQChild, constant.StatusValid).
			First(&existing).Error

		if findErr == nil {
			// 已有旗下账号——挂上来 + 学校信息覆盖
			parentID := int(user.ID)
			updates := map[string]interface{}{
				"parent_user_id": parentID,
				"school_id":      user.SchoolID, // 严格学校覆盖不严格的
			}
			if err := tx.Model(&existing).Updates(updates).Error; err != nil {
				return fmt.Errorf("挂载旗下账号失败: %w", err)
			}
			return nil
		}

		if !errors.Is(findErr, gorm.ErrRecordNotFound) {
			return fmt.Errorf("查找旗下账号失败: %w", findErr)
		}

		// 没有旗下账号——创建一个空的直接挂上
		parentID := int(user.ID)
		qqNum := qqNumber
		username := "qq" + qqNumber
		newU := &model.User{
			Username:     username,
			Password:     "", // 不可登录
			SchoolID:     user.SchoolID,
			AccountType:  model.AccountTypeQQChild,
			ParentUserID: &parentID,
			QQNumber:     &qqNum,
			Status:       constant.StatusValid,
			Role:         constant.RoleUser,
		}
		if err := tx.Create(newU).Error; err != nil {
			return fmt.Errorf("创建旗下账号失败: %w", err)
		}
		return nil
	}); err != nil {
		return err
	}

	// 4) 删验证码（tx 外面删，redis 失败不回滚 DB）
	_ = commonredis.Client.Del(ctx, qqBindCodeKey(qqNumber)).Err()

	// 5) 给 QQ 发绑定成功通知——让用户感知账号变更，发现异常时能立刻去解绑/找回
	//
	// 通知发不出去**不影响**绑定结果（事务已 commit）：仅 log warn，不重试也不报错。
	// 用 noticeCtx 隔离父 ctx 的 cancel 影响——前端如果 confirm 已经超时取消，
	// 我们仍要把通知发出去；30s 通知超时跟 SendPrivate 内置超时对齐就够了。
	noticeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if botinternal.Default != nil {
		qqInt, _ := strconv.ParseInt(qqNumber, 10, 64)
		text := fmt.Sprintf(
			"【HFUT 校园平台】绑定成功 ✅ 当前 QQ %s 已成功绑定到 app 账号 %q。如非本人操作，请立即在 app 里解绑并修改密码。",
			qqNumber, user.Username,
		)
		if nerr := botinternal.Default.SendPrivate(noticeCtx, qqInt, text); nerr != nil {
			logger.Warnf(ctx, "qq_bind: 绑定成功通知发送失败（不影响绑定结果） qq=%d err=%v", qqInt, nerr)
		}
	}

	return nil
}

// QQUnbindRequestCode 解绑流程第一步：给当前绑定的 QQ 发解绑验证码。
//
// 安全动机：跟绑定流程对称——主账号 token 被盗时，攻击者可以解绑别人 QQ + 自己重新
// 绑（盗取旗下账号的全部数据）。要求"QQ 端收到验证码"才能完成解绑相当于二次身份证明，
// 攻击者拿不到 QQ 私聊也就走不到 confirm。
//
// 流程：
//  1. 校验主账号已绑 QQ（取出来 qq_number，没绑就直接 ErrUserHasNoQQChild）
//  2. 限流（同 user 60s 一次）
//  3. 调 bot 发"解绑确认验证码"私聊
//  4. 存 redis qq_unbind_code:{qq}={code, requesting_user_id}
//
// 注意：这里**不需要再调 CheckFriend**——目标 QQ 既然之前能绑成功就说明是 bot 好友；
// 即便对方后来把 bot 删了好友，发私聊会失败 → 上层报"系统繁忙"，让用户先重新加好友。
func QQUnbindRequestCode(ctx context.Context, callerUserID uint) (ttlSeconds int, err error) {
	// 1) 主账号校验
	user, err := getActiveUser(ctx, callerUserID)
	if err != nil {
		if errors.Is(err, ErrBotUserNotFound) {
			return 0, ErrUserNotFound
		}
		return 0, err
	}
	if user.AccountType != model.AccountTypeNormal {
		return 0, ErrUserNotFound
	}

	// 2) 取当前绑定的旗下账号 QQ；没绑直接拒
	var child model.User
	parentID := int(user.ID)
	findErr := pgsql.DB.WithContext(ctx).
		Where("parent_user_id = ? AND account_type = ? AND status = ?",
			parentID, model.AccountTypeQQChild, constant.StatusValid).
		First(&child).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		return 0, ErrUserHasNoQQChild
	}
	if findErr != nil {
		return 0, fmt.Errorf("查找旗下账号失败: %w", findErr)
	}
	if child.QQNumber == nil || *child.QQNumber == "" {
		// 数据异常：旗下账号没记 qq_number——理论上 P1 之后所有 QQ 旗下号必有
		return 0, ErrUserHasNoQQChild
	}
	qqNumber := *child.QQNumber

	// 3) 限流（同 user 60s 一次；跟绑定独立）
	throttleKey := qqUnbindThrottleKey(user.ID)
	ok, err := commonredis.Client.SetNX(ctx, throttleKey, "1", qqBindThrottleTTL).Result()
	if err != nil {
		return 0, fmt.Errorf("限流锁失败: %w", err)
	}
	if !ok {
		retryAfter := 0
		if d, terr := commonredis.Client.TTL(ctx, throttleKey).Result(); terr == nil && d > 0 {
			retryAfter = int(d.Seconds())
			if retryAfter == 0 {
				retryAfter = 1
			}
		}
		return 0, &ThrottledError{RetryAfterSeconds: retryAfter}
	}

	// 4) 生成 code 存 redis（覆盖旧的）
	if botinternal.Default == nil {
		logger.Warnf(ctx, "qq_unbind: botinternal.Default == nil（BOT_INTERNAL_API_URL 没配 / URL 不合法）")
		return 0, ErrBotUnavailable
	}
	_ = commonredis.Client.Del(ctx, qqUnbindCodeKey(qqNumber)).Err()
	code, err := generateBindCode()
	if err != nil {
		return 0, fmt.Errorf("生成验证码失败: %w", err)
	}
	payload := qqBindCodePayload{
		Code:             code,
		RequestingUserID: user.ID,
		CreatedAt:        time.Now().Unix(),
	}
	raw, _ := json.Marshal(payload)
	if err := commonredis.Client.Set(ctx, qqUnbindCodeKey(qqNumber), raw, qqBindCodeTTL).Err(); err != nil {
		return 0, fmt.Errorf("缓存验证码失败: %w", err)
	}

	// 5) 调 bot 发解绑验证码私聊（文案区别于绑定，让用户清楚是什么操作）
	qqInt, _ := strconv.ParseInt(qqNumber, 10, 64)
	text := fmt.Sprintf("【HFUT 校园平台】您正在**解除**当前 QQ 与 app 账号的绑定，验证码：%s（5 分钟内有效）。如非本人操作请忽略此消息——可能是你的 app 账号被盗。", code)
	if err := botinternal.Default.SendPrivate(ctx, qqInt, text); err != nil {
		logger.Error(ctx, "qq_unbind: 调 bot SendPrivate 失败", zap.Int64("qq", qqInt), zap.Error(err))
		_ = commonredis.Client.Del(ctx, qqUnbindCodeKey(qqNumber)).Err()
		return 0, ErrBotUnavailable
	}

	return int(qqBindCodeTTL.Seconds()), nil
}

// QQUnbindConfirm 解绑流程第二步：校验 code + 真解绑。
//
// 跟绑定 confirm 对称：主账号 + code 都对得上才把 parent_user_id 设回 NULL。
// 旗下账号的所有数据（商品 / 提问 / 订单等）保留，变成"孤儿"等以后再被绑回来。
func QQUnbindConfirm(ctx context.Context, callerUserID uint, code string) error {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return ErrCodeInvalid
	}

	// 1) 主账号校验 + 取当前 QQ
	user, err := getActiveUser(ctx, callerUserID)
	if err != nil {
		if errors.Is(err, ErrBotUserNotFound) {
			return ErrUserNotFound
		}
		return err
	}
	if user.AccountType != model.AccountTypeNormal {
		return ErrUserNotFound
	}

	var child model.User
	parentID := int(user.ID)
	findErr := pgsql.DB.WithContext(ctx).
		Where("parent_user_id = ? AND account_type = ? AND status = ?",
			parentID, model.AccountTypeQQChild, constant.StatusValid).
		First(&child).Error
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		return ErrUserHasNoQQChild
	}
	if findErr != nil {
		return fmt.Errorf("查找旗下账号失败: %w", findErr)
	}
	if child.QQNumber == nil || *child.QQNumber == "" {
		return ErrUserHasNoQQChild
	}
	qqNumber := *child.QQNumber

	// 2) 校验验证码
	raw, err := commonredis.Client.Get(ctx, qqUnbindCodeKey(qqNumber)).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return ErrCodeExpired
		}
		return fmt.Errorf("读验证码失败: %w", err)
	}
	var payload qqBindCodePayload
	if jerr := json.Unmarshal([]byte(raw), &payload); jerr != nil {
		return ErrCodeExpired
	}
	if payload.RequestingUserID != user.ID {
		return ErrCodeInvalid
	}
	if payload.Code != code {
		return ErrCodeInvalid
	}

	// 3) 真解绑：parent_user_id 设回 NULL
	res := pgsql.DB.WithContext(ctx).Model(&model.User{}).
		Where("id = ? AND parent_user_id = ? AND account_type = ?",
			child.ID, parentID, model.AccountTypeQQChild).
		Update("parent_user_id", nil)
	if res.Error != nil {
		return fmt.Errorf("解绑失败: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		// 校验后到 update 之间被并发解绑掉了？当作已经解过
		return ErrUserHasNoQQChild
	}

	// 4) 删 code（事务外，redis 失败不回滚 DB）
	_ = commonredis.Client.Del(ctx, qqUnbindCodeKey(qqNumber)).Err()

	// 5) 给 QQ 发解绑成功通知——让用户确认操作生效，万一是被盗号也能立刻发现
	noticeCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if botinternal.Default != nil {
		qqInt, _ := strconv.ParseInt(qqNumber, 10, 64)
		text := fmt.Sprintf(
			"【HFUT 校园平台】解绑成功 ✅ QQ %s 已与 app 账号 %q 解除绑定，旗下账号变成孤儿状态（商品/提问数据保留）。如非本人操作，请立即修改 app 密码并重新绑定。",
			qqNumber, user.Username,
		)
		if nerr := botinternal.Default.SendPrivate(noticeCtx, qqInt, text); nerr != nil {
			logger.Warnf(ctx, "qq_unbind: 解绑成功通知发送失败（不影响解绑结果） qq=%d err=%v", qqInt, nerr)
		}
	}

	return nil
}

// =============================================================================
// helpers
// =============================================================================

// isValidQQNumber QQ 号校验：5-12 位纯数字。
func isValidQQNumber(s string) bool {
	if len(s) < 5 || len(s) > 12 {
		return false
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// generateBindCode 生成 6 位数字验证码。crypto/rand 防猜。
func generateBindCode() (string, error) {
	const max = 1_000_000
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// hasBoundQQChild 当前主账号是否已经有挂上来的旗下账号。
func hasBoundQQChild(ctx context.Context, parentID uint) (bool, error) {
	var count int64
	pid := int(parentID)
	err := pgsql.DB.WithContext(ctx).Model(&model.User{}).
		Where("parent_user_id = ? AND account_type = ? AND status = ?",
			pid, model.AccountTypeQQChild, constant.StatusValid).
		Count(&count).Error
	return count > 0, err
}
