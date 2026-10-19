/*
@Author: omenkk7
@Date: 2025/10/26
@Description: Scalar API 文档配置
*/

package server

import (
	"html/template"
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// ScalarConfig Scalar 配置
type ScalarConfig struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	SwaggerURL         string `json:"swaggerUrl"`
	Theme              string `json:"theme"`              // "light", "dark", "auto"
	Layout             string `json:"layout"`             // "modern", "classic"
	ShowSidebar        bool   `json:"showSidebar"`
	HideDownloadButton bool   `json:"hideDownloadButton"`
}

// DefaultScalarConfig 默认 Scalar 配置
func DefaultScalarConfig() *ScalarConfig {
	return &ScalarConfig{
		Title:              "ZK Code Arena API",
		Description:        "在线编程竞赛平台 API 文档",
		SwaggerURL:         "/swagger/doc.json",
		Theme:              "auto",
		Layout:             "modern",
		ShowSidebar:        true,
		HideDownloadButton: false,
	}
}

// LoadScalarConfigFromFile 从配置文件加载 Scalar 配置
func LoadScalarConfigFromFile() *ScalarConfig {
	// 这里可以从 conf.Config 加载配置
	// 为了简化，先使用默认配置
	return DefaultScalarConfig()
}

// RegisterScalar 注册 Scalar 路由
func (s *Server) RegisterScalar(r *gin.Engine, config *ScalarConfig) {
	if config == nil {
		config = DefaultScalarConfig()
	}

	// 注册 Scalar 文档路由
	r.GET("/docs", func(c *gin.Context) {
		s.renderScalar(c, config)
	})

	// 可选：注册多个主题的路由
	r.GET("/docs/light", func(c *gin.Context) {
		lightConfig := *config
		lightConfig.Theme = "light"
		s.renderScalar(c, &lightConfig)
	})

	r.GET("/docs/dark", func(c *gin.Context) {
		darkConfig := *config
		darkConfig.Theme = "dark"
		s.renderScalar(c, &darkConfig)
	})

	// 经典布局
	r.GET("/docs/classic", func(c *gin.Context) {
		classicConfig := *config
		classicConfig.Layout = "classic"
		s.renderScalar(c, &classicConfig)
	})
}

// renderScalar 渲染 Scalar 页面
func (s *Server) renderScalar(c *gin.Context, config *ScalarConfig) {
	// 获取模板文件路径
	templatePath := filepath.Join("pkg", "app", "api-server", "server", "templates", "scalar.html")
	
	// 解析模板
	tmpl, err := template.ParseFiles(templatePath)
	if err != nil {
		// 如果模板文件不存在，使用内嵌的 HTML
		s.renderScalarInline(c, config)
		return
	}

	// 设置响应头
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)

	// 渲染模板
	if err := tmpl.Execute(c.Writer, config); err != nil {
		s.lg.Errorf("渲染 Scalar 模板失败: %v", err)
		c.String(http.StatusInternalServerError, "渲染文档页面失败")
		return
	}
}

// renderScalarInline 内嵌渲染 Scalar 页面（备用方案）
func (s *Server) renderScalarInline(c *gin.Context, config *ScalarConfig) {
	html := `<!DOCTYPE html>
<html>
<head>
    <title>` + config.Title + ` - API Documentation</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1" />
    <style>
        body {
            margin: 0;
            padding: 0;
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
        }
        .loading {
            display: flex;
            justify-content: center;
            align-items: center;
            height: 100vh;
            font-size: 18px;
            color: #666;
        }
    </style>
</head>
<body>
    <div class="loading">正在加载 API 文档...</div>
    <script
        id="api-reference"
        data-url="` + config.SwaggerURL + `"
        data-configuration='{
            "theme": "` + config.Theme + `",
            "layout": "` + config.Layout + `",
            "showSidebar": ` + boolToString(config.ShowSidebar) + `,
            "hideDownloadButton": ` + boolToString(config.HideDownloadButton) + `,
            "searchHotKey": "k",
            "metaData": {
                "title": "` + config.Title + `",
                "description": "` + config.Description + `"
            },
            "customCss": ".scalar-app { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif; }",
            "hideModels": false,
            "hideAuthentication": false,
            "hideTestRequestButton": false
        }'></script>
    <script src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
    <script>
        // 隐藏加载提示
        document.addEventListener('DOMContentLoaded', function() {
            setTimeout(function() {
                const loading = document.querySelector('.loading');
                if (loading) {
                    loading.style.display = 'none';
                }
            }, 1000);
        });
    </script>
</body>
</html>`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, html)
}

// boolToString 将布尔值转换为字符串
func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// ScalarMiddleware 中间件，用于自定义 Scalar 行为
func ScalarMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置缓存头
		c.Header("Cache-Control", "public, max-age=3600")
		c.Next()
	}
}
