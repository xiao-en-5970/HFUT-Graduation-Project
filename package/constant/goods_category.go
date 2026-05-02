package constant

// 商品类别（与 GoodsType 履约方式正交）
// 新增维度：允许把"有偿求助"与普通"二手买卖"区分开，但仍共用同一条交易流程。
// 有偿求助语义：发布者 = 付款方；接单者 = 收款方。
const (
	GoodsCategoryNormal int16 = 1 // 二手买卖：发布者是卖家，收款方
	GoodsCategoryHelp   int16 = 2 // 有偿求助：发布者是买家，付款方；接单者完成任务获取报酬
)

// GoodsCategoryLabel 供 API 展示
func GoodsCategoryLabel(c int16) string {
	switch c {
	case GoodsCategoryNormal:
		return "二手买卖"
	case GoodsCategoryHelp:
		return "有偿求助"
	default:
		return "未知"
	}
}

// IsValidGoodsCategory 接口层校验入参
func IsValidGoodsCategory(c int16) bool {
	return c == GoodsCategoryNormal || c == GoodsCategoryHelp
}
