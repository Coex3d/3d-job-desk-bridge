//go:build windows

package main

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const autoStartValue = "3DJobDeskBridge"
const autoStartKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func autoStartCommand() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	exe, err = filepath.Abs(exe)
	if err != nil {
		return "", err
	}
	return `"` + exe + `" --tray`, nil
}

func isAutoStartEnabled() bool {
	k, err := registry.OpenKey(registry.CURRENT_USER, autoStartKey, registry.QUERY_VALUE)
	if err != nil {
		return false
	}
	defer k.Close()
	_, _, err = k.GetStringValue(autoStartValue)
	return err == nil
}

func setAutoStart(enabled bool) error {
	k, _, err := registry.CreateKey(registry.CURRENT_USER, autoStartKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer k.Close()
	if !enabled {
		_ = k.DeleteValue(autoStartValue)
		return nil
	}
	cmd, err := autoStartCommand()
	if err != nil {
		return err
	}
	return k.SetStringValue(autoStartValue, cmd)
}
