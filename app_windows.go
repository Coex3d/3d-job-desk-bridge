//go:build windows

package main

import (
	_ "embed"
	"os"
	"path/filepath"
	"runtime"
	"time"
	"unsafe"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
	"golang.org/x/sys/windows"
)

//go:embed winres/icon.ico
var iconICO []byte

const windowTitle = "3D Job Desk Printer Bridge"

var (
	mw          *walk.MainWindow
	codeEdit    *walk.LineEdit
	statusLabel *walk.TextLabel
	autoStartCB *walk.CheckBox
	pairBtn     *walk.PushButton
	ni          *walk.NotifyIcon
	quitting    bool
	hidOnce     bool
	appIcon     *walk.Icon
	mutexHandle windows.Handle
)

func init() {
	runtime.LockOSThread()
}

func runApp(trayOnly bool) int {
	if alreadyRunning() {
		restoreExistingWindow()
		return 0
	}
	appIcon = loadAppIcon()

	visible := true
	if trayOnly {
		if _, err := loadConfig(); err == nil {
			visible = false
		}
	}

	status := "Not paired. Paste a pairing code from Printers."
	if cfg, err := loadConfig(); err == nil {
		status = "Connected to " + cfg.DeskName
	}

	err := (declarative.MainWindow{
		AssignTo: &mw,
		Title:    windowTitle,
		Icon:     appIcon,
		Visible:  visible,
		Size:     declarative.Size{Width: 480, Height: 430},
		MinSize:  declarative.Size{Width: 430, Height: 380},
		Layout:   declarative.VBox{Margins: declarative.Margins{Left: 18, Top: 16, Right: 18, Bottom: 16}, Spacing: 8},
		Children: []declarative.Widget{
			declarative.Label{
				Text: "Printer Bridge",
				Font: declarative.Font{Family: "Segoe UI", PointSize: 14, Bold: true},
			},
			declarative.TextLabel{
				Text:      "This computer reports shop printer status to 3D Job Desk. It only uses outbound HTTPS and does not open a port on your network.",
				TextColor: walk.RGB(40, 40, 40),
			},
			declarative.Label{Text: "Website"},
			declarative.LineEdit{Text: defaultOrigin, ReadOnly: true},
			declarative.Label{Text: "Pairing code from Printers"},
			declarative.Composite{
				Layout: declarative.HBox{MarginsZero: true, Spacing: 8},
				Children: []declarative.Widget{
					declarative.LineEdit{
						AssignTo:  &codeEdit,
						CueBanner: "ABCD-EFGH",
					},
					declarative.PushButton{
						AssignTo:  &pairBtn,
						Text:      "Connect",
						MaxSize:   declarative.Size{Width: 110},
						OnClicked: onPairClicked,
					},
				},
			},
			declarative.TextLabel{AssignTo: &statusLabel, Text: status},
			declarative.CheckBox{
				AssignTo:  &autoStartCB,
				Text:      "Start with Windows",
				Checked:   isAutoStartEnabled(),
				OnClicked: onAutoStartClicked,
			},
			declarative.TextLabel{
				Text:      "Closing this window keeps the bridge running in the system tray. Right-click the tray icon and choose Quit to stop it.",
				TextColor: walk.RGB(80, 80, 80),
			},
		},
	}).Create()
	if err != nil {
		writeLog("window create failed: " + err.Error())
		return 1
	}

	if err := setupTray(); err != nil {
		writeLog("tray icon failed: " + err.Error())
	}
	if autoStartCB != nil {
		autoStartCB.SetChecked(isAutoStartEnabled())
	}
	if bounds := mw.Bounds(); bounds.Width > 0 {
		_ = mw.SetBounds(bounds)
	}

	mw.Closing().Attach(func(canceled *bool, _ walk.CloseReason) {
		if quitting {
			return
		}
		*canceled = true
		hideToTray()
	})

	setStatusHandler(func(msg string) {
		if mw == nil {
			return
		}
		mw.Synchronize(func() {
			if statusLabel != nil {
				_ = statusLabel.SetText(msg)
			}
			if ni != nil {
				_ = ni.SetToolTip("3D Job Desk · " + msg)
			}
		})
	})

	go runBridgeWorker()
	mw.Run()
	return 0
}

func onPairClicked() {
	if codeEdit == nil {
		return
	}
	code := codeEdit.Text()
	if pairBtn != nil {
		pairBtn.SetEnabled(false)
		defer pairBtn.SetEnabled(true)
	}
	cfg, err := claimPairing(defaultOrigin, code, hostname())
	if err != nil {
		walk.MsgBox(mw, "Pairing failed", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
		reportStatus(err.Error())
		return
	}
	msg := "Paired with " + cfg.DeskName + ". Close this window to keep running in the tray."
	reportStatus(msg)
	walk.MsgBox(mw, "Connected", msg, walk.MsgBoxOK|walk.MsgBoxIconInformation)
}

func onAutoStartClicked() {
	if autoStartCB == nil {
		return
	}
	if err := setAutoStart(autoStartCB.Checked()); err != nil {
		walk.MsgBox(mw, "Could not change startup", err.Error(), walk.MsgBoxOK|walk.MsgBoxIconError)
	}
}

func hideToTray() {
	if mw != nil {
		mw.Hide()
	}
	if hidOnce || ni == nil {
		return
	}
	hidOnce = true
	_ = ni.ShowInfo("3D Job Desk", "Printer Bridge is still running in the system tray.")
}

func showMainWindow() {
	if mw == nil {
		return
	}
	mw.Show()
	_ = mw.SetFocus()
}

func setupTray() error {
	var err error
	ni, err = walk.NewNotifyIcon(mw)
	if err != nil {
		return err
	}
	if appIcon != nil {
		_ = ni.SetIcon(appIcon)
	}
	_ = ni.SetToolTip("3D Job Desk Printer Bridge")
	_ = ni.SetVisible(true)

	openAction := walk.NewAction()
	_ = openAction.SetText("Open")
	openAction.Triggered().Attach(showMainWindow)
	_ = ni.ContextMenu().Actions().Add(openAction)
	_ = ni.ContextMenu().Actions().Add(walk.NewSeparatorAction())

	quitAction := walk.NewAction()
	_ = quitAction.SetText("Quit")
	quitAction.Triggered().Attach(func() {
		quitting = true
		if ni != nil {
			_ = ni.Dispose()
		}
		walk.App().Exit(0)
	})
	_ = ni.ContextMenu().Actions().Add(quitAction)

	ni.MouseDown().Attach(func(_ int, _ int, button walk.MouseButton) {
		if button == walk.LeftButton {
			showMainWindow()
		}
	})
	return nil
}

func runBridgeWorker() {
	for {
		cfg, err := loadConfig()
		if err != nil {
			reportStatus("Not paired. Paste a pairing code from Printers.")
			time.Sleep(2 * time.Second)
			continue
		}
		if err := pollOnce(cfg); err != nil {
			reportStatus(err.Error())
			time.Sleep(3 * time.Second)
			continue
		}
		reportStatus("Connected to " + cfg.DeskName)
	}
}

func loadAppIcon() *walk.Icon {
	if ic, err := walk.NewIconFromResource("APP"); err == nil {
		return ic
	}
	tmp := filepath.Join(os.TempDir(), "3d-job-desk-bridge.ico")
	if err := os.WriteFile(tmp, iconICO, 0o644); err == nil {
		if ic, err := walk.NewIconFromFile(tmp); err == nil {
			return ic
		}
	}
	return nil
}

func alreadyRunning() bool {
	name, err := windows.UTF16PtrFromString("Local\\3DJobDeskPrinterBridge")
	if err != nil {
		return false
	}
	handle, err := windows.CreateMutex(nil, false, name)
	mutexHandle = handle
	if err == windows.ERROR_ALREADY_EXISTS {
		return true
	}
	return false
}

func restoreExistingWindow() {
	user32 := windows.NewLazySystemDLL("user32.dll")
	find := user32.NewProc("FindWindowW")
	show := user32.NewProc("ShowWindow")
	setFore := user32.NewProc("SetForegroundWindow")
	title, err := windows.UTF16PtrFromString(windowTitle)
	if err != nil {
		return
	}
	hwnd, _, _ := find.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return
	}
	show.Call(hwnd, 9) // SW_RESTORE
	setFore.Call(hwnd)
}
