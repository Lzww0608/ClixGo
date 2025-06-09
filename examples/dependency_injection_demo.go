/*
 * @Author: Lzww0608
 * @Date: 2025-06-09 16:45:00
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-06-09 16:45:00
 * @Description: 依赖注入容器使用演示
 */

package main

import (
	"fmt"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/container"
	"github.com/Lzww0608/ClixGo/pkg/interfaces"
	"go.uber.org/zap"
)

// 简单的日志服务实现
type SimpleLogger struct {
	logger *zap.Logger
}

func NewSimpleLogger() *SimpleLogger {
	logger, _ := zap.NewProduction()
	return &SimpleLogger{logger: logger}
}

func (s *SimpleLogger) Info(msg string, fields ...zap.Field) {
	s.logger.Info(msg, fields...)
}

func (s *SimpleLogger) Error(msg string, fields ...zap.Field) {
	s.logger.Error(msg, fields...)
}

func (s *SimpleLogger) Debug(msg string, fields ...zap.Field) {
	s.logger.Debug(msg, fields...)
}

func (s *SimpleLogger) Warn(msg string, fields ...zap.Field) {
	s.logger.Warn(msg, fields...)
}

func (s *SimpleLogger) Fatal(msg string, fields ...zap.Field) {
	s.logger.Fatal(msg, fields...)
}

func (s *SimpleLogger) With(fields ...zap.Field) interfaces.Logger {
	return &SimpleLogger{logger: s.logger.With(fields...)}
}

func (s *SimpleLogger) WithOptions(opts ...zap.Option) interfaces.Logger {
	return &SimpleLogger{logger: s.logger.WithOptions(opts...)}
}

func (s *SimpleLogger) Named(name string) interfaces.Logger {
	return &SimpleLogger{logger: s.logger.Named(name)}
}

func (s *SimpleLogger) Sync() error {
	return s.logger.Sync()
}

// 简单的配置服务
type ConfigService struct {
	logger interfaces.Logger
	config map[string]interface{}
}

func NewConfigService(logger interfaces.Logger) *ConfigService {
	return &ConfigService{
		logger: logger,
		config: make(map[string]interface{}),
	}
}

func (c *ConfigService) Set(key string, value interface{}) {
	c.logger.Info("设置配置", zap.String("key", key), zap.Any("value", value))
	c.config[key] = value
}

func (c *ConfigService) Get(key string) interface{} {
	c.logger.Debug("获取配置", zap.String("key", key))
	return c.config[key]
}

func (c *ConfigService) GetString(key string) string {
	if val, ok := c.config[key].(string); ok {
		return val
	}
	return ""
}

func (c *ConfigService) Dispose() error {
	c.logger.Info("释放配置服务资源")
	c.config = nil
	return nil
}

// 业务服务
type AppService struct {
	logger interfaces.Logger
	config *ConfigService
}

func NewAppService(logger interfaces.Logger, config *ConfigService) *AppService {
	return &AppService{
		logger: logger,
		config: config,
	}
}

func (a *AppService) DoWork() {
	a.logger.Info("开始执行业务逻辑")

	appName := a.config.GetString("app.name")
	if appName == "" {
		appName = "ClixGo"
		a.config.Set("app.name", appName)
	}

	a.logger.Info("执行应用程序", zap.String("name", appName))
	time.Sleep(100 * time.Millisecond)
	a.logger.Info("业务逻辑执行完成")
}

func (a *AppService) Dispose() error {
	a.logger.Info("释放应用服务资源")
	return nil
}

// 演示应用程序
type DemoApp struct {
	container *container.Container
}

func NewDemoApp() *DemoApp {
	return &DemoApp{
		container: container.NewContainer(),
	}
}

func (app *DemoApp) Setup() {
	// 注册日志服务 (单例)
	app.container.RegisterSingleton("logger", func(c *container.Container) (interface{}, error) {
		return NewSimpleLogger(), nil
	})

	// 注册配置服务 (单例，依赖日志服务)
	app.container.RegisterSingleton("config", func(c *container.Container) (interface{}, error) {
		logger, err := c.Get("logger")
		if err != nil {
			return nil, err
		}
		return NewConfigService(logger.(interfaces.Logger)), nil
	})

	// 注册应用服务 (瞬态，依赖日志和配置服务)
	app.container.RegisterTransient("app", func(c *container.Container) (interface{}, error) {
		logger, err := c.Get("logger")
		if err != nil {
			return nil, err
		}

		config, err := c.Get("config")
		if err != nil {
			return nil, err
		}

		return NewAppService(logger.(interfaces.Logger), config.(*ConfigService)), nil
	})
}

func (app *DemoApp) Run() {
	fmt.Println("🚀 ClixGo 依赖注入容器演示")
	fmt.Println("==========================================")

	// 获取服务
	logger, err := app.container.Get("logger")
	if err != nil {
		fmt.Printf("获取日志服务失败: %v\n", err)
		return
	}
	log := logger.(interfaces.Logger)

	log.Info("演示开始")

	// 验证容器
	if err := app.container.ValidateContainer(); err != nil {
		log.Error("容器验证失败", zap.Error(err))
		return
	}
	log.Info("容器验证成功")

	// 获取配置服务
	config, err := app.container.Get("config")
	if err != nil {
		log.Error("获取配置服务失败", zap.Error(err))
		return
	}
	configSvc := config.(*ConfigService)

	// 设置一些配置
	configSvc.Set("app.name", "ClixGo Demo")
	configSvc.Set("app.version", "1.0.0")
	configSvc.Set("debug", true)

	// 创建多个应用服务实例 (演示瞬态生命周期)
	for i := 0; i < 3; i++ {
		appSvc, err := app.container.Get("app")
		if err != nil {
			log.Error("获取应用服务失败", zap.Error(err))
			continue
		}

		service := appSvc.(*AppService)
		fmt.Printf("\n=== 运行应用服务实例 %d ===\n", i+1)
		service.DoWork()
	}

	log.Info("演示完成")
}

func (app *DemoApp) Shutdown() {
	fmt.Println("\n📋 正在关闭应用程序...")

	// 释放所有资源
	if err := app.container.Dispose(); err != nil {
		fmt.Printf("释放容器资源时出错: %v\n", err)
	}

	fmt.Println("✅ 应用程序已安全关闭")
}

func main() {
	app := NewDemoApp()

	// 设置依赖关系
	app.Setup()

	// 运行演示
	app.Run()

	// 关闭应用程序
	app.Shutdown()

	fmt.Println("\n📋 演示特性:")
	fmt.Println("- ✅ 依赖注入容器")
	fmt.Println("- ✅ 服务生命周期管理 (单例/瞬态)")
	fmt.Println("- ✅ 依赖关系解析")
	fmt.Println("- ✅ 容器验证")
	fmt.Println("- ✅ 资源自动释放")
	fmt.Println("- ✅ 类型安全的服务获取")
}
