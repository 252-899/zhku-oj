/*
@Author: urmsone urmsone@163.com
@Date: 2025/1/24 09:49
@Name: main.go
@Description: ZK Code Arena Server - 仲恺农业工程学院在线刷题平台
*/

// @title           ZK Code Arena API
// @version         1.0
// @description     ZK Code Arena 在线编程平台 REST API 文档
// @description     提供用户管理、题库管理、班级课程管理、代码评测等功能
// @termsOfService  http://localhost/terms/

// @contact.name   API Support
// @contact.url    http://localhost/support
// @contact.email  support@zkcodearena.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      118.195.215.81
// @BasePath  /oj/api/v1

// @schemes   http

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Type "Bearer" followed by a space and JWT token.

package main

import (
	"fmt"
	"os"
	_ "zk-code-arena-server/docs"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "zk-code-arena-server",
	Short: "ZK Code Arena Server - 仲恺农业工程学院在线刷题平台",
	Long: `ZK Code Arena Server 是一个基于 Golang + Gin + MongoDB + go-judge 构建的
现代化在线刷题平台，为仲恺农业工程学院学生提供专业的编程练习环境。

功能特性：
- 用户管理和权限控制
- 题库管理和分类浏览
- 班级和课程管理
- 作业系统和自动评测
- 竞赛系统和实时排名
- 多语言代码评测支持`,
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
