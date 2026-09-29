package cmd

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/output"
)

const (
	// discoverMDNSHost is the hostname every TickerBox advertises on its
	// local segment; macOS resolves it through the OS mDNS resolver.
	discoverMDNSHost = "tickerbox.local"

	// httpScheme prefixes a bare discovered host before it is probed or
	// persisted, matching the scheme-qualified form every other Host value
	// in the config file uses (see client.DefaultHost).
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
}

var deviceListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured devices",
	Args:  cobra.NoArgs,
	RunE:  runDeviceList,
}

var deviceAddCmd = &cobra.Command{
	Use:   "add NAME HOST",
	Short: "Add or replace a configured device",
	Args:  cobra.ExactArgs(2),
	RunE:  runDeviceAdd,
}

var deviceRmCmd = &cobra.Command{
	Use:   "rm NAME",
	Short: "Remove a configured device",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeviceRm,
}

var deviceUseCmd = &cobra.Command{
	Use:   "use NAME",
	Short: "Set the default device",
	Args:  cobra.ExactArgs(1),
	RunE:  runDeviceUse,
}

var deviceDiscoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Find TickerBoxes on the local network",
	Args:  cobra.NoArgs,
	RunE:  runDeviceDiscover,
}

var devicePingAll bool

var devicePingCmd = &cobra.Command{
	Use:   "ping [NAME]",
	Short: "Check reachability of one or all configured devices",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDevicePing,
}

func init() {
	devicePingCmd.Flags().BoolVar(&devicePingAll, "all", false, "ping every configured device")

	deviceCmd.AddCommand(deviceListCmd, deviceAddCmd, deviceRmCmd, deviceUseCmd, deviceDiscoverCmd, devicePingCmd)
	rootCmd.AddCommand(deviceCmd)
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
	output.Table([]string{"NAME", "HOST", "DEFAULT"}, rows)
	return nil
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

// probeTickerbox reports whether host serves /rest/features with
// tickerbox==true, within timeout. A transport failure, a non-2xx status,
// or a malformed body are all treated as "not a TickerBox" rather than
// propagated, since discover and ping probe many hosts where most are
// expected not to answer.
func probeTickerbox(host string, timeout time.Duration) bool {
	c := client.New(restBase(httpScheme+host), timeout)
	var f featuresPayload
	return c.Get("features", &f) == nil && f.Tickerbox
}

// primaryIPv4 returns the local IPv4 address the OS would use to reach the
// wider network. Dialing UDP performs no handshake and sends no packet; it
// only asks the OS to resolve the outbound route and its local address.
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

// subnetHosts enumerates every host address in ip's /24, from .1 to .254,
// excluding ip itself. It assumes a /24 regardless of the interface's real
// netmask, per the discover command's stated scan scope.
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

// scanSubnet probes each of hosts concurrently, bounded to workers
// goroutines at a time, and returns those that answered as a TickerBox.
func scanSubnet(hosts []string, timeout time.Duration, workers int) []discoveredDevice {
	jobs := make(chan string)
	results := make(chan discoveredDevice)

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for host := range jobs {
				if probeTickerbox(host, timeout) {
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

// lookupHostname resolves host's PTR record, returning "" when host has
// none or the lookup fails; a missing reverse record is expected on most
// home networks and is not an error worth surfacing.
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
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

func runDeviceDiscover(cmd *cobra.Command, args []string) error {
	var found []discoveredDevice

	if probeTickerbox(discoverMDNSHost, discoverProbeTimeout) {
		found = append(found, discoveredDevice{
			Host:     discoverMDNSHost,
			Hostname: strings.TrimSuffix(discoverMDNSHost, ".local"),
		})
	}

	if ip, err := primaryIPv4(); err != nil {
		fmt.Printf("skipping subnet scan: %v\n", err)
	} else {
		found = append(found, scanSubnet(subnetHosts(ip), discoverProbeTimeout, discoverWorkers)...)
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
	output.Table([]string{"HOST", "HOSTNAME"}, rows)

	for _, d := range found {
		add, err := confirmSystemAction(fmt.Sprintf("Add %s as a device?", d.Host))
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

func pingDevice(d config.Device) pingResult {
	c := client.New(restBase(d.Host), pingTimeout)
	start := time.Now()
	err := c.Get("features", nil)
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
	for i, d := range targets {
		results[i] = pingDevice(d)
	}

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
	output.Table([]string{"NAME", "HOST", "REACHABLE", "RTT"}, rows)
	return nil
}
