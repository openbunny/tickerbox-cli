package cmd

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/output"
	"github.com/openbunny/tickerbox-cli/internal/tz"
)

type apStatusPayload struct {
	Status     int    `json:"status"`
	IPAddress  string `json:"ip_address"`
	MACAddress string `json:"mac_address"`
	StationNum int    `json:"station_num"`
}

type apSettingsPayload struct {
	ProvisionMode int    `json:"provision_mode"`
	SSID          string `json:"ssid"`
	Password      string `json:"password"`
	Channel       int    `json:"channel"`
	SSIDHidden    bool   `json:"ssid_hidden"`
	MaxClients    int    `json:"max_clients"`
	LocalIP       string `json:"local_ip"`
	GatewayIP     string `json:"gateway_ip"`
	SubnetMask    string `json:"subnet_mask"`
}

type ntpStatusPayload struct {
	Status    int    `json:"status"`
	UTCTime   string `json:"utc_time"`
	LocalTime string `json:"local_time"`
	Server    string `json:"server"`
	Uptime    int    `json:"uptime"`
}

type ntpSettingsPayload struct {
	Enabled  bool   `json:"enabled"`
	Server   string `json:"server"`
	TZLabel  string `json:"tz_label"`
	TZFormat string `json:"tz_format"`
}

func apStatusLabel(status int) string {
	switch status {
	case 0:
		return "ACTIVE"
	case 1:
		return "INACTIVE"
	case 2:
		return "LINGERING"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}

func apModeLabel(mode int) string {
	switch mode {
	case 0:
		return "AP always"
	case 1:
		return "AP when wifi disconnected"
	case 2:
		return "AP never"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", mode)
	}
}

func ntpStatusLabel(status int) string {
	switch status {
	case 0:
		return "INACTIVE"
	case 1:
		return "ACTIVE"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", status)
	}
}

const timeLayout = "2006-01-02T15:04:05"

// now returns the current instant in UTC. Tests replace it with a fixed
// clock so the device time-set command is deterministic.
var now = func() time.Time { return time.Now().UTC() }

func parseTimeValue(v string) (time.Time, error) {
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	if t, err := time.Parse(timeLayout, v); err == nil {
		return t, nil
	}
	return time.Time{}, fmt.Errorf("parse time %q: accepted formats are RFC3339 or %s", v, timeLayout)
}

var apCmd = &cobra.Command{
	Use:   "ap",
	Short: "Access point",
}

var apStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show access point status",
	RunE: func(cmd *cobra.Command, args []string) error {
		var st apStatusPayload
		if err := newClient().Get("apStatus", &st); err != nil {
			return fmt.Errorf("get ap status: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(st)
		}
		output.KV([][2]string{
			{"status", apStatusLabel(st.Status)},
			{"ip_address", st.IPAddress},
			{"mac_address", st.MACAddress},
			{"station_num", strconv.Itoa(st.StationNum)},
		})
		return nil
	},
}

var apSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show access point settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		var s apSettingsPayload
		if err := newClient().Get("apSettings", &s); err != nil {
			return fmt.Errorf("get ap settings: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(s)
		}
		output.KV([][2]string{
			{"provision_mode", apModeLabel(s.ProvisionMode)},
			{"ssid", s.SSID},
			{"password", s.Password},
			{"channel", strconv.Itoa(s.Channel)},
			{"ssid_hidden", strconv.FormatBool(s.SSIDHidden)},
			{"max_clients", strconv.Itoa(s.MaxClients)},
			{"local_ip", s.LocalIP},
			{"gateway_ip", s.GatewayIP},
			{"subnet_mask", s.SubnetMask},
		})
		return nil
	},
}

var apSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change access point settings",
}

func init() {
	var (
		mode       string
		ssid       string
		password   string
		channel    int
		hidden     bool
		noHidden   bool
		maxClients int
		localIP    string
		gatewayIP  string
		subnetMask string
	)

	apSetCmd.Flags().StringVar(&mode, "mode", "", "AP mode: always, disconnected, or never")
	apSetCmd.Flags().StringVar(&ssid, "ssid", "", "AP SSID (max 32 chars)")
	apSetCmd.Flags().StringVar(&password, "password", "", "AP password (8-64 chars)")
	apSetCmd.Flags().IntVar(&channel, "channel", 0, "AP channel (1-14)")
	apSetCmd.Flags().BoolVar(&hidden, "hidden", false, "hide the AP SSID")
	apSetCmd.Flags().BoolVar(&noHidden, "no-hidden", false, "broadcast the AP SSID")
	apSetCmd.Flags().IntVar(&maxClients, "max-clients", 0, "max AP clients (1-9)")
	apSetCmd.Flags().StringVar(&localIP, "local-ip", "", "AP local IP")
	apSetCmd.Flags().StringVar(&gatewayIP, "gateway-ip", "", "AP gateway IP")
	apSetCmd.Flags().StringVar(&subnetMask, "subnet-mask", "", "AP subnet mask")

	apSetCmd.RunE = func(cmd *cobra.Command, args []string) error {
		c := newClient()

		var cur apSettingsPayload
		if err := c.Get("apSettings", &cur); err != nil {
			return fmt.Errorf("get ap settings: %w", err)
		}

		flags := cmd.Flags()

		if flags.Changed("mode") {
			switch mode {
			case "always":
				cur.ProvisionMode = 0
			case "disconnected":
				cur.ProvisionMode = 1
			case "never":
				cur.ProvisionMode = 2
			default:
				return fmt.Errorf("invalid --mode %q: must be always, disconnected, or never", mode)
			}
		}
		if flags.Changed("ssid") {
			if len(ssid) > 32 {
				return fmt.Errorf("invalid --ssid: max 32 characters, got %d", len(ssid))
			}
			cur.SSID = ssid
		}
		if flags.Changed("password") {
			if len(password) < 8 || len(password) > 64 {
				return fmt.Errorf("invalid --password: must be 8-64 characters, got %d", len(password))
			}
			cur.Password = password
		}
		if flags.Changed("channel") {
			if channel < 1 || channel > 14 {
				return fmt.Errorf("invalid --channel %d: must be 1-14", channel)
			}
			cur.Channel = channel
		}
		if flags.Changed("hidden") && flags.Changed("no-hidden") {
			return fmt.Errorf("cannot set both --hidden and --no-hidden")
		}
		if flags.Changed("hidden") {
			cur.SSIDHidden = true
		}
		if flags.Changed("no-hidden") {
			cur.SSIDHidden = false
		}
		if flags.Changed("max-clients") {
			if maxClients < 1 || maxClients > 9 {
				return fmt.Errorf("invalid --max-clients %d: must be 1-9", maxClients)
			}
			cur.MaxClients = maxClients
		}
		if flags.Changed("local-ip") {
			cur.LocalIP = localIP
		}
		if flags.Changed("gateway-ip") {
			cur.GatewayIP = gatewayIP
		}
		if flags.Changed("subnet-mask") {
			cur.SubnetMask = subnetMask
		}

		if err := c.Post("apSettings", cur); err != nil {
			return fmt.Errorf("set ap settings: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(cur)
		}
		output.KV([][2]string{
			{"provision_mode", apModeLabel(cur.ProvisionMode)},
			{"ssid", cur.SSID},
			{"channel", strconv.Itoa(cur.Channel)},
			{"ssid_hidden", strconv.FormatBool(cur.SSIDHidden)},
			{"max_clients", strconv.Itoa(cur.MaxClients)},
			{"local_ip", cur.LocalIP},
			{"gateway_ip", cur.GatewayIP},
			{"subnet_mask", cur.SubnetMask},
		})
		return nil
	}
}

var ntpCmd = &cobra.Command{
	Use:   "ntp",
	Short: "NTP / time sync",
}

var ntpStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show NTP sync status",
	RunE: func(cmd *cobra.Command, args []string) error {
		var st ntpStatusPayload
		if err := newClient().Get("ntpStatus", &st); err != nil {
			return fmt.Errorf("get ntp status: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(st)
		}
		output.KV([][2]string{
			{"status", ntpStatusLabel(st.Status)},
			{"utc_time", st.UTCTime},
			{"local_time", st.LocalTime},
			{"server", st.Server},
			{"uptime", strconv.Itoa(st.Uptime)},
		})
		return nil
	},
}

var ntpSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show NTP settings",
	RunE: func(cmd *cobra.Command, args []string) error {
		var s ntpSettingsPayload
		if err := newClient().Get("ntpSettings", &s); err != nil {
			return fmt.Errorf("get ntp settings: %w", err)
		}
		if jsonOut() {
			return output.EmitJSON(s)
		}
		output.KV([][2]string{
			{"enabled", strconv.FormatBool(s.Enabled)},
			{"server", s.Server},
			{"tz_label", s.TZLabel},
			{"tz_format", s.TZFormat},
		})
		return nil
	},
}

var ntpSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Change NTP settings",
}

func init() {
	var (
		enabled  bool
		disabled bool
		server   string
		label    string
	)

	ntpSetCmd.Flags().BoolVar(&enabled, "enabled", false, "enable NTP sync")
	ntpSetCmd.Flags().BoolVar(&disabled, "disabled", false, "disable NTP sync")
	ntpSetCmd.Flags().StringVar(&server, "server", "", "NTP server hostname or IP")
	ntpSetCmd.Flags().StringVar(&label, "tz", "", "timezone label")

	ntpSetCmd.RunE = func(cmd *cobra.Command, args []string) error {
		c := newClient()

		var cur ntpSettingsPayload
		if err := c.Get("ntpSettings", &cur); err != nil {
			return fmt.Errorf("get ntp settings: %w", err)
		}

		flags := cmd.Flags()

		if flags.Changed("enabled") && flags.Changed("disabled") {
			return fmt.Errorf("cannot set both --enabled and --disabled")
		}
		if flags.Changed("enabled") {
			cur.Enabled = true
		}
		if flags.Changed("disabled") {
			cur.Enabled = false
		}
		if flags.Changed("server") {
			cur.Server = server
		}
		if flags.Changed("tz") {
			posix, ok := tz.PosixFor(label)
			if !ok {
				return fmt.Errorf("unknown timezone label %q", label)
			}
			cur.TZLabel = label
			cur.TZFormat = posix
		}

		if err := c.Post("ntpSettings", cur); err != nil {
			return fmt.Errorf("set ntp settings: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(cur)
		}
		output.KV([][2]string{
			{"enabled", strconv.FormatBool(cur.Enabled)},
			{"server", cur.Server},
			{"tz_label", cur.TZLabel},
			{"tz_format", cur.TZFormat},
		})
		return nil
	}
}

var timeCmd = &cobra.Command{
	Use:   "time [VALUE]",
	Short: "Device wall-clock time",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		t := now()
		if len(args) == 1 && args[0] != "" && args[0] != "now" {
			var err error
			t, err = parseTimeValue(args[0])
			if err != nil {
				return err
			}
		}

		formatted := t.UTC().Format(timeLayout)
		body := map[string]string{"local_time": formatted}
		if err := newClient().Post("time", body); err != nil {
			return fmt.Errorf("set device time: %w", err)
		}

		if jsonOut() {
			return output.EmitJSON(body)
		}
		output.KV([][2]string{{"local_time", formatted}})
		return nil
	},
}

func init() {
	apCmd.AddCommand(apStatusCmd, apSettingsCmd, apSetCmd)
	ntpCmd.AddCommand(ntpStatusCmd, ntpSettingsCmd, ntpSetCmd)
	rootCmd.AddCommand(apCmd, ntpCmd, timeCmd)
}
