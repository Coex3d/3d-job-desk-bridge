package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const moonrakerQuery = "/printer/objects/query?webhooks&print_stats&virtual_sdcard&extruder&heater_bed&display_status"

type config struct {
	Origin       string `json:"origin"`
	DeviceID     string `json:"deviceId"`
	DeviceSecret string `json:"deviceSecret"`
	DeskName     string `json:"deskName,omitempty"`
}

type printer struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	APIKey     string `json:"apiKey"`
	AccessCode string `json:"accessCode"`
	Serial     string `json:"serial"`
}

type liveStatus struct {
	ID           string   `json:"id"`
	Online       bool     `json:"online"`
	State        string   `json:"state"`
	FileName     *string  `json:"fileName"`
	Progress     *int     `json:"progress"`
	NozzleTemp   *float64 `json:"nozzleTemp"`
	NozzleTarget *float64 `json:"nozzleTarget"`
	BedTemp      *float64 `json:"bedTemp"`
	BedTarget    *float64 `json:"bedTarget"`
	Message      *string  `json:"message"`
	Source       string   `json:"source"`
}

func main() {
	trayOnly := false
	args := make([]string, 0, len(os.Args[1:]))
	for _, arg := range os.Args[1:] {
		if arg == "--tray" || arg == "-tray" {
			trayOnly = true
			continue
		}
		args = append(args, arg)
	}
	if len(args) > 0 {
		switch args[0] {
		case "pair":
			if code := runPair(args[1:]); code != 0 {
				pauseOnError()
				os.Exit(code)
			}
			os.Exit(runLoop())
		case "run", "start":
			os.Exit(runLoop())
		}
	}
	os.Exit(runApp(trayOnly))
}

func runPair(args []string) int {
	fs := flag.NewFlagSet("pair", flag.ExitOnError)
	origin := fs.String("url", defaultOrigin, "Desk website")
	code := fs.String("code", "", "Pairing code from Printers")
	name := fs.String("name", hostname(), "Name for this computer")
	_ = fs.Parse(args)

	in := bufio.NewScanner(os.Stdin)
	if strings.TrimSpace(*code) == "" {
		fmt.Print("Pairing code from Printers: ")
		if in.Scan() {
			*code = strings.TrimSpace(in.Text())
		}
	}
	cfg, err := claimPairing(*origin, *code, *name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Printf("Paired with %s. Credentials stored in %s\n", cfg.DeskName, configPath())
	fmt.Println("Keeping this window open to report printer status.")
	return 0
}

func claimPairing(origin, code, name string) (config, error) {
	if strings.TrimSpace(origin) == "" {
		origin = defaultOrigin
	}
	originURL, err := normalizeOrigin(origin)
	if err != nil {
		return config{}, err
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return config{}, errors.New("A pairing code is required.")
	}
	if strings.TrimSpace(name) == "" {
		name = hostname()
	}

	body, _ := json.Marshal(map[string]string{
		"code":       code,
		"deviceName": name,
		"origin":     originURL,
	})
	req, err := http.NewRequest(http.MethodPost, originURL+"/api/printers/bridge/claim", bytes.NewReader(body))
	if err != nil {
		return config{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := cloudClient().Do(req)
	if err != nil {
		return config{}, fmt.Errorf("Could not reach the website: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var claimed struct {
		DeviceID     string `json:"deviceId"`
		DeviceSecret string `json:"deviceSecret"`
		DeskName     string `json:"deskName"`
		Error        string `json:"error"`
	}
	if err := json.Unmarshal(raw, &claimed); err != nil {
		return config{}, errors.New("Unexpected response from the website.")
	}
	if res.StatusCode != 200 || claimed.DeviceID == "" || claimed.DeviceSecret == "" {
		if claimed.Error == "" {
			claimed.Error = fmt.Sprintf("Pairing failed (%d).", res.StatusCode)
		}
		return config{}, errors.New(claimed.Error)
	}
	cfg := config{Origin: originURL, DeviceID: claimed.DeviceID, DeviceSecret: claimed.DeviceSecret, DeskName: claimed.DeskName}
	if err := saveConfig(cfg); err != nil {
		return config{}, err
	}
	return cfg, nil
}

func pauseOnError() {
	if runtime.GOOS != "windows" {
		return
	}
	info, err := os.Stdin.Stat()
	if err != nil || info.Mode()&os.ModeCharDevice == 0 {
		return
	}
	fmt.Print("Press Enter to close...")
	_, _ = bufio.NewReader(os.Stdin).ReadBytes('\n')
}

func runLoop() int {
	cfg, err := loadConfig()
	if err != nil {
		fmt.Fprintln(os.Stderr, "This computer is not paired yet.")
		fmt.Fprintln(os.Stderr, "Run the downloaded program and enter a pairing code from Printers.")
		return 1
	}
	fmt.Printf("Printer bridge connected to %s (%s)\n", cfg.Origin, cfg.DeskName)
	for {
		if err := pollOnce(cfg); err != nil {
			reportError(err.Error())
			time.Sleep(3 * time.Second)
		}
	}
}

func pollOnce(cfg config) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cfg.Origin+"/api/printers/bridge", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.DeviceID+"."+cfg.DeviceSecret)
	res, err := cloudClient().Do(req)
	if err != nil {
		return fmt.Errorf("website poll failed: %w", err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode == 401 {
		return errors.New("this computer was disconnected in Printers. Pair again.")
	}
	if res.StatusCode != 200 {
		return fmt.Errorf("website poll failed (%d)", res.StatusCode)
	}
	var payload struct {
		Jobs []struct {
			ID      string  `json:"id"`
			Printer printer `json:"printer"`
		} `json:"jobs"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return errors.New("unexpected poll payload")
	}
	for _, job := range payload.Jobs {
		status, err := probePrinter(job.Printer)
		result := map[string]any{"jobId": job.ID}
		if err != nil {
			result["error"] = err.Error()
			reportError(job.Printer.Name + ": " + err.Error())
		} else {
			result["status"] = status
			reportStatus(job.Printer.Name + ": " + status.State)
		}
		body, _ := json.Marshal(result)
		post, _ := http.NewRequest(http.MethodPost, cfg.Origin+"/api/printers/bridge", bytes.NewReader(body))
		post.Header.Set("Authorization", "Bearer "+cfg.DeviceID+"."+cfg.DeviceSecret)
		post.Header.Set("Content-Type", "application/json")
		pres, perr := cloudClient().Do(post)
		if perr != nil {
			reportError("could not report status: " + perr.Error())
			continue
		}
		io.Copy(io.Discard, pres.Body)
		pres.Body.Close()
	}
	return nil
}

func probePrinter(p printer) (*liveStatus, error) {
	if strings.ToLower(p.Kind) == "bambu" {
		return nil, errors.New("Bambu uses MQTT; use a Tailscale hostname for Bambu or keep Klipper on this bridge")
	}
	host, port, err := printerTarget(p)
	if err != nil {
		return nil, err
	}
	if !allowedPrinterHost(host) {
		return nil, fmt.Errorf("refusing to contact %s — only shop LAN, localhost, or .local names are allowed", host)
	}
	scheme := "http"
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(p.Host)), "https://") {
		scheme = "https"
	}
	u := fmt.Sprintf("%s://%s:%d%s", scheme, host, port, moonrakerQuery)
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	if p.APIKey != "" {
		req.Header.Set("X-Api-Key", p.APIKey)
	}
	res, err := printerClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, fmt.Errorf("Moonraker returned %d", res.StatusCode)
	}
	raw, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	return parseMoonraker(raw, p)
}

func parseMoonraker(raw []byte, p printer) (*liveStatus, error) {
	var payload struct {
		Result struct {
			Status struct {
				Webhooks struct {
					State   string `json:"state"`
					Message string `json:"message"`
				} `json:"webhooks"`
				PrintStats struct {
					State    string `json:"state"`
					Filename string `json:"filename"`
					Message  string `json:"message"`
				} `json:"print_stats"`
				VirtualSD struct {
					Progress *float64 `json:"progress"`
				} `json:"virtual_sdcard"`
				Display struct {
					Progress *float64 `json:"progress"`
				} `json:"display_status"`
				Extruder struct {
					Temperature *float64 `json:"temperature"`
					Target      *float64 `json:"target"`
				} `json:"extruder"`
				Bed struct {
					Temperature *float64 `json:"temperature"`
					Target      *float64 `json:"target"`
				} `json:"heater_bed"`
			} `json:"status"`
		} `json:"result"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, errors.New("Moonraker response was not JSON")
	}
	st := payload.Result.Status
	klippy := st.Webhooks.State
	if klippy == "" {
		klippy = "unknown"
	}
	printState := st.PrintStats.State
	if printState == "" {
		if klippy == "ready" {
			printState = "standby"
		} else {
			printState = klippy
		}
	}
	var progress *int
	if st.VirtualSD.Progress != nil {
		v := int(*st.VirtualSD.Progress * 100)
		progress = &v
	} else if st.Display.Progress != nil {
		v := int(*st.Display.Progress * 100)
		progress = &v
	}
	online := klippy == "ready" || printState == "printing" || printState == "paused"
	msg := st.PrintStats.Message
	if msg == "" {
		msg = st.Webhooks.Message
	}
	var filePtr *string
	if st.PrintStats.Filename != "" {
		f := st.PrintStats.Filename
		filePtr = &f
	}
	var msgPtr *string
	if msg != "" {
		msgPtr = &msg
	}
	return &liveStatus{
		ID:           p.ID,
		Online:       online,
		State:        printState,
		FileName:     filePtr,
		Progress:     progress,
		NozzleTemp:   st.Extruder.Temperature,
		NozzleTarget: st.Extruder.Target,
		BedTemp:      st.Bed.Temperature,
		BedTarget:    st.Bed.Target,
		Message:      msgPtr,
		Source:       "klipper",
	}, nil
}

func printerTarget(p printer) (string, int, error) {
	raw := strings.TrimSpace(p.Host)
	raw = strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	hostport := strings.Split(raw, "/")[0]
	host, portStr, err := net.SplitHostPort(hostport)
	if err != nil {
		host = strings.TrimSuffix(strings.TrimPrefix(hostport, "["), "]")
		portStr = ""
	}
	port := p.Port
	if port == 0 && portStr != "" {
		port, _ = strconv.Atoi(portStr)
	}
	if port == 0 {
		port = 7125
	}
	if port < 1 || port > 65535 {
		return "", 0, errors.New("invalid printer port")
	}
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return "", 0, errors.New("missing printer host")
	}
	return host, port, nil
}

func allowedPrinterHost(host string) bool {
	if host == "localhost" || host == "127.0.0.1" || host == "::1" || strings.HasSuffix(host, ".local") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsPrivate() {
		return true
	}
	if ip.IsLinkLocalUnicast() || ip.IsMulticast() || ip.IsUnspecified() {
		return false
	}
	return false
}

func normalizeOrigin(raw string) (string, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", errors.New("website URL is required")
	}
	if !strings.Contains(value, "://") {
		value = "https://" + value
	}
	u, err := url.Parse(value)
	if err != nil || u.Host == "" {
		return "", errors.New("that website URL is not valid")
	}
	if u.Scheme == "https" {
		return strings.TrimRight(u.Scheme+"://"+u.Host, "/"), nil
	}
	host := u.Hostname()
	if u.Scheme == "http" && (host == "127.0.0.1" || host == "localhost") {
		return strings.TrimRight(u.Scheme+"://"+u.Host, "/"), nil
	}
	return "", errors.New("the bridge only connects over HTTPS (localhost HTTP is allowed for testing)")
}

func cloudClient() *http.Client {
	return &http.Client{
		Timeout: 35 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{MinVersion: tls.VersionTLS12},
		},
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("refusing redirect")
		},
	}
}

func printerClient() *http.Client {
	return &http.Client{
		Timeout: 6 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return errors.New("refusing printer redirect")
		},
	}
}

func configPath() string {
	if configPathOverride != "" {
		return configPathOverride
	}
	if home, err := os.UserConfigDir(); err == nil && home != "" {
		return filepath.Join(home, "3d-job-desk", "bridge.json")
	}
	u, _ := user.Current()
	base := "."
	if u != nil && u.HomeDir != "" {
		base = u.HomeDir
	}
	return filepath.Join(base, ".3d-job-desk", "bridge.json")
}

func saveConfig(cfg config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	raw, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0o600)
}

func loadConfig() (config, error) {
	var cfg config
	raw, err := os.ReadFile(configPath())
	if err != nil {
		return cfg, err
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return cfg, err
	}
	if cfg.Origin == "" || cfg.DeviceID == "" || cfg.DeviceSecret == "" {
		return cfg, errors.New("incomplete pairing file")
	}
	if _, err := normalizeOrigin(cfg.Origin); err != nil {
		return cfg, err
	}
	cfg.Origin = strings.TrimRight(cfg.Origin, "/")
	return cfg, nil
}

func hostname() string {
	name, err := os.Hostname()
	if err != nil || name == "" {
		return "Shop computer"
	}
	return name
}

var configPathOverride string
