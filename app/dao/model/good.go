package model

import (
	"time"

	"github.com/lib/pq"
)

// Good 商品表
type Good struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID     *int           `gorm:"column:user_id;index" json:"user_id"`                                  // 用户ID
	SchoolID   *int           `gorm:"column:school_id;index" json:"school_id"`                              // 学校ID
	Title      string         `gorm:"type:varchar(255);not null" json:"title"`                              // 商品名称
	Images     pq.StringArray `gorm:"type:varchar(255)[]" json:"images"`                                    // 图片数组
	Content    string         `gorm:"type:text;not null" json:"content"`                                    // 商品内容
	Status     int16          `gorm:"type:smallint;not null;default:1" json:"status"`                       // 1:正常 2:禁用
	GoodStatus int            `gorm:"column:good_status;type:int;not null;default:1" json:"good_status"`    // 1:在售 2:下架 3:已售出
	GoodsType  int16          `gorm:"column:goods_type;type:smallint;not null;default:1" json:"goods_type"` // 1送货上门 2自提 3在线商品
	GoodsAddr  string         `gorm:"column:goods_addr;type:varchar(512)" json:"goods_addr,omitempty"`      // 商品地址：发货地/自提点合一，用于默认卖方发货地址与自提说明
	PickupAddr string         `gorm:"column:pickup_addr;type:varchar(512)" json:"pickup_addr,omitempty"`    // 与 goods_addr 同步，兼容旧字段
	GoodsLat   *float64       `gorm:"column:goods_lat" json:"goods_lat,omitempty"`                          // 商品位置纬度 WGS84，与发货地一致
	GoodsLng   *float64       `gorm:"column:goods_lng" json:"goods_lng,omitempty"`                          // 商品位置经度 WGS84
	// Price 商品价格（单位：分）。
	// 配合 Negotiable 字段使用：Negotiable=true 时 Price 字段被业务层忽略、前端展示"面议"。
	// 历史数据：Negotiable=false（默认）时按原 Price 字段展示。
	Price        int  `gorm:"type:integer;not null;default:0" json:"price"`
	Negotiable   bool `gorm:"column:negotiable;type:boolean;not null;default:false" json:"negotiable"`   // true=面议，价格字段被忽略
	MarkedPrice  int  `gorm:"column:marked_price;type:integer;not null;default:0" json:"marked_price"`   // 标价，单位分
	Stock        int  `gorm:"type:integer;not null;default:0" json:"stock"`                              // 库存数量
	ImageCount   int  `gorm:"column:image_count;type:integer;not null;default:0" json:"image_count"`     // 图片数量
	LikeCount    int  `gorm:"column:like_count;type:integer;not null;default:0" json:"like_count"`       // 点赞次数
	CollectCount int  `gorm:"column:collect_count;type:integer;not null;default:0" json:"collect_count"` // 收藏次数
	StartTime    int  `gorm:"column:start_time;type:integer;not null;default:0" json:"start_time"`       // 开始时间
	EndTime      int  `gorm:"column:end_time;type:integer;not null;default:0" json:"end_time"`           // 结束时间
	// 类别与收款码：二手买卖 (1) 才有意义传收款码图，有偿求助 (2) 时 PaymentQRURL 固定为空
	GoodsCategory int16  `gorm:"column:goods_category;type:smallint;not null;default:1" json:"goods_category"` // 1 二手买卖 / 2 有偿求助
	PaymentQRURL  string `gorm:"column:payment_qr_url;type:varchar(255);not null;default:''" json:"payment_qr_url,omitempty"`
	// 定时下架：仅 HasDeadline=true 时 Deadline 有意义；过期后 cron 会把 good_status 置为 2
	HasDeadline bool       `gorm:"column:has_deadline;type:boolean;not null;default:false" json:"has_deadline"`
	Deadline    *time.Time `gorm:"column:deadline" json:"deadline,omitempty"`
	// CreatedInGroupID bot 通过 QQ 群上架本商品时所在的群号；非 bot 路径（管理员 / app 用户）
	// 上架的商品 NULL。用于 RequestOffShelfFromOrphan 定位 "在哪个群 @ 卖家"——比
	// users.created_in_group_id 更精准（用户跨群活动时不同群发不同商品）。
	// 详见 QQ-bot/skill/bot/orphan.md "请求下架" 段 + migrate_goods_created_in_group.sql。
	CreatedInGroupID *int64    `gorm:"column:created_in_group_id" json:"created_in_group_id,omitempty"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt        time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (Good) TableName() string {
	return "goods"
}
