//go:build !windows

package main

import (
	"fmt"
)

func runApp(_ bool) int {
	if _, err := loadConfig(); err != nil {
		fmt.Println("3D Job Desk printer bridge")
		fmt.Println("This computer is not paired yet. Enter a pairing code from Printers.")
		fmt.Println()
		if code := runPair(nil); code != 0 {
			return code
		}
	}
	return runLoop()
}
