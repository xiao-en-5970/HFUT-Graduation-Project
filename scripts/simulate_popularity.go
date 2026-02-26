// 模拟发帖热度随时间变化
// 运行: go run scripts/simulate_popularity.go
package main

import "fmt"

func main() {
	const (
		wCollect  = 10
		wLike     = 5
		wView     = 1
		decayDays = 90.0
	)

	// 模拟场景：帖子发布后固定互动量（假设发布当天即达到）
	collects := 5
	likes := 20
	views := 300
	interaction := collects*wCollect + likes*wLike + views*wView

	fmt.Println("=== 发帖热度变化模拟 ===")
	fmt.Printf("互动量(固定): 收藏=%d 点赞=%d 浏览=%d\n", collects, likes, views)
	fmt.Printf("互动分基数: %d×10 + %d×5 + %d×1 = %d\n", collects, likes, views, interaction)
	fmt.Printf("权重: 收藏×%d 点赞×%d 浏览×%d | 互动衰减半衰期=%.0f天\n\n", wCollect, wLike, wView, decayDays)

	fmt.Println("| 距今天数 | 互动衰减系数 | 有效互动分（总热度） |")
	fmt.Println("|---------|-------------|---------------------|")

	daysList := []float64{0, 1, 3, 7, 14, 30, 60, 90, 180, 365}
	for _, days := range daysList {
		interDecay := 1.0 / (1 + days/decayDays)
		total := float64(interaction) * interDecay
		fmt.Printf("| %6.0f天 |     %6.2f%% |  %8.1f |\n", days, interDecay*100, total)
	}

	fmt.Println("\n--- 场景对比：不同互动量的帖子 ---")
	scenarios := []struct {
		name    string
		c, l, v int
	}{
		{"冷门帖", 1, 5, 50},
		{"普通帖", 5, 20, 300},
		{"热门帖", 20, 100, 2000},
	}
	for _, s := range scenarios {
		ia := s.c*wCollect + s.l*wLike + s.v*wView
		fmt.Printf("\n【%s】收藏=%d 点赞=%d 浏览=%d, 互动分=%d\n", s.name, s.c, s.l, s.v, ia)
		for _, days := range []float64{0, 7, 30, 90} {
			interDecay := 1.0 / (1 + days/decayDays)
			total := float64(ia) * interDecay
			fmt.Printf("  第%.0f天: 衰减%.2f → 热度 %.1f\n", days, interDecay, total)
		}
	}
}
