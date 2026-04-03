package constant

// 订单业务状态（平台不经手资金；下单即进入待卖方确认收款，卖方不确认则视为未成交）
// 流程：下单 → 卖方确认收款 → 派送/自提 → 买方确认收货 → 完成
const (
	OrderStatusAwaitSellerPaymentConfirm int16 = 1 // 待卖方确认收款（买方已下单，契约上视为已付款意向）
	OrderStatusFulfillment               int16 = 2 // 履约中：送货上门=正在派送；自提=待买方自提
	OrderStatusPendingBuyerConfirm       int16 = 3 // 待买方确认收货
	OrderStatusCompleted                 int16 = 4 // 已完成：买方确认收货，库存已扣减
	OrderStatusCancelled                 int16 = 5 // 已取消
)

// 订单内聊天消息类型
const (
	OrderMsgTypeText    int16 = 1
	OrderMsgTypeImage   int16 = 2
	OrderMsgTypeOfficial int16 = 3 // 官方系统通知（仅服务端写入）
)

// OrderOfficialUsername 订单聊天官方通知所用系统用户（迁移脚本插入，status=禁用不可登录）
const OrderOfficialUsername = "__order_official__"
