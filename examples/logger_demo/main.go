/*
* @Author: Lzww0608
* @Date: 2025-05-29 10:00:00
* @LastEditors: Lzww0608
* @LastEditTime: 2025-05-29 10:00:00
* @Description: 日志功能示例程序
 */

package main

import (
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"go.uber.org/zap"
)

func main() {
	// 初始化日志系统
	err := logger.InitLogger()
	if err != nil {
		panic(err)
	}
	defer logger.Close()

	// 演示基本日志功能
	logger.Info("ClixGo 日志系统演示开始")
	logger.Debug("这是一条调试信息")
	logger.Warn("这是一条警告信息")

	// 演示结构化日志
	logger.Info("用户登录",
		zap.String("username", "admin"),
		zap.String("ip", "192.168.1.100"),
		zap.Time("timestamp", time.Now()),
	)

	// 演示错误日志
	logger.Error("数据库连接失败",
		zap.String("database", "mysql"),
		zap.String("host", "localhost:3306"),
		zap.Error(err),
	)

	// 性能测试
	start := time.Now()
	for i := 0; i < 10000; i++ {
		logger.Debug("性能测试日志",
			zap.Int("iteration", i),
			zap.String("test", "performance"),
		)
	}
	duration := time.Since(start)

	logger.Info("性能测试完成",
		zap.Int("logs_count", 10000),
		zap.Duration("duration", duration),
		zap.Float64("logs_per_second", float64(10000)/duration.Seconds()),
	)

	logger.Info("ClixGo 日志系统演示结束")
}
