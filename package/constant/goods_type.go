package constant

// 商品交易/履约类型（影响订单流程与是否计算送货距离）
const (
	GoodsTypeDelivery int16 = 1 // 送货上门：买卖方地址 + 可选步行距离
	GoodsTypePickup   int16 = 2 // 自提：买方到约定自提点提货，收货地址一般为卖方设置的自提地址
	GoodsTypeOnline   int16 = 3 // 在线商品：无实体派送；双方同意后即视为卖方已发货，进入待买方确认收货
)

// GoodsTypeLabel 供 API 展示
func GoodsTypeLabel(t int16) string {
	switch t {
	case GoodsTypeDelivery:
		return "送货上门"
	case GoodsTypePickup:
		return "自提"
	case GoodsTypeOnline:
		return "在线商品"
	default:
		return "未知"
	}
}
