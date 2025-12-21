# Controller 更新指南

本文档说明如何将 controller 中的错误响应更新为使用 `package/reply` 和 `package/errcode`。

## 更新步骤

### 1. 更新 import

将：
```go
import (
	"net/http"
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/response"
)
```

替换为：
```go
import (
	"strconv"
	"github.com/gin-gonic/gin"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/middleware"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/service"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/app/vo/request"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/errcode"
	"github.com/xiao-en-5970/HFUT-Graduation-Project/package/reply"
)
```

### 2. 替换响应模式

#### 参数错误
```go
// 旧代码
if err := ctx.ShouldBindJSON(&req); err != nil {
	ctx.JSON(http.StatusBadRequest, response.Response{
		Code:    400,
		Message: "参数错误: " + err.Error(),
	})
	return
}

// 新代码
if err := ctx.ShouldBindJSON(&req); err != nil {
	reply.ReplyInvalidParams(ctx, err)
	return
}
```

#### 成功响应（无数据）
```go
// 旧代码
ctx.JSON(http.StatusOK, response.Response{
	Code:    200,
	Message: "操作成功",
})

// 新代码
reply.ReplyOKWithMessage(ctx, "操作成功")
```

#### 成功响应（带数据）
```go
// 旧代码
ctx.JSON(http.StatusOK, response.Response{
	Code:    200,
	Message: "获取成功",
	Data:    result,
})

// 新代码
reply.ReplyOKWithData(ctx, result)
```

#### 成功响应（自定义消息和数据）
```go
// 旧代码
ctx.JSON(http.StatusOK, response.Response{
	Code:    200,
	Message: "创建成功",
	Data:    data,
})

// 新代码
reply.ReplyOKWithMessageAndData(ctx, "创建成功", data)
```

#### 资源不存在
```go
// 旧代码
ctx.JSON(http.StatusNotFound, response.Response{
	Code:    404,
	Message: "资源不存在",
})

// 新代码（使用对应的错误码）
reply.ReplyNotFound(ctx, errcode.ErrUserNotFound)  // 用户模块
reply.ReplyNotFound(ctx, errcode.ErrArticleNotFound)  // 文章模块
reply.ReplyNotFound(ctx, errcode.ErrGoodNotFound)  // 商品模块
// ... 其他模块类似
```

#### 业务错误
```go
// 旧代码
ctx.JSON(http.StatusBadRequest, response.Response{
	Code:    400,
	Message: err.Error(),
})

// 新代码
reply.ReplyErrWithMessage(ctx, errcode.ErrUserAlreadyExists, err.Error())
// 或使用对应的错误码
```

#### 无权限
```go
// 旧代码
ctx.JSON(http.StatusForbidden, response.Response{
	Code:    403,
	Message: "无权限访问此资源",
})

// 新代码
reply.ReplyForbidden(ctx)
```

#### 未认证
```go
// 旧代码
ctx.JSON(http.StatusUnauthorized, response.Response{
	Code:    401,
	Message: "未认证",
})

// 新代码
reply.ReplyUnauthorized(ctx)
```

#### 服务器内部错误
```go
// 旧代码
ctx.JSON(http.StatusInternalServerError, response.Response{
	Code:    500,
	Message: err.Error(),
})

// 新代码
reply.ReplyInternalError(ctx, err)
```

## 错误码映射

### 用户模块 (2000-2099)
- `errcode.ErrUserNotFound` - 用户不存在
- `errcode.ErrUserAlreadyExists` - 用户已存在
- `errcode.ErrUserPasswordWrong` - 密码错误
- `errcode.ErrUserNoPermission` - 用户无权限

### 文章模块 (2100-2199)
- `errcode.ErrArticleNotFound` - 文章不存在
- `errcode.ErrArticleNoPermission` - 无权限操作文章
- `errcode.ErrArticleCreateFailed` - 文章创建失败
- `errcode.ErrArticleUpdateFailed` - 文章更新失败
- `errcode.ErrArticleDeleteFailed` - 文章删除失败

### 商品模块 (2400-2499)
- `errcode.ErrGoodNotFound` - 商品不存在
- `errcode.ErrGoodNoPermission` - 无权限操作商品
- `errcode.ErrGoodOutOfStock` - 商品库存不足
- `errcode.ErrGoodNotOnSale` - 商品不在售

### 收藏模块 (2500-2599)
- `errcode.ErrCollectNotFound` - 收藏不存在
- `errcode.ErrCollectAlreadyExists` - 已收藏
- `errcode.ErrCollectNoPermission` - 无权限操作收藏

### 关注模块 (2600-2699)
- `errcode.ErrFollowNotFound` - 关注关系不存在
- `errcode.ErrFollowAlreadyExists` - 已关注
- `errcode.ErrFollowSelf` - 不能关注自己
- `errcode.ErrFollowNoPermission` - 无权限操作关注

### 订单模块 (2700-2799)
- `errcode.ErrOrderNotFound` - 订单不存在
- `errcode.ErrOrderNoPermission` - 无权限操作订单
- `errcode.ErrOrderCreateFailed` - 订单创建失败
- `errcode.ErrOrderUpdateFailed` - 订单更新失败

## 已更新的文件

- ✅ `app/controller/user.go` - 已完成
- ✅ `app/controller/article.go` - 已完成
- ✅ `app/controller/order.go` - 已完成
- ✅ `app/controller/collect.go` - 已完成
- ✅ `app/controller/follow.go` - 已完成
- ✅ `app/controller/comment.go` - 已完成
- ✅ `app/controller/school.go` - 已完成
- ✅ `app/controller/like.go` - 已完成
- ✅ `app/controller/good.go` - 已完成
- ✅ `app/controller/tag.go` - 已完成

**所有 controller 文件已全部更新完成！**

## 注意事项

1. 删除 `net/http` import（如果不再需要）
2. 删除 `app/vo/response` import（如果不再直接使用）
3. 根据业务逻辑选择合适的错误码
4. 保持错误消息的一致性

