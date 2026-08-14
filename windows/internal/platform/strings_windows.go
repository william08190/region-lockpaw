//go:build windows

package platform

import "github.com/zzl/go-win32api/v2/win32"

type localizedText struct {
	lockRegion          string
	cancelSelection     string
	unlock              string
	unlockButton        string
	startAtLogin        string
	about               string
	exit                string
	ready               string
	readyDetail         string
	unlockTitle         string
	unlockPrompt        string
	unlockFailed        string
	displayChanged      string
	registrationFailed  string
	aboutBody           string
	visualPrivacyNotice string
	unexpectedError     string
	startAtLoginError   string
}

func loadLocalizedText(version string) localizedText {
	if win32.GetUserDefaultUILanguage()&0x3ff == 0x04 {
		return localizedText{
			lockRegion:          "锁定区域...\tCtrl+Alt+L",
			cancelSelection:     "取消选择\tEsc",
			unlock:              "解锁...\tCtrl+Alt+L",
			unlockButton:        "解锁",
			startAtLogin:        "开机启动",
			about:               "关于",
			exit:                "退出",
			ready:               "Region Lockpaw 已就绪",
			readyDetail:         "程序正在系统托盘中运行。",
			unlockTitle:         "解锁 Region Lockpaw",
			unlockPrompt:        "使用 Windows 安全登录解除区域遮罩？",
			unlockFailed:        "无法打开 Windows 安全登录。区域仍保持锁定。",
			displayChanged:      "所选区域当前不可用；为保持锁定，所有活动屏幕均已遮罩。",
			registrationFailed:  "无法注册全局快捷键 Ctrl+Alt+L。",
			aboutBody:           "Region Lockpaw " + version,
			visualPrivacyNotice: "这是视觉隐私工具，不是 Windows 安全边界。",
			unexpectedError:     "Region Lockpaw 遇到错误，无法启动。",
			startAtLoginError:   "无法更新开机启动设置。",
		}
	}

	return localizedText{
		lockRegion:          "Lock Region...\tCtrl+Alt+L",
		cancelSelection:     "Cancel selection\tEsc",
		unlock:              "Unlock...\tCtrl+Alt+L",
		unlockButton:        "Unlock",
		startAtLogin:        "Start at sign-in",
		about:               "About",
		exit:                "Exit",
		ready:               "Region Lockpaw is ready",
		readyDetail:         "The app is running in the notification area.",
		unlockTitle:         "Unlock Region Lockpaw",
		unlockPrompt:        "Use Windows secure sign-in to remove the region mask?",
		unlockFailed:        "Windows secure sign-in could not be opened. The region remains locked.",
		displayChanged:      "The selected region is unavailable; all active displays remain masked to preserve the lock.",
		registrationFailed:  "The Ctrl+Alt+L global shortcut could not be registered.",
		aboutBody:           "Region Lockpaw " + version,
		visualPrivacyNotice: "Visual privacy tool; not a Windows security boundary.",
		unexpectedError:     "Region Lockpaw encountered an error and could not start.",
		startAtLoginError:   "The start-at-sign-in setting could not be updated.",
	}
}
