/*
 * @Author: Lzww0608
 * @Date: 2025-6-26 19:58:59
 * @LastEditors: Lzww0608
 * @LastEditTime: 2025-6-26 19:58:59
 * @Description: 主题配置集成 - 扩展主题系统，支持动态加载和切换
 */

package ui

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Lzww0608/ClixGo/pkg/logger"
	"github.com/gdamore/tcell/v2"
	"go.uber.org/zap"
)

// ===== 扩展主题结构 =====

// EnhancedTheme 增强版主题结构
type EnhancedTheme struct {
	// 基础主题信息
	Name        string    `json:"name" yaml:"name"`
	Description string    `json:"description" yaml:"description"`
	Author      string    `json:"author" yaml:"author"`
	Version     string    `json:"version" yaml:"version"`
	CreatedAt   time.Time `json:"created_at" yaml:"created_at"`

	// 核心颜色配置
	Colors ThemeColors `json:"colors" yaml:"colors"`

	// 样式配置
	Styles ThemeStyles `json:"styles" yaml:"styles"`

	// 组件特定配置
	Components ThemeComponents `json:"components" yaml:"components"`

	// 动画和效果
	Effects ThemeEffects `json:"effects" yaml:"effects"`

	// 快捷键绑定
	KeyBindings map[string]string `json:"key_bindings" yaml:"key_bindings"`
}

// ThemeColors 主题颜色配置
type ThemeColors struct {
	// 基础颜色
	Primary   tcell.Color `json:"primary" yaml:"primary"`
	Secondary tcell.Color `json:"secondary" yaml:"secondary"`
	Success   tcell.Color `json:"success" yaml:"success"`
	Warning   tcell.Color `json:"warning" yaml:"warning"`
	Error     tcell.Color `json:"error" yaml:"error"`
	Info      tcell.Color `json:"info" yaml:"info"`

	// 背景和前景
	Background tcell.Color `json:"background" yaml:"background"`
	Foreground tcell.Color `json:"foreground" yaml:"foreground"`
	Muted      tcell.Color `json:"muted" yaml:"muted"`
	Accent     tcell.Color `json:"accent" yaml:"accent"`

	// 边框和分隔符
	Border       tcell.Color `json:"border" yaml:"border"`
	ActiveBorder tcell.Color `json:"active_border" yaml:"active_border"`
	Separator    tcell.Color `json:"separator" yaml:"separator"`

	// 状态栏
	StatusBar  tcell.Color `json:"status_bar" yaml:"status_bar"`
	StatusText tcell.Color `json:"status_text" yaml:"status_text"`

	// 侧边栏
	SidebarBg     tcell.Color `json:"sidebar_bg" yaml:"sidebar_bg"`
	SidebarText   tcell.Color `json:"sidebar_text" yaml:"sidebar_text"`
	SidebarActive tcell.Color `json:"sidebar_active" yaml:"sidebar_active"`

	// 面板
	PanelBg    tcell.Color `json:"panel_bg" yaml:"panel_bg"`
	PanelText  tcell.Color `json:"panel_text" yaml:"panel_text"`
	PanelTitle tcell.Color `json:"panel_title" yaml:"panel_title"`

	// 高亮和选择
	Highlight tcell.Color `json:"highlight" yaml:"highlight"`
	Selection tcell.Color `json:"selection" yaml:"selection"`
	Focus     tcell.Color `json:"focus" yaml:"focus"`
}

// ThemeStyles 样式配置
type ThemeStyles struct {
	// 文本样式
	Bold          bool `json:"bold" yaml:"bold"`
	Italic        bool `json:"italic" yaml:"italic"`
	Underline     bool `json:"underline" yaml:"underline"`
	Strikethrough bool `json:"strikethrough" yaml:"strikethrough"`

	// 边框样式
	BorderStyle BorderStyle `json:"border_style" yaml:"border_style"`

	// 透明度
	Transparency float32 `json:"transparency" yaml:"transparency"`
}

// BorderStyle 边框样式
type BorderStyle string

const (
	BorderStyleNone    BorderStyle = "none"
	BorderStyleSolid   BorderStyle = "solid"
	BorderStyleDouble  BorderStyle = "double"
	BorderStyleRounded BorderStyle = "rounded"
	BorderStyleDashed  BorderStyle = "dashed"
)

// ThemeComponents 组件特定配置
type ThemeComponents struct {
	StatusBar StatusBarTheme `json:"status_bar" yaml:"status_bar"`
	Sidebar   SidebarTheme   `json:"sidebar" yaml:"sidebar"`
	Panel     PanelTheme     `json:"panel" yaml:"panel"`
	Dialog    DialogTheme    `json:"dialog" yaml:"dialog"`
}

// StatusBarTheme 状态栏主题
type StatusBarTheme struct {
	Height     int         `json:"height" yaml:"height"`
	ShowTime   bool        `json:"show_time" yaml:"show_time"`
	ShowStats  bool        `json:"show_stats" yaml:"show_stats"`
	Format     string      `json:"format" yaml:"format"`
	Background tcell.Color `json:"background" yaml:"background"`
	Foreground tcell.Color `json:"foreground" yaml:"foreground"`
	Separator  string      `json:"separator" yaml:"separator"`
}

// SidebarTheme 侧边栏主题
type SidebarTheme struct {
	Width      int         `json:"width" yaml:"width"`
	ShowIcons  bool        `json:"show_icons" yaml:"show_icons"`
	AutoHide   bool        `json:"auto_hide" yaml:"auto_hide"`
	Background tcell.Color `json:"background" yaml:"background"`
	Foreground tcell.Color `json:"foreground" yaml:"foreground"`
	ActiveBg   tcell.Color `json:"active_bg" yaml:"active_bg"`
	ActiveFg   tcell.Color `json:"active_fg" yaml:"active_fg"`
}

// PanelTheme 面板主题
type PanelTheme struct {
	ShowBorder  bool        `json:"show_border" yaml:"show_border"`
	ShowTitle   bool        `json:"show_title" yaml:"show_title"`
	Padding     int         `json:"padding" yaml:"padding"`
	Background  tcell.Color `json:"background" yaml:"background"`
	Foreground  tcell.Color `json:"foreground" yaml:"foreground"`
	BorderColor tcell.Color `json:"border_color" yaml:"border_color"`
	TitleColor  tcell.Color `json:"title_color" yaml:"title_color"`
}

// DialogTheme 对话框主题
type DialogTheme struct {
	Background  tcell.Color `json:"background" yaml:"background"`
	Foreground  tcell.Color `json:"foreground" yaml:"foreground"`
	BorderColor tcell.Color `json:"border_color" yaml:"border_color"`
	ButtonBg    tcell.Color `json:"button_bg" yaml:"button_bg"`
	ButtonFg    tcell.Color `json:"button_fg" yaml:"button_fg"`
	Shadow      bool        `json:"shadow" yaml:"shadow"`
}

// ThemeEffects 主题效果配置
type ThemeEffects struct {
	FadeIn      bool          `json:"fade_in" yaml:"fade_in"`
	FadeOut     bool          `json:"fade_out" yaml:"fade_out"`
	Transitions bool          `json:"transitions" yaml:"transitions"`
	Duration    time.Duration `json:"duration" yaml:"duration"`
	Animations  bool          `json:"animations" yaml:"animations"`
}

// ===== 主题管理器 =====

// ThemeManager 主题管理器
type ThemeManager struct {
	// 主题存储
	themes      map[string]*EnhancedTheme // 主题名称 -> 主题对象
	activeTheme string                    // 当前活动主题名称

	// 配置路径
	themesDir  string // 主题目录路径
	configFile string // 配置文件路径

	// 控制
	mutex    sync.RWMutex
	watchers []ThemeWatcher // 主题变更监听器

	// 缓存
	cache       map[string]*EnhancedTheme
	cacheExpiry time.Time
}

// ThemeWatcher 主题变更监听器
type ThemeWatcher interface {
	OnThemeChanged(oldTheme, newTheme *EnhancedTheme)
}

// NewThemeManager 创建主题管理器
func NewThemeManager(themesDir, configFile string) (*ThemeManager, error) {
	tm := &ThemeManager{
		themes:     make(map[string]*EnhancedTheme),
		themesDir:  themesDir,
		configFile: configFile,
		cache:      make(map[string]*EnhancedTheme),
		watchers:   make([]ThemeWatcher, 0),
	}

	// 确保主题目录存在
	if err := os.MkdirAll(themesDir, 0755); err != nil {
		return nil, fmt.Errorf("创建主题目录失败: %w", err)
	}

	// 加载内置主题
	if err := tm.loadBuiltinThemes(); err != nil {
		return nil, fmt.Errorf("加载内置主题失败: %w", err)
	}

	// 加载用户主题
	if err := tm.loadUserThemes(); err != nil {
		logger.Warn("加载用户主题失败", zap.Error(err))
	}

	// 加载配置
	if err := tm.loadConfig(); err != nil {
		logger.Warn("加载主题配置失败", zap.Error(err))
		// 使用默认主题
		tm.activeTheme = "default"
	}

	logger.Info("主题管理器初始化完成",
		zap.String("themes_dir", themesDir),
		zap.String("config_file", configFile),
		zap.Int("theme_count", len(tm.themes)),
		zap.String("active_theme", tm.activeTheme))

	return tm, nil
}

// ===== 内置主题定义 =====

// loadBuiltinThemes 加载内置主题
func (tm *ThemeManager) loadBuiltinThemes() error {
	// 默认主题
	defaultTheme := &EnhancedTheme{
		Name:        "default",
		Description: "默认主题 - 经典黑白配色",
		Author:      "ClixGo Team",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		Colors: ThemeColors{
			Primary:       tcell.ColorBlue,
			Secondary:     tcell.ColorGray,
			Success:       tcell.ColorGreen,
			Warning:       tcell.ColorYellow,
			Error:         tcell.ColorRed,
			Info:          tcell.ColorTeal,
			Background:    tcell.ColorBlack,
			Foreground:    tcell.ColorWhite,
			Muted:         tcell.ColorGray,
			Accent:        tcell.ColorBlue,
			Border:        tcell.ColorGray,
			ActiveBorder:  tcell.ColorBlue,
			Separator:     tcell.ColorGray,
			StatusBar:     tcell.ColorDarkBlue,
			StatusText:    tcell.ColorWhite,
			SidebarBg:     tcell.ColorDarkGray,
			SidebarText:   tcell.ColorWhite,
			SidebarActive: tcell.ColorBlue,
			PanelBg:       tcell.ColorBlack,
			PanelText:     tcell.ColorWhite,
			PanelTitle:    tcell.ColorTeal,
			Highlight:     tcell.ColorYellow,
			Selection:     tcell.ColorBlue,
			Focus:         tcell.ColorGreen,
		},
		Styles: ThemeStyles{
			BorderStyle:  BorderStyleSolid,
			Transparency: 0.0,
		},
		Components: ThemeComponents{
			StatusBar: StatusBarTheme{
				Height:     1,
				ShowTime:   true,
				ShowStats:  true,
				Format:     "[%s] %s | %s",
				Background: tcell.ColorDarkBlue,
				Foreground: tcell.ColorWhite,
				Separator:  " | ",
			},
			Sidebar: SidebarTheme{
				Width:      25,
				ShowIcons:  true,
				AutoHide:   false,
				Background: tcell.ColorDarkGray,
				Foreground: tcell.ColorWhite,
				ActiveBg:   tcell.ColorBlue,
				ActiveFg:   tcell.ColorWhite,
			},
			Panel: PanelTheme{
				ShowBorder:  true,
				ShowTitle:   true,
				Padding:     1,
				Background:  tcell.ColorBlack,
				Foreground:  tcell.ColorWhite,
				BorderColor: tcell.ColorGray,
				TitleColor:  tcell.ColorTeal,
			},
			Dialog: DialogTheme{
				Background:  tcell.ColorDarkGray,
				Foreground:  tcell.ColorWhite,
				BorderColor: tcell.ColorBlue,
				ButtonBg:    tcell.ColorBlue,
				ButtonFg:    tcell.ColorWhite,
				Shadow:      true,
			},
		},
		Effects: ThemeEffects{
			FadeIn:      false,
			FadeOut:     false,
			Transitions: false,
			Duration:    time.Millisecond * 200,
			Animations:  false,
		},
		KeyBindings: map[string]string{
			"F9":  "toggle_theme",
			"F10": "next_theme",
			"F11": "prev_theme",
		},
	}

	// 暗色主题
	darkTheme := &EnhancedTheme{
		Name:        "dark",
		Description: "暗色主题 - 护眼深色配色",
		Author:      "ClixGo Team",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		Colors: ThemeColors{
			Primary:       tcell.ColorDodgerBlue,
			Secondary:     tcell.ColorSlateGray,
			Success:       tcell.ColorLimeGreen,
			Warning:       tcell.ColorOrange,
			Error:         tcell.ColorCrimson,
			Info:          tcell.ColorDeepSkyBlue,
			Background:    tcell.ColorBlack,
			Foreground:    tcell.ColorLightGray,
			Muted:         tcell.ColorDimGray,
			Accent:        tcell.ColorDodgerBlue,
			Border:        tcell.ColorDimGray,
			ActiveBorder:  tcell.ColorDodgerBlue,
			Separator:     tcell.ColorDimGray,
			StatusBar:     tcell.ColorMidnightBlue,
			StatusText:    tcell.ColorLightGray,
			SidebarBg:     tcell.ColorDarkSlateGray,
			SidebarText:   tcell.ColorLightGray,
			SidebarActive: tcell.ColorDodgerBlue,
			PanelBg:       tcell.ColorBlack,
			PanelText:     tcell.ColorLightGray,
			PanelTitle:    tcell.ColorDeepSkyBlue,
			Highlight:     tcell.ColorGold,
			Selection:     tcell.ColorDodgerBlue,
			Focus:         tcell.ColorLimeGreen,
		},
		Styles: ThemeStyles{
			BorderStyle:  BorderStyleRounded,
			Transparency: 0.1,
		},
		Components: ThemeComponents{
			StatusBar: StatusBarTheme{
				Height:     1,
				ShowTime:   true,
				ShowStats:  true,
				Format:     "🌙 [%s] %s | %s",
				Background: tcell.ColorMidnightBlue,
				Foreground: tcell.ColorLightGray,
				Separator:  " • ",
			},
			Sidebar: SidebarTheme{
				Width:      28,
				ShowIcons:  true,
				AutoHide:   false,
				Background: tcell.ColorDarkSlateGray,
				Foreground: tcell.ColorLightGray,
				ActiveBg:   tcell.ColorDodgerBlue,
				ActiveFg:   tcell.ColorWhite,
			},
			Panel: PanelTheme{
				ShowBorder:  true,
				ShowTitle:   true,
				Padding:     1,
				Background:  tcell.ColorBlack,
				Foreground:  tcell.ColorLightGray,
				BorderColor: tcell.ColorDimGray,
				TitleColor:  tcell.ColorDeepSkyBlue,
			},
			Dialog: DialogTheme{
				Background:  tcell.ColorDarkSlateGray,
				Foreground:  tcell.ColorLightGray,
				BorderColor: tcell.ColorDodgerBlue,
				ButtonBg:    tcell.ColorDodgerBlue,
				ButtonFg:    tcell.ColorWhite,
				Shadow:      true,
			},
		},
		Effects: ThemeEffects{
			FadeIn:      true,
			FadeOut:     true,
			Transitions: true,
			Duration:    time.Millisecond * 300,
			Animations:  true,
		},
		KeyBindings: map[string]string{
			"F9":  "toggle_theme",
			"F10": "next_theme",
			"F11": "prev_theme",
		},
	}

	// 亮色主题
	lightTheme := &EnhancedTheme{
		Name:        "light",
		Description: "亮色主题 - 清新明亮配色",
		Author:      "ClixGo Team",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		Colors: ThemeColors{
			Primary:       tcell.ColorRoyalBlue,
			Secondary:     tcell.ColorSlateGray,
			Success:       tcell.ColorForestGreen,
			Warning:       tcell.ColorDarkOrange,
			Error:         tcell.ColorFireBrick,
			Info:          tcell.ColorSteelBlue,
			Background:    tcell.ColorWhite,
			Foreground:    tcell.ColorBlack,
			Muted:         tcell.ColorGray,
			Accent:        tcell.ColorRoyalBlue,
			Border:        tcell.ColorLightGray,
			ActiveBorder:  tcell.ColorRoyalBlue,
			Separator:     tcell.ColorLightGray,
			StatusBar:     tcell.ColorLightSteelBlue,
			StatusText:    tcell.ColorBlack,
			SidebarBg:     tcell.ColorWhiteSmoke,
			SidebarText:   tcell.ColorBlack,
			SidebarActive: tcell.ColorRoyalBlue,
			PanelBg:       tcell.ColorWhite,
			PanelText:     tcell.ColorBlack,
			PanelTitle:    tcell.ColorSteelBlue,
			Highlight:     tcell.ColorGold,
			Selection:     tcell.ColorLightSkyBlue,
			Focus:         tcell.ColorForestGreen,
		},
		Styles: ThemeStyles{
			BorderStyle:  BorderStyleSolid,
			Transparency: 0.0,
		},
		Components: ThemeComponents{
			StatusBar: StatusBarTheme{
				Height:     1,
				ShowTime:   true,
				ShowStats:  true,
				Format:     "☀️ [%s] %s | %s",
				Background: tcell.ColorLightSteelBlue,
				Foreground: tcell.ColorBlack,
				Separator:  " | ",
			},
			Sidebar: SidebarTheme{
				Width:      25,
				ShowIcons:  true,
				AutoHide:   false,
				Background: tcell.ColorWhiteSmoke,
				Foreground: tcell.ColorBlack,
				ActiveBg:   tcell.ColorRoyalBlue,
				ActiveFg:   tcell.ColorWhite,
			},
			Panel: PanelTheme{
				ShowBorder:  true,
				ShowTitle:   true,
				Padding:     1,
				Background:  tcell.ColorWhite,
				Foreground:  tcell.ColorBlack,
				BorderColor: tcell.ColorLightGray,
				TitleColor:  tcell.ColorSteelBlue,
			},
			Dialog: DialogTheme{
				Background:  tcell.ColorWhiteSmoke,
				Foreground:  tcell.ColorBlack,
				BorderColor: tcell.ColorRoyalBlue,
				ButtonBg:    tcell.ColorRoyalBlue,
				ButtonFg:    tcell.ColorWhite,
				Shadow:      true,
			},
		},
		Effects: ThemeEffects{
			FadeIn:      true,
			FadeOut:     true,
			Transitions: true,
			Duration:    time.Millisecond * 250,
			Animations:  true,
		},
		KeyBindings: map[string]string{
			"F9":  "toggle_theme",
			"F10": "next_theme",
			"F11": "prev_theme",
		},
	}

	// 终端主题
	terminalTheme := &EnhancedTheme{
		Name:        "terminal",
		Description: "终端主题 - 经典绿色终端风格",
		Author:      "ClixGo Team",
		Version:     "1.0.0",
		CreatedAt:   time.Now(),
		Colors: ThemeColors{
			Primary:       tcell.ColorLimeGreen,
			Secondary:     tcell.ColorDarkGreen,
			Success:       tcell.ColorGreen,
			Warning:       tcell.ColorYellow,
			Error:         tcell.ColorRed,
			Info:          tcell.ColorTeal,
			Background:    tcell.ColorBlack,
			Foreground:    tcell.ColorLimeGreen,
			Muted:         tcell.ColorDarkGreen,
			Accent:        tcell.ColorLimeGreen,
			Border:        tcell.ColorDarkGreen,
			ActiveBorder:  tcell.ColorLimeGreen,
			Separator:     tcell.ColorDarkGreen,
			StatusBar:     tcell.ColorDarkGreen,
			StatusText:    tcell.ColorLimeGreen,
			SidebarBg:     tcell.ColorBlack,
			SidebarText:   tcell.ColorLimeGreen,
			SidebarActive: tcell.ColorGreen,
			PanelBg:       tcell.ColorBlack,
			PanelText:     tcell.ColorLimeGreen,
			PanelTitle:    tcell.ColorGreen,
			Highlight:     tcell.ColorYellow,
			Selection:     tcell.ColorGreen,
			Focus:         tcell.ColorLimeGreen,
		},
		Styles: ThemeStyles{
			BorderStyle:  BorderStyleSolid,
			Transparency: 0.0,
		},
		Components: ThemeComponents{
			StatusBar: StatusBarTheme{
				Height:     1,
				ShowTime:   true,
				ShowStats:  true,
				Format:     "$ [%s] %s | %s",
				Background: tcell.ColorDarkGreen,
				Foreground: tcell.ColorLimeGreen,
				Separator:  " | ",
			},
			Sidebar: SidebarTheme{
				Width:      25,
				ShowIcons:  false,
				AutoHide:   false,
				Background: tcell.ColorBlack,
				Foreground: tcell.ColorLimeGreen,
				ActiveBg:   tcell.ColorDarkGreen,
				ActiveFg:   tcell.ColorLimeGreen,
			},
			Panel: PanelTheme{
				ShowBorder:  true,
				ShowTitle:   true,
				Padding:     1,
				Background:  tcell.ColorBlack,
				Foreground:  tcell.ColorLimeGreen,
				BorderColor: tcell.ColorDarkGreen,
				TitleColor:  tcell.ColorGreen,
			},
			Dialog: DialogTheme{
				Background:  tcell.ColorBlack,
				Foreground:  tcell.ColorLimeGreen,
				BorderColor: tcell.ColorGreen,
				ButtonBg:    tcell.ColorDarkGreen,
				ButtonFg:    tcell.ColorLimeGreen,
				Shadow:      false,
			},
		},
		Effects: ThemeEffects{
			FadeIn:      false,
			FadeOut:     false,
			Transitions: false,
			Duration:    time.Millisecond * 100,
			Animations:  false,
		},
		KeyBindings: map[string]string{
			"F9":  "toggle_theme",
			"F10": "next_theme",
			"F11": "prev_theme",
		},
	}

	// 注册内置主题
	tm.themes["default"] = defaultTheme
	tm.themes["dark"] = darkTheme
	tm.themes["light"] = lightTheme
	tm.themes["terminal"] = terminalTheme

	logger.Info("内置主题加载完成", zap.Int("count", 4))
	return nil
}

// ===== 主题管理功能 =====

// GetTheme 获取指定主题
func (tm *ThemeManager) GetTheme(name string) (*EnhancedTheme, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	theme, exists := tm.themes[name]
	if !exists {
		return nil, fmt.Errorf("主题 '%s' 不存在", name)
	}

	return theme, nil
}

// GetActiveTheme 获取当前活动主题
func (tm *ThemeManager) GetActiveTheme() (*EnhancedTheme, error) {
	tm.mutex.RLock()
	activeThemeName := tm.activeTheme
	tm.mutex.RUnlock()

	return tm.GetTheme(activeThemeName)
}

// SetActiveTheme 设置活动主题
func (tm *ThemeManager) SetActiveTheme(name string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 检查主题是否存在
	newTheme, exists := tm.themes[name]
	if !exists {
		return fmt.Errorf("主题 '%s' 不存在", name)
	}

	// 获取旧主题
	var oldTheme *EnhancedTheme
	if tm.activeTheme != "" {
		oldTheme = tm.themes[tm.activeTheme]
	}

	// 设置新主题
	tm.activeTheme = name

	// 保存配置
	if err := tm.saveConfig(); err != nil {
		logger.Warn("保存主题配置失败", zap.Error(err))
	}

	// 通知监听器
	tm.notifyWatchers(oldTheme, newTheme)

	logger.Info("主题切换成功",
		zap.String("old_theme", func() string {
			if oldTheme != nil {
				return oldTheme.Name
			}
			return "none"
		}()),
		zap.String("new_theme", newTheme.Name))

	return nil
}

// ListThemes 列出所有可用主题
func (tm *ThemeManager) ListThemes() []string {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	themes := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		themes = append(themes, name)
	}

	return themes
}

// GetThemeInfo 获取主题详细信息
func (tm *ThemeManager) GetThemeInfo(name string) (*EnhancedTheme, error) {
	return tm.GetTheme(name)
}

// ===== 动态加载功能 =====

// loadUserThemes 加载用户自定义主题
func (tm *ThemeManager) loadUserThemes() error {
	files, err := ioutil.ReadDir(tm.themesDir)
	if err != nil {
		return fmt.Errorf("读取主题目录失败: %w", err)
	}

	loadedCount := 0
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		// 只处理 .json 文件
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		themePath := filepath.Join(tm.themesDir, file.Name())
		if err := tm.loadThemeFromFile(themePath); err != nil {
			logger.Warn("加载主题文件失败",
				zap.String("file", themePath),
				zap.Error(err))
			continue
		}

		loadedCount++
	}

	logger.Info("用户主题加载完成",
		zap.Int("loaded", loadedCount))

	return nil
}

// loadThemeFromFile 从文件加载主题
func (tm *ThemeManager) loadThemeFromFile(filePath string) error {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取主题文件失败: %w", err)
	}

	var theme EnhancedTheme
	if err := json.Unmarshal(data, &theme); err != nil {
		return fmt.Errorf("解析主题文件失败: %w", err)
	}

	// 验证主题
	if err := tm.validateTheme(&theme); err != nil {
		return fmt.Errorf("主题验证失败: %w", err)
	}

	tm.mutex.Lock()
	tm.themes[theme.Name] = &theme
	tm.mutex.Unlock()

	logger.Debug("主题加载成功",
		zap.String("name", theme.Name),
		zap.String("file", filePath))

	return nil
}

// validateTheme 验证主题配置
func (tm *ThemeManager) validateTheme(theme *EnhancedTheme) error {
	if theme.Name == "" {
		return fmt.Errorf("主题名称不能为空")
	}

	if theme.Version == "" {
		theme.Version = "1.0.0"
	}

	if theme.CreatedAt.IsZero() {
		theme.CreatedAt = time.Now()
	}

	return nil
}

// SaveTheme 保存主题到文件
func (tm *ThemeManager) SaveTheme(theme *EnhancedTheme) error {
	if err := tm.validateTheme(theme); err != nil {
		return fmt.Errorf("主题验证失败: %w", err)
	}

	// 序列化主题
	data, err := json.MarshalIndent(theme, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化主题失败: %w", err)
	}

	// 写入文件
	filePath := filepath.Join(tm.themesDir, theme.Name+".json")
	if err := ioutil.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("写入主题文件失败: %w", err)
	}

	// 更新内存中的主题
	tm.mutex.Lock()
	tm.themes[theme.Name] = theme
	tm.mutex.Unlock()

	logger.Info("主题保存成功",
		zap.String("name", theme.Name),
		zap.String("file", filePath))

	return nil
}

// DeleteTheme 删除主题
func (tm *ThemeManager) DeleteTheme(name string) error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 检查是否为内置主题
	builtinThemes := []string{"default", "dark", "light", "terminal"}
	for _, builtin := range builtinThemes {
		if name == builtin {
			return fmt.Errorf("不能删除内置主题: %s", name)
		}
	}

	// 检查主题是否存在
	if _, exists := tm.themes[name]; !exists {
		return fmt.Errorf("主题 '%s' 不存在", name)
	}

	// 删除文件
	filePath := filepath.Join(tm.themesDir, name+".json")
	if err := os.Remove(filePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除主题文件失败: %w", err)
	}

	// 从内存中删除
	delete(tm.themes, name)

	// 如果删除的是当前活动主题，切换到默认主题
	if tm.activeTheme == name {
		tm.activeTheme = "default"
		if err := tm.saveConfig(); err != nil {
			logger.Warn("保存配置失败", zap.Error(err))
		}
	}

	logger.Info("主题删除成功", zap.String("name", name))
	return nil
}

// ===== 配置管理 =====

// ThemeConfig 主题配置
type ThemeConfig struct {
	ActiveTheme string    `json:"active_theme"`
	LastUpdate  time.Time `json:"last_update"`
}

// loadConfig 加载配置
func (tm *ThemeManager) loadConfig() error {
	if _, err := os.Stat(tm.configFile); os.IsNotExist(err) {
		// 配置文件不存在，使用默认配置
		tm.activeTheme = "default"
		return tm.saveConfig()
	}

	data, err := ioutil.ReadFile(tm.configFile)
	if err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config ThemeConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证主题是否存在
	if _, exists := tm.themes[config.ActiveTheme]; !exists {
		logger.Warn("配置中的主题不存在，使用默认主题",
			zap.String("theme", config.ActiveTheme))
		tm.activeTheme = "default"
		return tm.saveConfig()
	}

	tm.activeTheme = config.ActiveTheme
	return nil
}

// saveConfig 保存配置
func (tm *ThemeManager) saveConfig() error {
	config := ThemeConfig{
		ActiveTheme: tm.activeTheme,
		LastUpdate:  time.Now(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	// 确保配置目录存在
	configDir := filepath.Dir(tm.configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("创建配置目录失败: %w", err)
	}

	if err := ioutil.WriteFile(tm.configFile, data, 0644); err != nil {
		return fmt.Errorf("写入配置文件失败: %w", err)
	}

	return nil
}

// ===== 监听器管理 =====

// AddWatcher 添加主题变更监听器
func (tm *ThemeManager) AddWatcher(watcher ThemeWatcher) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	tm.watchers = append(tm.watchers, watcher)
}

// RemoveWatcher 移除主题变更监听器
func (tm *ThemeManager) RemoveWatcher(watcher ThemeWatcher) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	for i, w := range tm.watchers {
		if w == watcher {
			tm.watchers = append(tm.watchers[:i], tm.watchers[i+1:]...)
			break
		}
	}
}

// notifyWatchers 通知所有监听器
func (tm *ThemeManager) notifyWatchers(oldTheme, newTheme *EnhancedTheme) {
	for _, watcher := range tm.watchers {
		go func(w ThemeWatcher) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("主题监听器异常", zap.Any("error", r))
				}
			}()
			w.OnThemeChanged(oldTheme, newTheme)
		}(watcher)
	}
}

// ===== 快捷功能 =====

// NextTheme 切换到下一个主题
func (tm *ThemeManager) NextTheme() error {
	themes := tm.ListThemes()
	if len(themes) <= 1 {
		return fmt.Errorf("没有足够的主题可切换")
	}

	currentIndex := -1
	for i, name := range themes {
		if name == tm.activeTheme {
			currentIndex = i
			break
		}
	}

	nextIndex := (currentIndex + 1) % len(themes)
	return tm.SetActiveTheme(themes[nextIndex])
}

// PrevTheme 切换到上一个主题
func (tm *ThemeManager) PrevTheme() error {
	themes := tm.ListThemes()
	if len(themes) <= 1 {
		return fmt.Errorf("没有足够的主题可切换")
	}

	currentIndex := -1
	for i, name := range themes {
		if name == tm.activeTheme {
			currentIndex = i
			break
		}
	}

	prevIndex := (currentIndex - 1 + len(themes)) % len(themes)
	return tm.SetActiveTheme(themes[prevIndex])
}

// ToggleTheme 在两个主题间切换（默认和暗色）
func (tm *ThemeManager) ToggleTheme() error {
	if tm.activeTheme == "default" {
		return tm.SetActiveTheme("dark")
	}
	return tm.SetActiveTheme("default")
}

// ===== 实用工具 =====

// ConvertToLegacyTheme 转换为旧版主题格式（兼容性）
func (tm *ThemeManager) ConvertToLegacyTheme(name string) (*Theme, error) {
	enhancedTheme, err := tm.GetTheme(name)
	if err != nil {
		return nil, err
	}

	return &Theme{
		Background:   enhancedTheme.Colors.Background,
		Foreground:   enhancedTheme.Colors.Foreground,
		Border:       enhancedTheme.Colors.Border,
		ActiveBorder: enhancedTheme.Colors.ActiveBorder,
		StatusBar:    enhancedTheme.Colors.StatusBar,
		StatusText:   enhancedTheme.Colors.StatusText,
	}, nil
}

// GetThemePreview 获取主题预览信息
func (tm *ThemeManager) GetThemePreview(name string) (string, error) {
	theme, err := tm.GetTheme(name)
	if err != nil {
		return "", err
	}

	preview := fmt.Sprintf(`
主题名称: %s
描述: %s
作者: %s
版本: %s
创建时间: %s

颜色配置:
  主色调: %v
  背景色: %v
  前景色: %v
  边框色: %v
  活动边框: %v

组件配置:
  状态栏: %s
  侧边栏宽度: %d
  面板边框: %v
  
效果配置:
  渐变效果: %v
  过渡动画: %v
  动画持续时间: %v
`,
		theme.Name,
		theme.Description,
		theme.Author,
		theme.Version,
		theme.CreatedAt.Format("2006-01-02 15:04:05"),
		theme.Colors.Primary,
		theme.Colors.Background,
		theme.Colors.Foreground,
		theme.Colors.Border,
		theme.Colors.ActiveBorder,
		theme.Components.StatusBar.Format,
		theme.Components.Sidebar.Width,
		theme.Components.Panel.ShowBorder,
		theme.Effects.FadeIn,
		theme.Effects.Transitions,
		theme.Effects.Duration,
	)

	return preview, nil
}

// Close 关闭主题管理器
func (tm *ThemeManager) Close() error {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// 保存当前配置
	if err := tm.saveConfig(); err != nil {
		logger.Warn("关闭时保存配置失败", zap.Error(err))
	}

	// 清理资源
	tm.themes = nil
	tm.watchers = nil
	tm.cache = nil

	logger.Info("主题管理器已关闭")
	return nil
}
