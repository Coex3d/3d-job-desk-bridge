//go:build !windows

package main

func isAutoStartEnabled() bool { return false }

func setAutoStart(bool) error { return nil }
