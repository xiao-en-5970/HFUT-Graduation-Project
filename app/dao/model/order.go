package model

import (
	"time"

	"github.com/lib/pq"
)

// Order 订单表（平台不经手资金）
type Order struct {
	ID                 uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID             *int           `gorm:"column:user_id;index" json:"user_id"`                                                    // 买家用户ID
	GoodsID            *int           `gorm:"column:goods_id;index" json:"goods_id"`                                                  // 商品ID
	Status             int16          `gorm:"type:smallint;not null;default:1" json:"status"`                                         // 1:正常 2:禁用
	OrderStatus        int16          `gorm:"column:order_status;type:smallint;not null;default:1" json:"order_status"`               // 见 constant 订单状态
	ReceiverAddr       string         `gorm:"column:receiver_addr;type:varchar(512)" json:"receiver_addr"`                            // 收货文字地址（口头描述）
	ReceiverLat        *float64       `gorm:"column:receiver_lat" json:"receiver_lat,omitempty"`                                      // 收货地图选点纬度 WGS84
	ReceiverLng        *float64       `gorm:"column:receiver_lng" json:"receiver_lng,omitempty"`                                      // 收货地图选点经度 WGS84
	SenderAddr         string         `gorm:"column:sender_addr;type:varchar(512)" json:"sender_addr"`                                // 发货文字地址
	SenderLat          *float64       `gorm:"column:sender_lat" json:"sender_lat,omitempty"`                                          // 发货地图选点纬度 WGS84
	SenderLng          *float64       `gorm:"column:sender_lng" json:"sender_lng,omitempty"`                                          // 发货地图选点经度 WGS84
	DistanceMeters     *int           `gorm:"column:distance_meters" json:"distance_meters,omitempty"`                                // 发货地与收货地步行路径距离（米），GraphHopper foot
	BuyerAgreedAt      *time.Time     `gorm:"column:buyer_agreed_at" json:"buyer_agreed_at,omitempty"`                                // 买方下单时间
	SellerAgreedAt     *time.Time     `gorm:"column:seller_agreed_at" json:"seller_agreed_at,omitempty"`                              // 卖方确认收款时间
	DeliveryImages     pq.StringArray `gorm:"column:delivery_images;type:varchar(2048)[]" json:"delivery_images,omitempty"`           // 卖方送达凭证图
	BuyerConfirmImages pq.StringArray `gorm:"column:buyer_confirm_images;type:varchar(2048)[]" json:"buyer_confirm_images,omitempty"` // 买方确认收货时附加图
	CompletedAt        *time.Time     `gorm:"column:completed_at" json:"completed_at,omitempty"`                                      // 订单完成时间
	CreatedAt          time.Time      `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt          time.Time      `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Order) TableName() string {
	return "orders"
}
