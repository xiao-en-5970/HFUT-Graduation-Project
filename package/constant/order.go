package constant

// 订单业务状态（平台不经手资金，线下沟通与付款）
const (
	OrderStatusPendingIntent       int16 = 1 // 待下单：买家「我想要」后，双方可聊天
	OrderStatusDelivering          int16 = 2 // 正在派送：双方同意后卖家开始送货
	OrderStatusPendingBuyerConfirm int16 = 3 // 待买方确认收货：卖家已确认送达
	OrderStatusCompleted           int16 = 4 // 已完成：买方确认收货，库存已扣减
	OrderStatusCancelled           int16 = 5 // 已取消
)

// 订单内聊天消息类型
const (
	OrderMsgTypeText  int16 = 1
	OrderMsgTypeImage int16 = 2
)
