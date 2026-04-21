package main

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/getlantern/systray"
	wailsruntime "github.com/wailsapp/wails/v2/pkg/runtime"
)

type SystrayManager struct {
	mu                sync.Mutex
	app               *DesktopApp
	mGoodProxies      *systray.MenuItem
	mMonitorStatus    *systray.MenuItem
	mThreeProxyStatus *systray.MenuItem
	mActiveParent     *systray.MenuItem
	iconData          []byte
}

func NewSystrayManager(app *DesktopApp) *SystrayManager {
	return &SystrayManager{
		app: app,
	}
}

func (s *SystrayManager) Start() {
	iconData, err := os.ReadFile("build/windows/icon.ico")
	if err != nil {
		iconData = nil
	}
	s.iconData = iconData

	go systray.Run(s.onReady, s.onExit)
}

func (s *SystrayManager) onReady() {
	if s.iconData != nil {
		systray.SetIcon(s.iconData)
	}
	systray.SetTitle("Proxy Checker")
	systray.SetTooltip("Proxy Checker - Управление прокси")

	s.mGoodProxies = systray.AddMenuItem("🔴 Рабочих прокси: 0", "Количество рабочих прокси")
	s.mGoodProxies.Disable()

	systray.AddSeparator()

	s.mMonitorStatus = systray.AddMenuItem("⚫ Монитор: выключен", "Статус монитора прокси")
	s.mMonitorStatus.Disable()

	s.mThreeProxyStatus = systray.AddMenuItem("⚫ 3proxy: выключен", "Статус 3proxy сервера")
	s.mThreeProxyStatus.Disable()

	systray.AddSeparator()

	s.mActiveParent = systray.AddMenuItem("⚫ Активный parent: нет", "Текущий активный прокси")
	s.mActiveParent.Disable()

	systray.AddSeparator()

	mShow := systray.AddMenuItem("Показать окно", "Открыть главное окно приложения")
	mQuit := systray.AddMenuItem("Выход", "Закрыть приложение")

	go func() {
		for {
			select {
			case <-mShow.ClickedCh:
				if s.app != nil && s.app.ctx != nil {
					s.showWindow()
				}
			case <-mQuit.ClickedCh:
				systray.Quit()
				if s.app != nil {
					s.app.Shutdown(s.app.ctx)
				}
				os.Exit(0)
			}
		}
	}()

	go s.updateLoop()
}

func (s *SystrayManager) onExit() {
}

func (s *SystrayManager) showWindow() {
	if s.app != nil && s.app.ctx != nil {
		wailsruntime.WindowShow(s.app.ctx)
		wailsruntime.WindowUnminimise(s.app.ctx)
	}
}

func (s *SystrayManager) updateLoop() {
	if s.app == nil {
		return
	}

	for {
		s.updateMenuItems()
		select {
		case <-s.app.ctx.Done():
			return
		default:
		}
		sleepDuration := 2000
		for i := 0; i < sleepDuration/100; i++ {
			select {
			case <-s.app.ctx.Done():
				return
			default:
			}
			sleepMillis(100)
		}
	}
}

func (s *SystrayManager) updateMenuItems() {
	if s.app == nil || s.app.state == nil {
		return
	}

	state := s.app.GetState()

	goodCount := 0
	for _, result := range state.Results {
		if result.OK {
			goodCount++
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mGoodProxies != nil {
		if goodCount > 0 {
			s.mGoodProxies.SetTitle(fmt.Sprintf("🟢 Рабочих прокси: %d", goodCount))
		} else {
			s.mGoodProxies.SetTitle(fmt.Sprintf("🔴 Рабочих прокси: %d", goodCount))
		}
	}

	if s.mMonitorStatus != nil {
		if state.MonitorRunning {
			s.mMonitorStatus.SetTitle("🟢 Монитор: ЗАПУЩЕН")
		} else {
			s.mMonitorStatus.SetTitle("⚫ Монитор: выключен")
		}
	}

	if s.mThreeProxyStatus != nil {
		if state.ThreeProxyRun {
			cfg := state.Config
			ip := cfg.ThreeProxy.InternalIP
			port := cfg.ThreeProxy.ProxyPort
			s.mThreeProxyStatus.SetTitle(fmt.Sprintf("🟢 3proxy: ЗАПУЩЕН (%s:%s)", ip, port))
		} else {
			s.mThreeProxyStatus.SetTitle("⚫ 3proxy: выключен")
		}
	}

	if s.mActiveParent != nil {
		if state.ActiveProxy != nil {
			parentIP := fmt.Sprintf("%s:%s", state.ActiveProxy.Host, state.ActiveProxy.Port)
			s.mActiveParent.SetTitle(fmt.Sprintf("🟢 Активный parent: %s", parentIP))
		} else {
			s.mActiveParent.SetTitle("⚫ Активный parent: нет")
		}
	}
}

func sleepMillis(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
