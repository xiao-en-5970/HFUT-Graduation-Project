package constant

// 订单业务状态（平台不经手资金：线下沟通与付款）
// 流程：磋商 → 买方表示已付款并下单 → 卖方确认收款 → 派送/自提 → 买方确认收货 → 完成
const (
	OrderStatusPendingBuyerPayment       int16 = 1 // 待买方付款下单：可聊天，未声明已付款
	OrderStatusAwaitSellerPaymentConfirm int16 = 2 // 待卖方确认收款（买方已表示已付款并下单）
	OrderStatusFulfillment               int16 = 3 // 履约中：送货上门=正在派送；自提=待买方自提
	OrderStatusPendingBuyerConfirm       int16 = 4 // 待买方确认收货（已送达或在线/自提待确认）
	OrderStatusCompleted                 int16 = 5 // 已完成：买方确认收货，库存已扣减
	OrderStatusCancelled                 int16 = 6 // 已取消
)

// 订单内聊天消息类型
const (
	OrderMsgTypeText  int16 = 1
	OrderMsgTypeImage int16 = 2
)
