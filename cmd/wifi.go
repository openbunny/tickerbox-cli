// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
)

const (
	wifiScanAttempts = 10
	wifiScanInterval = time.Second
)

var secretStdin io.Reader = os.Stdin

var wifiCmd = &cobra.Command{
	Use:   "wifi",
	Short: "Wi-Fi station configuration",
}

var wifiStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current Wi-Fi station status",
	RunE:  runWifiStatus,
}

var wifiSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show Wi-Fi station settings",
	RunE:  runWifiSettings,
}

var wifiSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Update Wi-Fi station settings",
	RunE:  runWifiSet,
}

var wifiScanCmd = &cobra.Command{
	Use:   "scan",
	Short: "Scan for nearby Wi-Fi networks",
	RunE:  runWifiScan,
}

func init() {
	rootCmd.AddCommand(wifiCmd)
	wifiCmd.AddCommand(wifiStatusCmd, wifiSettingsCmd, wifiSetCmd, wifiScanCmd)

	wifiSettingsCmd.Flags().Bool("show-secrets", false, "reveal the Wi-Fi password instead of masking it")

	wifiSetCmd.Flags().String("ssid", "", "Wi-Fi network name (max 32 characters)")
	wifiSetCmd.Flags().String("password", "", "Wi-Fi network password")
	wifiSetCmd.Flags().Bool("password-stdin", false, "read the Wi-Fi network password from stdin")
	wifiSetCmd.Flags().BoolP("yes", "y", false, "skip the plaintext-HTTP password warning")
	wifiSetCmd.Flags().String("hostname", "", "device hostname")
	wifiSetCmd.Flags().Bool("static-ip", false, "use a static IP instead of DHCP")
	wifiSetCmd.Flags().Bool("no-static-ip", false, "use DHCP instead of a static IP")
	wifiSetCmd.Flags().String("local-ip", "", "static local IP address")
	wifiSetCmd.Flags().String("gateway-ip", "", "static gateway IP address")
	wifiSetCmd.Flags().String("subnet-mask", "", "static subnet mask")
	wifiSetCmd.Flags().String("dns1", "", "primary DNS server")
	wifiSetCmd.Flags().String("dns2", "", "secondary DNS server")
}

type wifiStatusResp struct {
	Status     int    `json:"status"`
	LocalIP    string `json:"local_ip"`
	MACAddress string `json:"mac_address"`
	RSSI       int    `json:"rssi"`
	SSID       string `json:"ssid"`
	BSSID      string `json:"bssid"`
	Channel    int    `json:"channel"`
	SubnetMask string `json:"subnet_mask"`
	GatewayIP  string `json:"gateway_ip"`
	DNSIP1     string `json:"dns_ip_1"`
}

type wifiSettingsResp struct {
	SSID           string `json:"ssid"`
	Password       string `json:"password"`
	Hostname       string `json:"hostname"`
	StaticIPConfig bool   `json:"static_ip_config"`
	LocalIP        string `json:"local_ip,omitempty"`
	GatewayIP      string `json:"gateway_ip,omitempty"`
	SubnetMask     string `json:"subnet_mask,omitempty"`
	DNSIP1         string `json:"dns_ip_1,omitempty"`
	DNSIP2         string `json:"dns_ip_2,omitempty"`
}

type wifiNetwork struct {
	SSID           string `json:"ssid"`
	RSSI           int    `json:"rssi"`
	BSSID          string `json:"bssid"`
	Channel        int    `json:"channel"`
	EncryptionType int    `json:"encryption_type"`
}

type wifiNetworksResp struct {
	Networks []wifiNetwork `json:"networks"`
}

func runWifiStatus(cmd *cobra.Command, args []string) error {
	c := newClient()

	var s wifiStatusResp
	if err := c.Get("wifiStatus", &s); err != nil {
		return fmt.Errorf("get wifi status: %w", err)
	}

	if jsonOut() {
		return output.EmitJSON(s)
	}

	output.KV([][2]string{
		{"status", wifiStatusLabel(s.Status)},
		{"ssid", s.SSID},
		{"bssid", s.BSSID},
		{"channel", strconv.Itoa(s.Channel)},
		{"rssi", strconv.Itoa(s.RSSI)},
		{"local_ip", s.LocalIP},
		{"subnet_mask", s.SubnetMask},
		{"gateway_ip", s.GatewayIP},
		{"dns_ip_1", s.DNSIP1},
		{"mac_address", s.MACAddress},
	})
	return nil
}

func wifiStatusLabel(status int) string {
	switch status {
	case 0:
		return "IDLE"
	case 1:
		return "NO_SSID_AVAIL"
	case 3:
		return "CONNECTED"
	case 4:
		return "CONNECT_FAILED"
	case 5:
		return "CONNECTION_LOST"
	case 6:
		return "DISCONNECTED"
	case 255:
		return "NO_SHIELD"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}

func runWifiSettings(cmd *cobra.Command, args []string) error {
	showSecrets, _ := cmd.Flags().GetBool("show-secrets")

	c := newClient()
	var s wifiSettingsResp
	if err := c.Get("wifiSettings", &s); err != nil {
		return fmt.Errorf("get wifi settings: %w", err)
	}
	if !showSecrets {
		s.Password = wifiMaskPassword(s.Password)
	}

	if jsonOut() {
		return output.EmitJSON(s)
	}

	pairs := [][2]string{
		{"ssid", s.SSID},
		{"password", s.Password},
		{"hostname", s.Hostname},
		{"static_ip_config", strconv.FormatBool(s.StaticIPConfig)},
	}
	if s.StaticIPConfig {
		pairs = append(pairs,
			[2]string{"local_ip", s.LocalIP},
			[2]string{"gateway_ip", s.GatewayIP},
			[2]string{"subnet_mask", s.SubnetMask},
			[2]string{"dns_ip_1", s.DNSIP1},
			[2]string{"dns_ip_2", s.DNSIP2},
		)
	}
	output.KV(pairs)
	return nil
}

func wifiMaskPassword(password string) string {
	if password == "" {
		return ""
	}
	return "********"
}

func runWifiSet(cmd *cobra.Command, args []string) error {
	flags := cmd.Flags()

	if flags.Changed("static-ip") && flags.Changed("no-static-ip") {
		return fmt.Errorf("--static-ip and --no-static-ip are mutually exclusive")
	}

	c := newClient()
	var settings map[string]any
	if err := c.Get("wifiSettings", &settings); err != nil {
		return fmt.Errorf("get wifi settings: %w", err)
	}

	if flags.Changed("ssid") {
		ssid, _ := flags.GetString("ssid")
		if len(ssid) > 32 {
			return fmt.Errorf("ssid %q exceeds 32 characters", ssid)
		}
		settings["ssid"] = ssid
	}
	password, passwordChanged, err := resolveSecretValue(cmd, "Wi-Fi password")
	if err != nil {
		return err
	}
	if passwordChanged {
		settings["password"] = password
	}
	if flags.Changed("hostname") {
		hostname, _ := flags.GetString("hostname")
		settings["hostname"] = hostname
	}
	if flags.Changed("static-ip") {
		settings["static_ip_config"] = true
	}
	if flags.Changed("no-static-ip") {
		settings["static_ip_config"] = false
	}
	if flags.Changed("local-ip") {
		localIP, _ := flags.GetString("local-ip")
		settings["local_ip"] = localIP
	}
	if flags.Changed("gateway-ip") {
		gatewayIP, _ := flags.GetString("gateway-ip")
		settings["gateway_ip"] = gatewayIP
	}
	if flags.Changed("subnet-mask") {
		subnetMask, _ := flags.GetString("subnet-mask")
		settings["subnet_mask"] = subnetMask
	}
	if flags.Changed("dns1") {
		dns1, _ := flags.GetString("dns1")
		settings["dns_ip_1"] = dns1
	}
	if flags.Changed("dns2") {
		dns2, _ := flags.GetString("dns2")
		settings["dns_ip_2"] = dns2
	}

	if passwordChanged {
		yes, _ := flags.GetBool("yes")
		warnPlaintextPassword(os.Stderr, resolvedHost, yes)
	}

	if err := c.Post("wifiSettings", settings); err != nil {
		return fmt.Errorf("update wifi settings: %w", err)
	}

	if jsonOut() {
		return output.EmitJSON(settings)
	}
	fmt.Println("wifi settings updated")
	return nil
}

func resolveSecretValue(cmd *cobra.Command, label string) (string, bool, error) {
	flags := cmd.Flags()
	stdinRequested, _ := flags.GetBool("password-stdin")
	value, _ := flags.GetString("password")

	switch {
	case stdinRequested && value != "":
		return "", false, fmt.Errorf("--password and --password-stdin are mutually exclusive")
	case stdinRequested:
		secret, err := readSecretStdin()
		return secret, true, err
	case value != "":
		return value, true, nil
	case terminalStdin():
		secret, err := promptSecret(label)
		if err != nil || secret == "" {
			return "", false, err
		}
		return secret, true, nil
	default:
		return "", false, nil
	}
}

func readSecretStdin() (string, error) {
	data, err := io.ReadAll(secretStdin)
	if err != nil {
		return "", fmt.Errorf("read password from stdin: %w", err)
	}
	return strings.TrimRight(string(data), "\r\n"), nil
}

func promptSecret(label string) (string, error) {
	fmt.Fprintf(os.Stderr, "%s (leave blank to keep unchanged): ", label)
	restore := disableStdinEcho()
	line, err := bufio.NewReader(secretStdin).ReadString('\n')
	restore()
	fmt.Fprintln(os.Stderr)
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read %s: %w", label, err)
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func disableStdinEcho() func() {
	off := exec.Command("stty", "-echo")
	off.Stdin = os.Stdin
	if off.Run() != nil {
		return func() {}
	}
	return func() {
		on := exec.Command("stty", "echo")
		on.Stdin = os.Stdin
		_ = on.Run()
	}
}

func terminalStdin() bool {
	info, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func warnPlaintextPassword(w io.Writer, host string, yes bool) {
	if yes {
		return
	}
	u, err := url.Parse(host)
	if err != nil || u.Scheme != "http" {
		return
	}
	_, _ = fmt.Fprintf(w, "warning: sending the password to %s over plaintext HTTP exposes it to anyone on the network path\n", host)
}

func runWifiScan(cmd *cobra.Command, args []string) error {
	c := newClient()
	if err := c.Get("scanNetworks", nil); err != nil {
		return fmt.Errorf("trigger wifi scan: %w", err)
	}

	var resp wifiNetworksResp
	var lastErr error
	for attempt := 0; attempt < wifiScanAttempts; attempt++ {
		resp = wifiNetworksResp{}
		lastErr = c.Get("listNetworks", &resp)
		if lastErr == nil && len(resp.Networks) > 0 {
			break
		}
		if attempt < wifiScanAttempts-1 {
			time.Sleep(wifiScanInterval)
		}
	}
	if lastErr != nil && len(resp.Networks) == 0 {
		return fmt.Errorf("list wifi networks: %w", lastErr)
	}

	sort.Slice(resp.Networks, func(i, j int) bool {
		return resp.Networks[i].RSSI > resp.Networks[j].RSSI
	})

	if jsonOut() {
		return output.EmitJSON(resp)
	}

	rows := make([][]string, len(resp.Networks))
	for i, n := range resp.Networks {
		rows[i] = []string{n.SSID, strconv.Itoa(n.RSSI), n.BSSID, strconv.Itoa(n.Channel), strconv.Itoa(n.EncryptionType)}
	}
	output.Table([]string{"SSID", "RSSI", "BSSID", "CHANNEL", "ENCRYPTION"}, rows)
	return nil
}
