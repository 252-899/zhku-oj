/*
@Author: AI Generated
@Date: 2026/03/06
@Name: role.go
@Description: 角色权限中间件
*/

package middleware

import (
	"zk-code-arena-server/pkg/models"
	"zk-code-arena-server/pkg/utils"

	"github.com/gin-gonic/gin"
)

// RequireRole 要求特定角色的中间件
func RequireRole(allowedRoles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户角色
		roleInterface, exists := c.Get("role")
		if !exists {
			utils.UnauthorizedResponse(c, "未找到用户角色信息")
			c.Abort()
			return
		}

		userRole := models.UserRole(roleInterface.(string))

		// 检查角色是否在允许列表中
		allowed := false
		for _, role := range allowedRoles {
			if userRole == role {
				allowed = true
				break
			}
		}

		if !allowed {
			utils.ForbiddenResponse(c, "权限不足，该功能仅限老师和管理员使用")
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireTeacherOrAdmin 要求老师或管理员角色
func RequireTeacherOrAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleTeacher, models.RoleAdmin)
}

// RequireAdmin 要求管理员角色
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.RoleAdmin)
}
