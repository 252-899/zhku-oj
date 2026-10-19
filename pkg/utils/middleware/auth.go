/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 17:38
@Name: auth.go
@Description: JWT 认证中间件
*/

package middleware

import (
	"strings"
	"time"
	"zk-code-arena-server/conf"
	"zk-code-arena-server/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
)

// Claims JWT 声明
type Claims struct {
	UserID   string `json:"user_id"`
	Username string `json:"username"`
	Role     string `json:"role"`
	jwt.StandardClaims
}

// GenerateToken 生成 JWT Token
func GenerateToken(userID, username, role string) (string, error) {
	claims := Claims{
		UserID:   userID,
		Username: username,
		Role:     role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(conf.Config.JWT.ExpireTime).Unix(),
			IssuedAt:  time.Now().Unix(),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(conf.Config.JWT.Secret))
}

// ParseToken 解析 JWT Token
func ParseToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(conf.Config.JWT.Secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrSignatureInvalid
}

// JWTMiddleware JWT 认证中间件
func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			utils.UnauthorizedResponse(c, "缺少认证令牌")
			c.Abort()
			return
		}

		// 检查 Bearer 前缀
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		if tokenString == authHeader {
			utils.UnauthorizedResponse(c, "认证令牌格式错误")
			c.Abort()
			return
		}

		// 解析 Token
		claims, err := ParseToken(tokenString)
		if err != nil {
			utils.UnauthorizedResponse(c, "认证令牌无效")
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)
		c.Set("claims", claims)

		c.Next()
	}
}
