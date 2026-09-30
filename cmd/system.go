// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/section"
)

type featuresPayload struct {
	Project        bool `json:"project"`
	Ntp            bool `json:"ntp"`
	Ota            bool `json:"ota"`
	UploadFirmware bool `json:"upload_firmware"`
	Tickerbox      bool `json:"tickerbox"`
	Shopify        bool `json:"shopify"`
	ExtraEtf       bool `json:"extra_etf"`
}

type systemStatusPayload struct {
	EspPlatform     string `json:"esp_platform"`
	MaxAllocHeap    int64  `json:"max_alloc_heap"`
	PsramSize       int64  `json:"psram_size"`
	FreePsram       int64  `json:"free_psram"`
	CPUFreqMHz      int64  `json:"cpu_freq_mhz"`
	FreeHeap        int64  `json:"free_heap"`
	SketchSize      int64  `json:"sketch_size"`
	FreeSketchSpace int64  `json:"free_sketch_space"`
	SdkVersion      string `json:"sdk_version"`
	FlashChipSize   int64  `json:"flash_chip_size"`
	FlashChipSpeed  int64  `json:"flash_chip_speed"`
	FsTotal         int64  `json:"fs_total"`
	FsUsed          int64  `json:"fs_used"`
}

type dashboardWifiStatus struct {
	Status     int    `json:"status"`
	LocalIP    string `json:"local_ip"`
	MacAddress string `json:"mac_address"`
	RSSI       int    `json:"rssi"`
	SSID       string `json:"ssid"`
	BSSID      string `json:"bssid"`
	Channel    int    `json:"channel"`
	SubnetMask string `json:"subnet_mask"`
	GatewayIP  string `json:"gateway_ip"`
	DNSIP1     string `json:"dns_ip_1"`
}

type dashboardApStatus struct {
	Status     int    `json:"status"`
	IPAddress  string `json:"ip_address"`
	MacAddress string `json:"mac_address"`
	StationNum int    `json:"station_num"`
}

type dashboardNtpStatus struct {
	Status    int    `json:"status"`
	UTCTime   string `json:"utc_time"`
	LocalTime string `json:"local_time"`
	Server    string `json:"server"`
	Uptime    int64  `json:"uptime"`
}

type dashboardSettingsState struct {
	Brightness     int    `json:"brightness"`
	ChangeInterval int    `json:"changeInterval"`
	SleepEnabled   bool   `json:"sleepEnabled"`
	SleepStart     string `json:"sleepStart"`
	SleepEnd       string `json:"sleepEnd"`
}

type dashboardClockSetupState struct {
	Enabled          bool   `json:"enabled"`
	TwelveHourFormat bool   `json:"twelweHourFormat"`
	AnimationSpeed   int    `json:"animationSpeed"`
	TzLabel          string `json:"tz_label"`
}

type statusReport struct {
	Features        *featuresPayload          `json:"features,omitempty"`
	SystemStatus    *systemStatusPayload      `json:"systemStatus,omitempty"`
	WifiStatus      *dashboardWifiStatus      `json:"wifiStatus,omitempty"`
	ApStatus        *dashboardApStatus        `json:"apStatus,omitempty"`
	NtpStatus       *dashboardNtpStatus       `json:"ntpStatus,omitempty"`
	SettingsState   *dashboardSettingsState   `json:"settingsState,omitempty"`
	ClockSetupState *dashboardClockSetupState `json:"clockSetupState,omitempty"`
	Errors          map[string]string         `json:"errors,omitempty"`
}

func boolText(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

func humanizeBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	val := float64(n) / unit
	i := 0
	for val >= unit && i < len(units)-1 {
		val /= unit
		i++
	}
	return fmt.Sprintf("%.1f %s", val, units[i])
}

const (
	wifiStatusIdle            = 0
	wifiStatusNoSSIDAvailable = 1
	wifiStatusConnected       = 3
	wifiStatusConnectFailed   = 4
	wifiStatusConnectionLost  = 5
	wifiStatusDisconnected    = 6
	wifiStatusNoShield        = 255
)

const (
	apStatusActive    = 0
	apStatusInactive  = 1
	apStatusLingering = 2
)

const (
	ntpStatusInactive = 0
	ntpStatusActive   = 1
)

func wifiStatusText(status int) string {
	switch status {
	case wifiStatusIdle:
		return "IDLE"
	case wifiStatusNoSSIDAvailable:
		return "NO_SSID_AVAIL"
	case wifiStatusConnected:
		return "CONNECTED"
	case wifiStatusConnectFailed:
		return "CONNECT_FAILED"
	case wifiStatusConnectionLost:
		return "CONNECTION_LOST"
	case wifiStatusDisconnected:
		return "DISCONNECTED"
	case wifiStatusNoShield:
		return "NO_SHIELD"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}

func apStatusText(status int) string {
	switch status {
	case apStatusActive:
		return "ACTIVE"
	case apStatusInactive:
		return "INACTIVE"
	case apStatusLingering:
		return "LINGERING"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}

func ntpStatusText(status int) string {
	switch status {
	case ntpStatusInactive:
		return "INACTIVE"
	case ntpStatusActive:
		return "ACTIVE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}

func confirmSystemAction(prompt string) (bool, error) {
	fmt.Printf("%s [y/N]: ", prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && err != io.EOF {
		return false, fmt.Errorf("read confirmation: %w", err)
	}
	line = strings.TrimSpace(strings.ToLower(line))
	return line == "y" || line == "yes", nil
}

func fetchStatusReport(c *client.Client) statusReport {
	var features featuresPayload
	featuresErr := c.Get("features", &features)

	var sysStatus systemStatusPayload
	sysStatusErr := c.Get("systemStatus", &sysStatus)

	var wifiStatus dashboardWifiStatus
	wifiStatusErr := c.Get("wifiStatus", &wifiStatus)

	var apStatus dashboardApStatus
	apStatusErr := c.Get("apStatus", &apStatus)

	var ntpStatus dashboardNtpStatus
	ntpStatusErr := c.Get("ntpStatus", &ntpStatus)

	var settingsState dashboardSettingsState
	settingsStateErr := c.Get("settingsState", &settingsState)

	var clockSetupState dashboardClockSetupState
	clockSetupStateErr := c.Get("clockSetupState", &clockSetupState)

	report := statusReport{Errors: map[string]string{}}
	if featuresErr == nil {
		report.Features = &features
	} else {
		report.Errors["features"] = featuresErr.Error()
	}
	if sysStatusErr == nil {
		report.SystemStatus = &sysStatus
	} else {
		report.Errors["systemStatus"] = sysStatusErr.Error()
	}
	if wifiStatusErr == nil {
		report.WifiStatus = &wifiStatus
	} else {
		report.Errors["wifiStatus"] = wifiStatusErr.Error()
	}
	if apStatusErr == nil {
		report.ApStatus = &apStatus
	} else {
		report.Errors["apStatus"] = apStatusErr.Error()
	}
	if ntpStatusErr == nil {
		report.NtpStatus = &ntpStatus
	} else {
		report.Errors["ntpStatus"] = ntpStatusErr.Error()
	}
	if settingsStateErr == nil {
		report.SettingsState = &settingsState
	} else {
		report.Errors["settingsState"] = settingsStateErr.Error()
	}
	if clockSetupStateErr == nil {
		report.ClockSetupState = &clockSetupState
	} else {
		report.Errors["clockSetupState"] = clockSetupStateErr.Error()
	}
	if len(report.Errors) == 0 {
		report.Errors = nil
	}
	return report
}

func printStatusReport(report statusReport) {
	fmt.Println("Features")
	if report.Features == nil {
		fmt.Println("  unavailable")
	} else {
		f := report.Features
		output.KV([][2]string{
			{"Project", boolText(f.Project)},
			{"NTP", boolText(f.Ntp)},
			{"OTA", boolText(f.Ota)},
			{"Upload Firmware", boolText(f.UploadFirmware)},
			{"TickerBox", boolText(f.Tickerbox)},
			{"Shopify", boolText(f.Shopify)},
			{"Extra ETF", boolText(f.ExtraEtf)},
		})
	}

	fmt.Println("\nSystem")
	if report.SystemStatus == nil {
		fmt.Println("  unavailable")
	} else {
		s := report.SystemStatus
		output.KV([][2]string{
			{"Platform", s.EspPlatform},
			{"SDK Version", s.SdkVersion},
			{"CPU Frequency", fmt.Sprintf("%d MHz", s.CPUFreqMHz)},
			{"Free Heap", humanizeBytes(s.FreeHeap)},
			{"Flash Chip Size", humanizeBytes(s.FlashChipSize)},
			{"Filesystem", fmt.Sprintf("%s / %s used", humanizeBytes(s.FsUsed), humanizeBytes(s.FsTotal))},
		})
	}

	fmt.Println("\nWi-Fi")
	if report.WifiStatus == nil {
		fmt.Println("  unavailable")
	} else {
		w := report.WifiStatus
		output.KV([][2]string{
			{"Status", wifiStatusText(w.Status)},
			{"SSID", w.SSID},
			{"IP Address", w.LocalIP},
			{"RSSI", fmt.Sprintf("%d dBm", w.RSSI)},
			{"Channel", fmt.Sprintf("%d", w.Channel)},
		})
	}

	fmt.Println("\nAccess Point")
	if report.ApStatus == nil {
		fmt.Println("  unavailable")
	} else {
		a := report.ApStatus
		output.KV([][2]string{
			{"Status", apStatusText(a.Status)},
			{"IP Address", a.IPAddress},
			{"Stations", fmt.Sprintf("%d", a.StationNum)},
		})
	}

	fmt.Println("\nNTP")
	if report.NtpStatus == nil {
		fmt.Println("  unavailable")
	} else {
		n := report.NtpStatus
		output.KV([][2]string{
			{"Status", ntpStatusText(n.Status)},
			{"Server", n.Server},
			{"Local Time", n.LocalTime},
		})
	}

	fmt.Println("\nDisplay")
	if report.SettingsState == nil {
		fmt.Println("  unavailable")
	} else {
		d := report.SettingsState
		output.KV([][2]string{
			{"Brightness", fmt.Sprintf("%d", d.Brightness)},
			{"Change Interval", fmt.Sprintf("%ds", d.ChangeInterval)},
			{"Sleep", boolText(d.SleepEnabled)},
		})
	}

	fmt.Println("\nClock")
	if report.ClockSetupState == nil {
		fmt.Println("  unavailable")
	} else {
		cl := report.ClockSetupState
		output.KV([][2]string{
			{"Enabled", boolText(cl.Enabled)},
			{"12-Hour Format", boolText(cl.TwelveHourFormat)},
			{"Timezone", cl.TzLabel},
		})
	}
}

type deviceStatusReport struct {
	Device string       `json:"device"`
	Host   string       `json:"host"`
	Report statusReport `json:"report"`
}

func collectDeviceStatuses(devices []config.Device) []deviceStatusReport {
	reports := make([]deviceStatusReport, len(devices))
	for i, d := range devices {
		c := client.New(restBase(d.Host), timeoutFlag)
		c.Retries = retryFlag
		reports[i] = deviceStatusReport{Device: d.Name, Host: d.Host, Report: fetchStatusReport(c)}
	}
	return reports
}

func runStatusAll() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}
	devices := cfg.List()
	if len(devices) == 0 {
		return fmt.Errorf("no devices configured in %s", config.Path())
	}

	reports := collectDeviceStatuses(devices)
	if jsonOut() {
		return output.EmitJSON(reports)
	}
	for i, r := range reports {
		if i > 0 {
			fmt.Println()
		}
		fmt.Printf("=== %s (%s) ===\n", r.Device, r.Host)
		printStatusReport(r.Report)
	}
	return nil
}

var statusAll bool

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Aggregate dashboard",
	RunE: func(cmd *cobra.Command, args []string) error {
		if statusAll {
			return runStatusAll()
		}

		report := fetchStatusReport(newClient())
		if jsonOut() {
			return output.EmitJSON(report)
		}
		printStatusReport(report)
		return nil
	},
}

var systemCmd = &cobra.Command{
	Use:   "system",
	Short: "System, status, maintenance",
}

var systemInfoCmd = &cobra.Command{
	Use:   "info",
	Short: "Show device system status",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		var status systemStatusPayload
		if err := c.Get("systemStatus", &status); err != nil {
			return fmt.Errorf("get system status: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(status)
		}
		output.KV([][2]string{
			{"Platform", status.EspPlatform},
			{"SDK Version", status.SdkVersion},
			{"CPU Frequency", fmt.Sprintf("%d MHz", status.CPUFreqMHz)},
			{"Free Heap", humanizeBytes(status.FreeHeap)},
			{"Max Alloc Heap", humanizeBytes(status.MaxAllocHeap)},
			{"PSRAM Size", humanizeBytes(status.PsramSize)},
			{"Free PSRAM", humanizeBytes(status.FreePsram)},
			{"Sketch Size", humanizeBytes(status.SketchSize)},
			{"Free Sketch Space", humanizeBytes(status.FreeSketchSpace)},
			{"Flash Chip Size", humanizeBytes(status.FlashChipSize)},
			{"Flash Chip Speed", fmt.Sprintf("%d MHz", status.FlashChipSpeed/1_000_000)},
			{"Filesystem", fmt.Sprintf("%s / %s used", humanizeBytes(status.FsUsed), humanizeBytes(status.FsTotal))},
		})
		return nil
	},
}

var systemFeaturesCmd = &cobra.Command{
	Use:   "features",
	Short: "Show enabled device features",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		var f featuresPayload
		if err := c.Get("features", &f); err != nil {
			return fmt.Errorf("get features: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(f)
		}
		output.KV([][2]string{
			{"Project", boolText(f.Project)},
			{"NTP", boolText(f.Ntp)},
			{"OTA", boolText(f.Ota)},
			{"Upload Firmware", boolText(f.UploadFirmware)},
			{"TickerBox", boolText(f.Tickerbox)},
			{"Shopify", boolText(f.Shopify)},
			{"Extra ETF", boolText(f.ExtraEtf)},
		})
		return nil
	},
}

var systemRestartYes bool

var systemRestartCmd = &cobra.Command{
	Use:   "restart",
	Short: "Restart the device",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !systemRestartYes {
			ok, err := confirmSystemAction("Restart the device?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := newClient().Post("restart", nil); err != nil {
			return fmt.Errorf("restart device: %w", err)
		}
		fmt.Println("restart requested")
		return nil
	},
}

const (
	backupFilePrefix = "tickerbox-backup-"
	backupTimeLayout = "20060102-150405"
	backupFileExt    = ".json"
	backupFilePerm   = 0o600
)

var systemClock = time.Now

func backupFileName(t time.Time) string {
	return backupFilePrefix + t.UTC().Format(backupTimeLayout) + backupFileExt
}

func writeBackup(c *client.Client) (string, error) {
	snap, err := section.Capture(c, section.Sections, false)
	if err != nil {
		return "", fmt.Errorf("capture config for backup: %w", err)
	}
	encoded, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return "", fmt.Errorf("encode backup: %w", err)
	}
	name := backupFileName(systemClock())
	if err := os.WriteFile(name, encoded, backupFilePerm); err != nil {
		return "", fmt.Errorf("write backup %s: %w", name, err)
	}
	return name, nil
}

var (
	systemFactoryResetYes         bool
	systemFactoryResetBackupFirst bool
)

var systemFactoryResetCmd = &cobra.Command{
	Use:   "factory-reset",
	Short: "Erase all device settings and restore factory defaults",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("WARNING: factory reset erases all device settings and cannot be undone.")
		if !systemFactoryResetYes {
			ok, err := confirmSystemAction("Proceed with factory reset?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}

		c := newClient()
		if systemFactoryResetBackupFirst {
			name, err := writeBackup(c)
			if err != nil {
				return fmt.Errorf("backup before factory reset: %w", err)
			}
			fmt.Printf("backed up config to %s\n", name)
		}

		if err := c.Post("factoryReset", nil); err != nil {
			return fmt.Errorf("factory reset device: %w", err)
		}
		fmt.Println("factory reset requested")
		return nil
	},
}

var systemFirmwareUploadYes bool

var systemFirmwareUploadCmd = &cobra.Command{
	Use:   "firmware-upload FILE",
	Short: "Upload and flash new firmware",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		path := args[0]
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("firmware file %s: %w", path, err)
		}
		if info.IsDir() {
			return fmt.Errorf("firmware file %s is a directory", path)
		}
		if !strings.HasSuffix(strings.ToLower(path), ".bin") {
			return fmt.Errorf("firmware file %s must end in .bin", path)
		}
		fmt.Println("WARNING: uploading firmware replaces the running firmware and cannot be undone.")
		if !systemFirmwareUploadYes {
			ok, err := confirmSystemAction("Proceed with firmware upload?")
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		if err := newClient().PostFile("uploadFirmware", "file", path); err != nil {
			return fmt.Errorf("upload firmware %s: %w", path, err)
		}
		fmt.Println("firmware upload complete")
		return nil
	},
}

func init() {
	statusCmd.Flags().BoolVar(&statusAll, "all", false, "show status for every device in the config file instead of the resolved target")

	systemRestartCmd.Flags().BoolVar(&systemRestartYes, "yes", false, "skip confirmation")
	systemFactoryResetCmd.Flags().BoolVar(&systemFactoryResetYes, "yes", false, "skip confirmation")
	systemFactoryResetCmd.Flags().BoolVar(&systemFactoryResetBackupFirst, "backup-first", false, "export the device config to a timestamped file before wiping it")
	systemFirmwareUploadCmd.Flags().BoolVar(&systemFirmwareUploadYes, "yes", false, "skip confirmation")

	systemCmd.AddCommand(systemInfoCmd, systemFeaturesCmd, systemRestartCmd, systemFactoryResetCmd, systemFirmwareUploadCmd)
	rootCmd.AddCommand(statusCmd, systemCmd)
}
