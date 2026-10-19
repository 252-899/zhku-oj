/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/25 13:09
@Name: server.go
@Description: 服务器主文件

@title ZK Code Arena API
@version 1.0
@description ZK Code Arena 在线编程平台API文档
@host 118.195.215.81
@BasePath /oj/api/v1
@schemes http

@securityDefinitions.apikey BearerAuth
@in header
@name Authorization
@description 输入"Bearer " + JWT Token
*/

package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"
	"zk-code-arena-server/conf"
	"zk-code-arena-server/pkg/app/api-server/service"

	"zk-code-arena-server/pkg/utils"
	"zk-code-arena-server/pkg/utils/middleware"
	wsManager "zk-code-arena-server/pkg/utils/websocket"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"

	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Server struct {
	lg     logrus.FieldLogger
	app    *gin.Engine
	svc    *service.Service
	opts   *CmdOptions
	stopCh <-chan struct{}
}

func NewServer(lg logrus.FieldLogger, svc *service.Service, opts *CmdOptions, stopCh <-chan struct{}) *Server {
	app := gin.Default()
	app.Use(middleware.CorsHandler())                          // 设置 CORS
	app.Use(middleware.LoggerHandler(lg, []string{"/health"})) // 日志中间件
	app.Use(gin.Recovery())                                    // panic 恢复

	// 设置 Gin 模式
	gin.SetMode(gin.DebugMode)

	return &Server{
		lg:     lg,
		app:    app,
		svc:    svc,
		opts:   opts,
		stopCh: stopCh,
	}
}

func (s *Server) Init() {
	s.RegisterRoutes()

	// 启动WebSocket连接管理器
	go func() {
		wsManager := wsManager.GlobalConnectionManager
		wsManager.Start(context.Background())
	}()

	// RSS 定时同步（每小时）
	if s.svc.RSSService != nil {
		go s.svc.RSSService.StartScheduler(context.Background(), s.stopCh)
	}
}

// GetApp 获取 Gin 引擎实例（用于测试）
func (s *Server) GetApp() *gin.Engine {
	return s.app
}

// RegisterRoutes 注册路由
func (s *Server) RegisterRoutes() {
	// 健康检查
	s.app.GET("/health", s.HealthCheck)

	// Swagger 文档路由（保留兼容性）
	s.app.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Scalar 文档路由（现代化 API 文档）
	s.RegisterScalar(s.app, nil)

	// API 路由组
	v1 := s.app.Group("/api/v1")
	{
		// 用户相关路由
		s.RegisterUser(v1)
		// 题目相关路由
		s.RegisterProblem(v1)
		// 提交相关路由
		s.RegisterSubmit(v1)

		// 测试用例相关路由
		s.RegisterTestCase(v1)
		// 班级相关路由
		s.RegisterClazz(v1)
		//课程相关路由
		s.RegisterCourse(v1)
		// 课程选择题相关路由
		s.RegisterChoiceQuestion(v1)
		// 课程填空题相关路由
		s.RegisterBlankQuestion(v1)
		// 混合题型统一提交路由
		s.RegisterContentSubmission(v1)
		// 统计相关路由
		s.RegisterStatistics(v1)
		// 题目统计相关路由（公开）
		s.RegisterProblemStats(v1)
		// WebSocket相关路由
		s.RegisterWebSocket(v1)
		// 标签相关路由
		s.RegisterTag(v1)
		// 通用聊天助手相关路由
		s.RegisterAIAgent(v1)
		// 教师侧 AI Copilot 业务入口
		s.RegisterTeacherAI(v1)
		// RSS 资讯聚合（登录用户可读，管理端为教师/管理员）
		s.RegisterRSS(v1)
	}
}

// HealthCheck 健康检查
func (s *Server) HealthCheck(c *gin.Context) {
	utils.SuccessResponse(c, gin.H{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "zk-code-arena-server",
	})
}

func (s *Server) Run() error {
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", s.opts.Host, s.opts.Port),
		Handler:      s.app,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		select {
		case <-s.stopCh:
			graceful := conf.Config.App.GracefulTime
			if graceful <= 0 {
				graceful = 10
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(graceful)*time.Second)
			defer cancel()
			// 优雅关闭服务器
			s.lg.Infoln("正在关闭服务器...")
			if err := srv.Shutdown(ctx); err != nil {
				s.lg.Errorf("服务器关闭错误: %v", err)
			}
			return
		}
	}()

	if err := srv.ListenAndServe(); err != nil {
		if errors.Is(err, http.ErrServerClosed) {
			// 主动关闭服务器
			s.lg.Println("HTTP 服务器已关闭")
			return err
		}
		s.lg.Errorf("服务器启动失败: %s", err.Error())
		return err
	}
	return nil
}

func (s *Server) Shutdown(ctx context.Context) {
	s.lg.Info("正在关闭判题服务...")
	if err := s.svc.JudgeService.Stop(ctx); err != nil {
		s.lg.Errorf("判题服务关闭失败: %v", err)
	} else {
		s.lg.Info("判题服务已关闭")
	}

	s.lg.Info("正在关闭数据库连接...")
	s.svc.Close(ctx)
	s.lg.Info("服务器已完全关闭")
}

type CmdOptions struct {
	Host string
	Port string
}

func NewCmdOptions(host, port string) *CmdOptions {
	return &CmdOptions{
		Host: host,
		Port: port,
	}
}
