package snowflake

import (
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

func initNode() {
	var err error
	node, err = snowflake.NewNode(1)
	if err != nil {
		panic(err)
	}
}

// NextID 生成全局唯一 ID（雪花算法）
func NextID() int64 {
	once.Do(initNode)
	return node.Generate().Int64()
}

// NextIDString 生成全局唯一 ID 字符串
func NextIDString() string {
	once.Do(initNode)
	return node.Generate().String()
}
