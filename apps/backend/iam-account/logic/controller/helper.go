// helper.go 提供 controller 层公用工具：
//   - respondError：把业务错误 errs.Err 统一映射成 HTTP 响应
//   - currentUserID：从请求头 X-User-Id 提取当前 user id
//
// X-User-Id 由上游（iam-authn 或 API 网关）在校验完 token 后注入。
// 当前脚手架尚未接入 authn，允许 dev 时前端直接传该 Header 以便端到端联调。
package controller

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/zhifenghao123/how-dev-iam/iam-account/logic/common/pkg/errs"
)

// respondError 把 error 转成统一响应格式。
func respondError(c *gin.Context, err error) {
	if e, ok := errs.As(err); ok {
		c.JSON(e.HTTPStatus, gin.H{"code": e.Code, "message": e.Message})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{
		"code":    errs.ErrInternal.Code,
		"message": err.Error(),
	})
}

// respondOK 统一成功响应。
func respondOK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, gin.H{"code": "OK", "data": data})
}

// currentUserID 从 X-User-Id 头提取当前登录用户 id；未登录返回 0。
func currentUserID(c *gin.Context) int64 {
	raw := c.GetHeader("X-User-Id")
	if raw == "" {
		return 0
	}
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return 0
	}
	return id
}

// clientIP 返回客户端 IP。
func clientIP(c *gin.Context) string {
	return c.ClientIP()
}
