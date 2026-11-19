//go:build windows

package deck

import (
	"encoding/json"
	"os/exec"
	"syscall"
	"unicode/utf16"
)

// Get-StartApps lists everything in the Start menu: desktop programs, Store
// apps and game shortcuts (e.g. steam://rungameid/...), with AppIDs that
// can be launched through the shell's Apps folder.
const startAppsScript = `[Console]::OutputEncoding=[Text.Encoding]::UTF8
$r = @(Get-StartApps | ForEach-Object { [pscustomobject]@{ id = $_.AppID; name = $_.Name } })
ConvertTo-Json -InputObject $r -Compress`

func listApps() ([]App, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", startAppsScript)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var apps []App
	if err := json.Unmarshal(out, &apps); err != nil {
		return nil, err
	}
	return apps, nil
}

func launchApp(a App) error { return shellOpen(`shell:AppsFolder\`+a.ID, "", "") }

// typeText types text into the focused window with Unicode key events.
func typeText(s string) error {
	var evs []keyboardInput
	for _, r := range s {
		switch r {
		case '\n':
			evs = append(evs, keyEvent(0x0D, false), keyEvent(0x0D, true))
			continue
		case '\t':
			evs = append(evs, keyEvent(0x09, false), keyEvent(0x09, true))
			continue
		}
		for _, unit := range utf16.Encode([]rune{r}) {
			evs = append(evs,
				keyboardInput{typ: inputKeyboard, wScan: unit, dwFlags: keyeventfUnicode},
				keyboardInput{typ: inputKeyboard, wScan: unit, dwFlags: keyeventfUnicode | keyeventfKeyUp})
		}
	}
	// Send in chunks so a long snippet isn't dropped by the input queue.
	for len(evs) > 0 {
		n := min(len(evs), 200)
		if err := sendInputs(evs[:n]); err != nil {
			return err
		}
		evs = evs[n:]
	}
	return nil
}
