// SPDX-License-Identifier: MIT

package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/concurrent"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/output"
)

const (
	discoverMDNSHost = "tickerbox.local"

	httpScheme = "http://"

	discoverProbeTimeout = 800 * time.Millisecond
	discoverWorkers      = 32
	pingTimeout          = 3 * time.Second

	subnetFirstHost = 1
	subnetLastHost  = 254
)

type deviceListEntry struct {
	Name    string `json:"name"`
	Host    string `json:"host"`
	Default bool   `json:"default"`
}

type deviceCurrentInfo struct {
	Host           string `json:"host"`
	Source         string `json:"source"`
	Device         string `json:"device"`
	DefaultDevice  string `json:"default_device"`
	MatchesDefault bool   `json:"matches_default"`
}

type discoveredDevice struct {
	Host     string `json:"host"`
	Hostname string `json:"hostname"`
}

type pingResult struct {
	Name      string `json:"name"`
	Host      string `json:"host"`
	Reachable bool   `json:"reachable"`
	RTTMillis int64  `json:"rtt_ms"`
}

var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Manage configured devices",
	Long: "Manages named devices this CLI can target: add one with `device add <name> <host>`, pick which is " +
		"used by default with `device use <name>`, or point a single call at any device with --host or " +
		"--device without changing the default. See `tickerbox profile` for named, device-agnostic setting " +
		"bundles, and `tickerbox config` for a raw export/import/diff of one device's live snapshot.",
}

var deviceListCmd = &cobra.Command{
	Use:     "list",
	Short:   "List configured devices",
	Example: "  tickerbox device list",
	Args:    cobra.NoArgs,
	RunE:    runDeviceList,
}

var deviceAddCmd = &cobra.Command{
	Use:   "add <name> <host>",
	Short: "Add or replace a configured device",
	Long: "Adds name as a device, or replaces it if the name already exists. host must include a scheme, " +
		"e.g. http://tickerbox.local or http://192.168.1.42.",
	Example: "  tickerbox device add desk http://tickerbox.local",
	Args:    cobra.ExactArgs(2),
	RunE:    runDeviceAdd,
}

var deviceRmYes bool

var deviceRmCmd = &cobra.Command{
	Use:     "rm <name>",
	Short:   "Remove a configured device",
	Long:    "Removes a configured device. Prompts for confirmation unless --yes.",
	Example: "  tickerbox device rm desk --yes",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !deviceRmYes {
			ok, err := confirm(fmt.Sprintf("Remove device %s?", args[0]))
			if err != nil {
				return err
			}
			if !ok {
				fmt.Println("aborted")
				return nil
			}
		}
		return runDeviceRm(cmd, args)
	},
}

var deviceUseCmd = &cobra.Command{
	Use:     "use <name>",
	Short:   "Set the default device",
	Long:    "Sets NAME as the device used when neither --host, --device, nor $TICKERBOX_HOST is given.",
	Example: "  tickerbox device use desk",
	Args:    cobra.ExactArgs(1),
	RunE:    runDeviceUse,
}

var deviceCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the device the CLI currently targets",
	Long: "Shows the host that resolves from --host, --device, $TICKERBOX_HOST, or the config default, in " +
		"that order, and whether it matches the stored default device.",
	Example: "  tickerbox device current",
	Args:    cobra.NoArgs,
	RunE:    runDeviceCurrent,
}

var deviceDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Find TickerBoxes on the local network",
	Long: "Probes tickerbox.local, then scans every host on the local /24 subnet for a TickerBox. " +
		"For each one found, prompts interactively to add it as a named device.",
	Example: "  tickerbox device discover",
	Args:    cobra.NoArgs,
	RunE:    runDeviceDiscover,
}

var devicePingAll bool

var devicePingCmd = &cobra.Command{
	Use:   "ping [name]",
	Short: "Check reachability of one or all configured devices",
	Long:  "Checks whether a configured device answers. Requires exactly one of NAME or --all.",
	Example: "  tickerbox device ping desk\n" +
		"  tickerbox device ping --all",
	Args: cobra.MaximumNArgs(1),
	RunE: runDevicePing,
}

func init() {
	devicePingCmd.Flags().BoolVar(&devicePingAll, "all", false, "ping every configured device")

	deviceRmCmd.Flags().BoolVarP(&deviceRmYes, "yes", "y", false, "skip confirmation")

	deviceCmd.AddCommand(deviceListCmd, deviceCurrentCmd, deviceAddCmd, deviceRmCmd, deviceUseCmd, deviceDiscoverCmd, devicePingCmd)
	rootCmd.AddCommand(deviceCmd)
}

func hostSourceLabel(s config.HostSource) string {
	switch s {
	case config.SourceHostFlag:
		return "--host"
	case config.SourceDeviceFlag:
		return "--device"
	case config.SourceEnv:
		return "$TICKERBOX_HOST"
	case config.SourceDefault:
		return "default"
	default:
		return "unknown"
	}
}

func runDeviceCurrent(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}
	resolved, err := cfg.Resolve(hostFlag, deviceFlag)
	if err != nil {
		return fmt.Errorf("resolve target host: %w", err)
	}

	matches := false
	if d, ok := cfg.Devices[cfg.Default]; ok {
		matches = resolved.Host == d.Host
	}

	info := deviceCurrentInfo{
		Host:           resolved.Host,
		Source:         hostSourceLabel(resolved.Source),
		Device:         resolved.DeviceName,
		DefaultDevice:  cfg.Default,
		MatchesDefault: matches,
	}

	if jsonOut() {
		return output.EmitJSON(info)
	}

	device := info.Device
	if device == "" {
		device = "-"
	}
	defaultDevice := info.DefaultDevice
	if defaultDevice == "" {
		defaultDevice = "-"
	}
	matchesText := "no"
	if info.MatchesDefault {
		matchesText = "yes"
	}
	return output.KV([][2]string{
		{"host", info.Host},
		{"source", info.Source},
		{"device", device},
		{"default_device", defaultDevice},
		{"matches_default", matchesText},
	})
}

func runDeviceList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}

	devices := cfg.List()
	entries := make([]deviceListEntry, len(devices))
	for i, d := range devices {
		entries[i] = deviceListEntry{Name: d.Name, Host: d.Host, Default: d.Name == cfg.Default}
	}

	if jsonOut() {
		return output.EmitJSON(entries)
	}

	rows := make([][]string, len(entries))
	for i, e := range entries {
		marker := ""
		if e.Default {
			marker = "*"
		}
		rows[i] = []string{e.Name, e.Host, marker}
	}
	return output.Table([]string{"NAME", "HOST", "DEFAULT"}, rows)
}

func runDeviceAdd(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}
	if err := cfg.Add(args[0], args[1]); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save device config: %w", err)
	}
	fmt.Printf("added device %s (%s)\n", args[0], args[1])
	return nil
}

func runDeviceRm(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}
	if err := cfg.Remove(args[0]); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save device config: %w", err)
	}
	fmt.Printf("removed device %s\n", args[0])
	return nil
}

func runDeviceUse(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}
	if err := cfg.SetDefault(args[0]); err != nil {
		return err
	}
	if err := cfg.Save(); err != nil {
		return fmt.Errorf("save device config: %w", err)
	}
	fmt.Printf("default device set to %s\n", args[0])
	return nil
}

func probeTickerbox(ctx context.Context, host string, timeout time.Duration) bool {
	c := client.New(restBase(httpScheme+host), timeout)
	var f featuresPayload
	return c.Get(ctx, "features", &f) == nil && f.Tickerbox
}

func primaryIPv4() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return nil, fmt.Errorf("determine local network address: %w", err)
	}
	defer func() { _ = conn.Close() }()

	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("determine local network address: unexpected address type %T", conn.LocalAddr())
	}
	ip4 := addr.IP.To4()
	if ip4 == nil {
		return nil, fmt.Errorf("determine local network address: %s is not IPv4", addr.IP)
	}
	return ip4, nil
}

func subnetHosts(ip net.IP) []string {
	ip4 := ip.To4()
	if ip4 == nil {
		return nil
	}

	hosts := make([]string, 0, subnetLastHost-subnetFirstHost+1)
	for i := subnetFirstHost; i <= subnetLastHost; i++ {
		candidate := net.IPv4(ip4[0], ip4[1], ip4[2], byte(i))
		if candidate.Equal(ip4) {
			continue
		}
		hosts = append(hosts, candidate.String())
	}
	return hosts
}

func scanSubnet(ctx context.Context, hosts []string, timeout time.Duration, workers int) []discoveredDevice {
	jobs := make(chan string)
	results := make(chan discoveredDevice)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range jobs {
				if probeTickerbox(ctx, host, timeout) {
					results <- discoveredDevice{Host: host}
				}
			}
		}()
	}

	go func() {
		for _, h := range hosts {
			jobs <- h
		}
		close(jobs)
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	var found []discoveredDevice
	for d := range results {
		found = append(found, d)
	}
	return found
}

func lookupHostname(host string) string {
	names, err := net.LookupAddr(host)
	if err != nil || len(names) == 0 {
		return ""
	}
	return strings.TrimSuffix(names[0], ".")
}

func promptLine(prompt string) (string, error) {
	fmt.Print(prompt)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func runDeviceDiscover(cmd *cobra.Command, args []string) error {
	var found []discoveredDevice

	if probeTickerbox(cmdContext(cmd), discoverMDNSHost, discoverProbeTimeout) {
		found = append(found, discoveredDevice{
			Host:     discoverMDNSHost,
			Hostname: strings.TrimSuffix(discoverMDNSHost, ".local"),
		})
	}

	if ip, err := primaryIPv4(); err != nil {
		fmt.Printf("skipping subnet scan: %v\n", err)
	} else {
		found = append(found, scanSubnet(cmdContext(cmd), subnetHosts(ip), discoverProbeTimeout, discoverWorkers)...)
	}

	for i, d := range found {
		if d.Hostname == "" {
			found[i].Hostname = lookupHostname(d.Host)
		}
	}

	if len(found) == 0 {
		fmt.Println("no TickerBoxes found")
		return nil
	}

	if jsonOut() {
		return output.EmitJSON(found)
	}

	rows := make([][]string, len(found))
	for i, d := range found {
		rows[i] = []string{d.Host, d.Hostname}
	}
	if err := output.Table([]string{"HOST", "HOSTNAME"}, rows); err != nil {
		return err
	}

	for _, d := range found {
		add, err := confirm(fmt.Sprintf("Add %s as a device?", d.Host))
		if err != nil {
			return err
		}
		if !add {
			continue
		}

		name, err := promptLine(fmt.Sprintf("Name for %s: ", d.Host))
		if err != nil {
			return err
		}
		if name == "" {
			fmt.Println("skipped: name required")
			continue
		}

		cfg, err := config.Load()
		if err != nil {
			return fmt.Errorf("load device config: %w", err)
		}
		host := httpScheme + d.Host
		if err := cfg.Add(name, host); err != nil {
			return err
		}
		if err := cfg.Save(); err != nil {
			return fmt.Errorf("save device config: %w", err)
		}
		fmt.Printf("added device %s (%s)\n", name, host)
	}

	return nil
}

func pingDevice(ctx context.Context, d config.Device) pingResult {
	c := client.New(restBase(d.Host), pingTimeout)
	start := time.Now()
	err := c.Get(ctx, "features", nil)
	return pingResult{
		Name:      d.Name,
		Host:      d.Host,
		Reachable: err == nil,
		RTTMillis: time.Since(start).Milliseconds(),
	}
}

func runDevicePing(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load device config: %w", err)
	}

	var targets []config.Device
	switch {
	case devicePingAll:
		if len(args) > 0 {
			return fmt.Errorf("--all does not take a device name")
		}
		targets = cfg.List()
	case len(args) == 1:
		d, ok := cfg.Devices[args[0]]
		if !ok {
			return fmt.Errorf("unknown device %q", args[0])
		}
		targets = []config.Device{d}
	default:
		return fmt.Errorf("specify a device name or --all")
	}

	if len(targets) == 0 {
		fmt.Println("no devices configured")
		return nil
	}

	results := make([]pingResult, len(targets))
	tasks := make([]func(), len(targets))
	for i, d := range targets {
		tasks[i] = func() {
			results[i] = pingDevice(cmdContext(cmd), d)
		}
	}
	concurrent.Run(deviceFanOutConcurrency, tasks...)

	if jsonOut() {
		return output.EmitJSON(results)
	}

	rows := make([][]string, len(results))
	for i, r := range results {
		reachable := "no"
		rtt := "-"
		if r.Reachable {
			reachable = "yes"
			rtt = fmt.Sprintf("%dms", r.RTTMillis)
		}
		rows[i] = []string{r.Name, r.Host, reachable, rtt}
	}
	return output.Table([]string{"NAME", "HOST", "REACHABLE", "RTT"}, rows)
}
