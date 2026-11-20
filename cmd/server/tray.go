//go:build windows || linux || (darwin && cgo)

package main

import (
	"fmt"
	"runtime"
	"time"

	"fyne.io/systray"

	"pc_monitor/pkg/buildinfo"
	"pc_monitor/pkg/icon"
	"pc_monitor/pkg/service"
)

const trayAvailable = true

// trayCommand shows a tray/menu-bar icon that controls the background
// server through the local settings API (starting it if needed).
func trayCommand() error {
	cfg := loadConfigOrDefaults()
	if ok, _ := pingSettings(cfg.SettingsPort); !ok {
		if _, err := service.StartDetached(); err != nil {
			return err
		}
		waitFor(cfg.SettingsPort, true, 20*time.Second)
	}
	systray.Run(func() { trayReady(cfg.SettingsPort) }, func() {})
	return nil
}

func trayReady(port int) {
	if runtime.GOOS == "windows" {
		systray.SetIcon(icon.ICO(32))
	} else {
		systray.SetTemplateIcon(icon.PNG(44), icon.PNG(44))
	}
	systray.SetTitle("")
	systray.SetTooltip("PC Monitor")

	title := systray.AddMenuItem("PC Monitor "+buildinfo.Version, "")
	title.Disable()
	status := systray.AddMenuItem("Checking…", "")
	status.Disable()
	systray.AddSeparator()
	openSettings := systray.AddMenuItem("Open settings…", "Pairing QR, PIN and options")
	openDash := systray.AddMenuItem("Open dashboard", "Open the pairing link in your browser")
	autostart := systray.AddMenuItemCheckbox("Start at login", "", service.AutostartEnabled())
	updateItem := systray.AddMenuItem("Check for updates", "")
	systray.AddSeparator()
	startStop := systray.AddMenuItem("Stop server", "")
	quit := systray.AddMenuItem("Quit tray (server keeps running)", "")
	quitAll := systray.AddMenuItem("Stop server and quit", "")

	var latest stateResp
	running := false
	refresh := func() {
		ok, _ := pingSettings(port)
		running = ok
		if !ok {
			status.SetTitle("Server not running")
			startStop.SetTitle("Start server")
			systray.SetTooltip("PC Monitor (stopped)")
			return
		}
		startStop.SetTitle("Stop server")
		if err := settingsCall(port, "GET", "/api/state", nil, &latest); err != nil {
			status.SetTitle("Running")
			return
		}
		s := "Running"
		if latest.Pairing.Control {
			s = fmt.Sprintf("Running · %d dashboard(s) paired", latest.Pairing.Paired)
		}
		status.SetTitle(s)
		systray.SetTooltip("PC Monitor " + latest.Version + " · " + s)
		if latest.Update.Available && latest.Update.CanApply {
			updateItem.SetTitle("Update to v" + latest.Update.Latest + "…")
		} else if latest.Update.Available {
			updateItem.SetTitle("v" + latest.Update.Latest + " available")
		} else {
			updateItem.SetTitle("Check for updates")
		}
		if service.AutostartEnabled() {
			autostart.Check()
		} else {
			autostart.Uncheck()
		}
	}
	refresh()
	ticker := time.NewTicker(10 * time.Second)

	go func() {
		for {
			select {
			case <-ticker.C:
				refresh()
			case <-openSettings.ClickedCh:
				_ = settingsCommand()
			case <-openDash.ClickedCh:
				if latest.Pairing.URL != "" {
					_ = service.OpenBrowser(latest.Pairing.URL)
				}
			case <-autostart.ClickedCh:
				if autostart.Checked() {
					_ = service.AutostartDisable()
				} else {
					_ = service.AutostartEnable()
				}
				refresh()
			case <-updateItem.ClickedCh:
				if latest.Update.Available && latest.Update.CanApply {
					updateItem.SetTitle("Updating…")
					updateItem.Disable()
					_ = settingsCall(port, "POST", "/api/update", map[string]string{"action": "apply"}, nil)
					updateItem.Enable()
				} else if latest.Update.Available {
					_ = service.OpenBrowser(latest.Update.URL)
				} else {
					_ = settingsCall(port, "POST", "/api/update", map[string]string{"action": "check"}, nil)
				}
				time.Sleep(2 * time.Second)
				refresh()
			case <-startStop.ClickedCh:
				if running {
					_ = settingsCall(port, "POST", "/api/stop", nil, nil)
					waitFor(port, false, 10*time.Second)
				} else {
					_, _ = service.StartDetached()
					waitFor(port, true, 20*time.Second)
				}
				refresh()
			case <-quit.ClickedCh:
				systray.Quit()
				return
			case <-quitAll.ClickedCh:
				_ = settingsCall(port, "POST", "/api/stop", nil, nil)
				systray.Quit()
				return
			}
		}
	}()
}
