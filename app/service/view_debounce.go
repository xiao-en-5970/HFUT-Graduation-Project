package service

import (
	"context"
	"fmt"
	"time"

	commonredis "github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/redis"
)

// ViewDebounceWindow 浏览量防抖窗口——同一 viewer 看同一 (extType, extID) 不重复 +1。
//
// 为什么 300s（5 分钟，与 product 校准）：
//   - 压住"点赞 / 收藏 / 编辑后前端立刻 getDetail 刷详情"那波副作用——detailScreen
//     在 toggleLike / toggleCollect 后会 await getDetail 拉一次最新数据，那次 GET 不该计入；
//   - 用户在详情页停留 + 反复切回的典型时长一般在 5 分钟以内；超过 5 分钟再看视为新一次浏览；
//   - 300s 也给客户端在弱网下的重试 / re-fetch burst 留足缓冲。
const ViewDebounceWindow = 300 * time.Second

// ShouldCountView 判定本次详情访问是否计入 view_count。
//
//	(userID, extType, extID) 在 ViewDebounceWindow 内首次访问 → 返回 true，并写一把对应 TTL 的
//	Redis 锁占住后续重复访问；命中已有锁则返回 false。
//
// 不计入的场景：
//   - 未登录（userID == 0）：无法稳定标识 viewer，直接 false
//   - Redis 不可用 / SetNX 报错：保守 false（宁可少计也不重复计）
func ShouldCountView(ctx context.Context, userID uint, extType int, extID uint) bool {
	if userID == 0 {
		return false
	}
	if commonredis.Client == nil {
		return false
	}
	key := fmt.Sprintf("view:debounce:%d:%d:%d", extType, extID, userID)
	ok, err := commonredis.Client.SetNX(ctx, key, "1", ViewDebounceWindow).Result()
	if err != nil {
		return false
	}
	return ok
}
