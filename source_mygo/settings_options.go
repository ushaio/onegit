package main

import (
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/oneo-me/onegit/source_mygo/launchers"
	"github.com/oneo-me/onevra/source/components"
	"github.com/oneo-me/onevra/source/icons/lucide"
)

func (s *settings) theme() mygo.ThemeSource {
	if s.values.Theme == "" {
		return mygo.ThemeSystem
	}
	return s.values.Theme
}

func (s *settings) themeLabel(value mygo.ThemeSource) string {
	switch value {
	case mygo.ThemeLight:
		return s.text("浅色")
	case mygo.ThemeDark:
		return s.text("深色")
	default:
		return s.text("跟随系统")
	}
}

func (s *settings) setTheme(value mygo.ThemeSource) {
	next := s.values
	next.Theme = value
	if s.save(next) && s.window != nil {
		mygo.Theme.SetSource(value)
		s.window.Invalidate()
	}
}

func (s *settings) setLanguage(value string) {
	next := s.values
	next.Language = value
	if s.save(next) && s.window != nil {
		mygo.App.SetMenu(s.menu())
		s.window.Invalidate()
	}
}

func (s *settings) categories() []components.SettingsCategory {
	return []components.SettingsCategory{
		{ID: "general", Title: s.text("通用"), Section: s.text("应用"), Icon: lucide.Settings(), Groups: []components.SettingsGroup{
			{ID: "language", Items: []components.SettingsItem{{ID: "language", Title: s.text("语言"), Description: s.text("修改后立即生效，并自动保存。"), Build: func(c *ui.Context) {
				value := s.values.Language
				if settingsSelect(c, &value, s.text("语言偏好"), []string{"", "zh-Hans", "en"}, func(v string) string {
					switch v {
					case "zh-Hans":
						return "简体中文"
					case "en":
						return "English"
					default:
						return s.text("跟随系统")
					}
				}).Changed() {
					s.setLanguage(value)
					c.Invalidate()
				}
			}}}},
		}},
		{ID: "tools", Title: s.text("工具"), Section: s.text("应用"), Icon: lucide.Wrench(), Groups: []components.SettingsGroup{{ID: "launchers", Items: []components.SettingsItem{
			{ID: "terminal", Title: s.text("终端"), Keywords: []string{"terminal", "shell", "console", "命令行"}, Build: func(c *ui.Context) {
				s.launcherSetting(c, &s.values.Terminal, launchers.Terminal, s.text("终端"))
			}},
			{ID: "editor", Title: s.text("开发工具"), Keywords: []string{"editor", "IDE", "code", "编辑器", "代码"}, Build: func(c *ui.Context) {
				s.launcherSetting(c, &s.values.Editor, launchers.Editor, s.text("开发工具"))
			}},
		}}}},
		{ID: "appearance", Title: s.text("外观"), Section: s.text("应用"), Icon: lucide.Palette(), Groups: []components.SettingsGroup{{ID: "theme", Items: []components.SettingsItem{{
			ID: "theme", Title: s.text("主题"), Description: s.text("与侧边栏底部的主题选项同步。"), Build: func(c *ui.Context) {
				value := string(s.theme())
				if settingsSelect(c, &value, s.text("主题偏好"), []string{"system", "light", "dark"}, func(v string) string { return s.themeLabel(mygo.ThemeSource(v)) }).Changed() {
					s.setTheme(mygo.ThemeSource(value))
					c.Invalidate()
				}
			},
		}}}}},
	}
}

func settingsSelect(c *ui.Context, value *string, label string, choices []string, title func(string) string) ui.Element {
	sel := components.Select(c, value)
	sel.Trigger.Label(label).Width(c.Theme().Space(32)).Shrink(0).Children(func() { components.Text(c, title(*value)).SingleLine() })
	sel.Popup(func(panel ui.Element) {
		for _, choice := range choices {
			sel.Item(choice).Label(title(choice)).Children(func() { components.Text(c, title(choice)) })
		}
	})
	return sel.Trigger
}
