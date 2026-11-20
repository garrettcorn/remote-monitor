// Command pc_monitor is the PC Monitor server plus a small management CLI.
//
//	pc_monitor [run] [flags]     run in the foreground (default)
//	pc_monitor start | stop | restart | status
//	pc_monitor settings          open the local settings page
//	pc_monitor pin               show the current pairing PIN and link
//	pc_monitor config list | get <key> | set <key> <value> | path
//	pc_monitor autostart on | off | status
//	pc_monitor update [--check | --auto on|off]
//	pc_monitor version
package main

import (
	"fmt"
	"os"
	"strings"

	"pc_monitor/pkg/buildinfo"
	"pc_monitor/pkg/service"
)

func main() {
	args := os.Args[1:]
	cmd := "run"
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd, args = args[0], args[1:]
	} else if guiApp() && len(args) == 0 {
		// The windowless Windows app (pc_monitor-app.exe) shows the tray
		// icon, starting the background server if needed.
		cmd = "tray"
	}
	if trayAvailable {
		service.LoginArgs = []string{"tray"} // start-at-login shows the tray too
	}

	var err error
	switch cmd {
	case "run":
		err = runCommand(args)
	case "start":
		err = startCommand()
	case "stop":
		err = stopCommand()
	case "restart":
		if err = stopCommand(); err == nil {
			err = startCommand()
		}
	case "status":
		err = statusCommand()
	case "settings", "open":
		err = settingsCommand()
	case "pin":
		err = pinCommand()
	case "tray":
		err = trayCommand()
	case "config":
		err = configCommand(args)
	case "autostart":
		err = autostartCommand(args)
	case "update":
		err = updateCommand(args)
	case "version", "--version", "-v":
		fmt.Println("pc_monitor", buildinfo.Version)
	case "help", "-h", "--help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", cmd)
		usage()
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Print(`PC Monitor server ` + buildinfo.Version + `

Usage:
  pc_monitor [run] [flags]          Run in the foreground (shows pairing QR and PIN)
  pc_monitor start                  Start in the background
  pc_monitor stop | restart         Stop or restart the background server
  pc_monitor status                 Show whether it's running, version and updates
  pc_monitor settings               Open the local settings page in your browser
  pc_monitor pin                    Show the current pairing link and PIN
  pc_monitor tray                   Show a tray / menu-bar icon (starts the server if needed)
  pc_monitor config list            Show saved settings (also: get <key>, set <key> <value>, path)
  pc_monitor autostart on|off       Start automatically when you log in
  pc_monitor update [--check]       Install the latest release (or just check)
  pc_monitor update --auto on|off   Install updates automatically (every 6 hours)
  pc_monitor version                Print the version

Run flags (override saved settings; add --save to keep them):
`)
	fs, _ := runFlags(nil)
	fs.PrintDefaults()
}
