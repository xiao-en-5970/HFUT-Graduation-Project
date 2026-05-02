// Package scheduler 后台轻量定时任务。
//
// 本包故意不引入 robfig/cron 这类依赖：目前全部任务都是固定周期的幂等 SQL UPDATE，
// 直接 time.Ticker + goroutine 就足够，少一个依赖、启动更快。
package scheduler

import (
	"context"
	"time"

	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/dao"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/common/logger"
	"go.uber.org/zap"
)

// goodsAutoOffShelfInterval 扫描间隔。
// 用户约定 5 分钟：实时性对"到点下架"足够，DB 压力最小。
const goodsAutoOffShelfInterval = 5 * time.Minute

// StartGoodsAutoOffShelf 启动"到期自动下架"后台任务。
// 调用方应传入一个长生命周期 context（通常是进程 rootCtx）；context 取消时协程退出。
// 函数本身不阻塞；返回前会立刻触发一次扫描，避免服务刚重启时积压的过期数据拖到下一个周期。
func StartGoodsAutoOffShelf(ctx context.Context) {
	go func() {
		logger.Info(ctx, "scheduler: goods auto-offshelf started",
			zap.Duration("interval", goodsAutoOffShelfInterval),
		)
		// 启动后先跑一次（给重启后的积压过期项一个立即清理的机会）
		runGoodsAutoOffShelfOnce(ctx)

		t := time.NewTicker(goodsAutoOffShelfInterval)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Info(ctx, "scheduler: goods auto-offshelf stopping", zap.Error(ctx.Err()))
				return
			case <-t.C:
				runGoodsAutoOffShelfOnce(ctx)
			}
		}
	}()
}

// runGoodsAutoOffShelfOnce 单次扫描；错误仅写日志，不 panic。
// 每次调用带 30s 超时，避免单次 DB 卡住导致 ticker 堆积。
func runGoodsAutoOffShelfOnce(parent context.Context) {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	n, err := dao.Good().AutoOffShelfExpired(ctx)
	if err != nil {
		logger.Warn(ctx, "scheduler: goods auto-offshelf sweep failed", zap.Error(err))
		return
	}
	if n > 0 {
		logger.Info(ctx, "scheduler: goods auto-offshelf sweep ok", zap.Int64("affected", n))
	}
}
