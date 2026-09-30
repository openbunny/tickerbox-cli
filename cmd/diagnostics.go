// SPDX-License-Identifier: MIT

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/client"
	"github.com/openbunny/tickerbox-cli/internal/config"
	"github.com/openbunny/tickerbox-cli/internal/output"
)

type severity int

const (
	severityOK severity = iota
	severityWarn
	severityCritical
)

func (s severity) String() string {
	switch s {
	case severityOK:
		return "ok"
	case severityWarn:
		return "warn"
	case severityCritical:
		return "critical"
	default:
		panic(fmt.Sprintf("severity: unhandled value %d", int(s)))
	}
}

type doctorCheck struct {
	Name     string `json:"name"`
	Severity string `json:"severity"`
	Detail   string `json:"detail"`
}

type doctorReport struct {
	Device  string        `json:"device"`
	Verdict string        `json:"verdict"`
	Checks  []doctorCheck `json:"checks"`
}

const (
	doctorFreeHeapCriticalBytes   = 20 * 1024
	doctorFreeHeapWarnBytes       = 50 * 1024
	doctorFsHeadroomCriticalBytes = 8 * 1024
	doctorFsHeadroomWarnBytes     = 32 * 1024

	doctorWifiConnectedStatus = 3 // matches wifiStatusText's CONNECTED
	doctorNtpActiveStatus     = 1 // matches ntpStatusText's ACTIVE
	doctorApActiveStatus      = 0 // matches apStatusText's ACTIVE
	doctorApLingeringStatus   = 2 // matches apStatusText's LINGERING
)

const doctorSecretExposureDetail = "GET /rest/wifiSettings and GET /rest/apSettings return the WiFi and AP passwords in plaintext; the device REST API has no authentication, so any client on the LAN can read them"

func secretExposureCheck() doctorCheck {
	return doctorCheck{Name: "plaintext credentials", Severity: severityWarn.String(), Detail: doctorSecretExposureDetail}
}

func evaluateFreeHeap(freeHeap int64) doctorCheck {
	switch {
	case freeHeap < doctorFreeHeapCriticalBytes:
		return doctorCheck{
			Name: "free heap", Severity: severityCritical.String(),
			Detail: fmt.Sprintf("%s free, below the %s critical floor", humanizeBytes(freeHeap), humanizeBytes(doctorFreeHeapCriticalBytes)),
		}
	case freeHeap < doctorFreeHeapWarnBytes:
		return doctorCheck{
			Name: "free heap", Severity: severityWarn.String(),
			Detail: fmt.Sprintf("%s free, below the %s warning floor", humanizeBytes(freeHeap), humanizeBytes(doctorFreeHeapWarnBytes)),
		}
	default:
		return doctorCheck{Name: "free heap", Severity: severityOK.String(), Detail: humanizeBytes(freeHeap) + " free"}
	}
}

func evaluateFsHeadroom(total, used int64) doctorCheck {
	headroom := total - used
	switch {
	case headroom < doctorFsHeadroomCriticalBytes:
		return doctorCheck{
			Name: "fs headroom", Severity: severityCritical.String(),
			Detail: fmt.Sprintf("%s free, below the %s critical floor", humanizeBytes(headroom), humanizeBytes(doctorFsHeadroomCriticalBytes)),
		}
	case headroom < doctorFsHeadroomWarnBytes:
		return doctorCheck{
			Name: "fs headroom", Severity: severityWarn.String(),
			Detail: fmt.Sprintf("%s free, below the %s warning floor", humanizeBytes(headroom), humanizeBytes(doctorFsHeadroomWarnBytes)),
		}
	default:
		return doctorCheck{Name: "fs headroom", Severity: severityOK.String(), Detail: humanizeBytes(headroom) + " free"}
	}
}

func evaluateWifi(status int) doctorCheck {
	if status == doctorWifiConnectedStatus {
		return doctorCheck{Name: "wifi", Severity: severityOK.String(), Detail: "connected (" + wifiStatusText(status) + ")"}
	}
	return doctorCheck{Name: "wifi", Severity: severityCritical.String(), Detail: "not connected (" + wifiStatusText(status) + ")"}
}

func evaluateNtp(status int) doctorCheck {
	if status == doctorNtpActiveStatus {
		return doctorCheck{Name: "ntp", Severity: severityOK.String(), Detail: "active (" + ntpStatusText(status) + ")"}
	}
	return doctorCheck{Name: "ntp", Severity: severityWarn.String(), Detail: "not active (" + ntpStatusText(status) + ")"}
}

func evaluateApExposure(status int) doctorCheck {
	switch status {
	case doctorApActiveStatus, doctorApLingeringStatus:
		return doctorCheck{
			Name: "ap exposure", Severity: severityWarn.String(),
			Detail: "access point is broadcasting (" + apStatusText(status) + "); anyone in range can reach the unauthenticated REST API",
		}
	default:
		return doctorCheck{Name: "ap exposure", Severity: severityOK.String(), Detail: "access point inactive (" + apStatusText(status) + ")"}
	}
}

func fetchFailureCheck(name string, err error) doctorCheck {
	return doctorCheck{Name: name, Severity: severityCritical.String(), Detail: err.Error()}
}

func aggregateSeverity(checks []doctorCheck) severity {
	worst := severityOK
	for _, c := range checks {
		switch c.Severity {
		case severityCritical.String():
			return severityCritical
		case severityWarn.String():
			worst = severityWarn
		}
	}
	return worst
}

func runDoctor(c *client.Client) doctorReport {
	checks := []doctorCheck{secretExposureCheck()}

	var features featuresPayload
	if err := c.Get("features", &features); err != nil {
		checks = append(checks, fetchFailureCheck("features", err))
	}

	var sys systemStatusPayload
	if err := c.Get("systemStatus", &sys); err != nil {
		checks = append(checks, fetchFailureCheck("systemStatus", err))
	} else {
		checks = append(checks, evaluateFreeHeap(sys.FreeHeap), evaluateFsHeadroom(sys.FsTotal, sys.FsUsed))
	}

	var wifi dashboardWifiStatus
	if err := c.Get("wifiStatus", &wifi); err != nil {
		checks = append(checks, fetchFailureCheck("wifiStatus", err))
	} else {
		checks = append(checks, evaluateWifi(wifi.Status))
	}

	var ap dashboardApStatus
	if err := c.Get("apStatus", &ap); err != nil {
		checks = append(checks, fetchFailureCheck("apStatus", err))
	} else {
		checks = append(checks, evaluateApExposure(ap.Status))
	}

	var ntp dashboardNtpStatus
	if err := c.Get("ntpStatus", &ntp); err != nil {
		checks = append(checks, fetchFailureCheck("ntpStatus", err))
	} else {
		checks = append(checks, evaluateNtp(ntp.Status))
	}

	return doctorReport{Verdict: aggregateSeverity(checks).String(), Checks: checks}
}

func printDoctorReport(r doctorReport) {
	fmt.Printf("%s - %s\n", r.Device, strings.ToUpper(r.Verdict))
	rows := make([][]string, len(r.Checks))
	for i, c := range r.Checks {
		rows[i] = []string{c.Name, c.Severity, c.Detail}
	}
	output.Table([]string{"CHECK", "SEVERITY", "DETAIL"}, rows)
}

var doctorAll bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Read-only health check",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		var reports []doctorReport

		if doctorAll {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("load device config: %w", err)
			}
			devices := cfg.List()
			if len(devices) == 0 {
				return errors.New("no devices configured: --all has nothing to check")
			}
			for _, d := range devices {
				c := client.New(restBase(d.Host), timeoutFlag)
				c.Retries = retryFlag
				report := runDoctor(c)
				report.Device = d.Name
				reports = append(reports, report)
			}
		} else {
			report := runDoctor(newClient())
			report.Device = resolvedHost
			reports = append(reports, report)
		}

		if jsonOut() {
			if err := output.EmitJSON(reports); err != nil {
				return err
			}
		} else {
			for i, r := range reports {
				if i > 0 {
					fmt.Println()
				}
				printDoctorReport(r)
			}
		}

		for _, r := range reports {
			if r.Verdict == severityCritical.String() {
				return fmt.Errorf("doctor: critical check failed on %s", r.Device)
			}
		}
		return nil
	},
}

const defaultWatchInterval = 2 * time.Second

const ansiClearScreen = "\x1b[H\x1b[2J"

var watchAllowed = map[*cobra.Command]bool{
	statusCmd:          true,
	systemInfoCmd:      true,
	systemFeaturesCmd:  true,
	wifiStatusCmd:      true,
	wifiSettingsCmd:    true,
	apStatusCmd:        true,
	apSettingsCmd:      true,
	ntpStatusCmd:       true,
	ntpSettingsCmd:     true,
	displaySettingsCmd: true,
	clockSettingsCmd:   true,
	tzListCmd:          true,
	tickersListCmd:     true,
}

func resolveWatchTarget(args []string) (*cobra.Command, error) {
	if len(args) == 0 {
		return statusCmd, nil
	}

	target, remaining, err := rootCmd.Find(args)
	if err != nil {
		return nil, fmt.Errorf("resolve watch target %q: %w", strings.Join(args, " "), err)
	}
	if len(remaining) > 0 {
		return nil, fmt.Errorf("watch does not accept arguments after the target command, got %q", strings.Join(remaining, " "))
	}
	if target.RunE == nil || !watchAllowed[target] {
		return nil, fmt.Errorf("%q is not a read-only view watch can re-run", target.CommandPath())
	}
	return target, nil
}

var watchInterval time.Duration

var watchCmd = &cobra.Command{
	Use:   "watch [command args...]",
	Short: "Re-run a read-only view on an interval until interrupted",
	Args:  cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveWatchTarget(args)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(cmd.Context(), os.Interrupt)
		defer stop()

		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		for {
			fmt.Print(ansiClearScreen)
			if err := target.RunE(target, nil); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	},
}

const (
	unsetVersion = "dev"
	unsetCommit  = "none"
	unsetDate    = "unknown"
)

var (
	version = unsetVersion
	commit  = unsetCommit
	date    = unsetDate
)

func buildSetting(info *debug.BuildInfo, key string) (string, bool) {
	if info == nil {
		return "", false
	}
	idx := slices.IndexFunc(info.Settings, func(s debug.BuildSetting) bool {
		return s.Key == key
	})
	if idx == -1 {
		return "", false
	}
	return info.Settings[idx].Value, true
}

func assembleVersion(v, c, d string, info *debug.BuildInfo, ok bool) string {
	if v == unsetVersion && ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		v = info.Main.Version
	}
	if c == unsetCommit && ok {
		if rev, found := buildSetting(info, "vcs.revision"); found {
			c = rev
			if dirty, found := buildSetting(info, "vcs.modified"); found && dirty == "true" {
				c += "-dirty"
			}
		}
	}
	if d == unsetDate && ok {
		if t, found := buildSetting(info, "vcs.time"); found {
			d = t
		}
	}
	return fmt.Sprintf("tickerbox %s (commit %s, built %s)", v, c, d)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the CLI version",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		info, ok := debug.ReadBuildInfo()
		fmt.Println(assembleVersion(version, commit, date, info, ok))
		return nil
	},
}

func deviceUIURL(host string) string {
	return strings.TrimRight(host, "/") + "/"
}

const (
	goosDarwin  = "darwin"
	goosLinux   = "linux"
	goosWindows = "windows"
)

func browserOpenCommand(goos string) (name string, args []string, ok bool) {
	switch goos {
	case goosDarwin:
		return "open", nil, true
	case goosLinux:
		return "xdg-open", nil, true
	case goosWindows:
		// "start" is a cmd.exe builtin; the empty quoted arg is its window
		// title parameter, required whenever the target itself is quoted.
		return "cmd", []string{"/c", "start", ""}, true
	default:
		return "", nil, false
	}
}

var openCmd = &cobra.Command{
	Use:     "open",
	Aliases: []string{"ui"},
	Short:   "Open the device's web UI in the default browser",
	Args:    cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target := deviceUIURL(resolvedHost)

		name, launchArgs, ok := browserOpenCommand(runtime.GOOS)
		if ok {
			if err := exec.Command(name, append(launchArgs, target)...).Start(); err == nil {
				return nil
			}
		}
		fmt.Println(target)
		return nil
	},
}

func init() {
	doctorCmd.Flags().BoolVar(&doctorAll, "all", false, "run against every device configured in the device config file")
	watchCmd.Flags().DurationVar(&watchInterval, "interval", defaultWatchInterval, "how often to refresh")

	rootCmd.AddCommand(doctorCmd, watchCmd, versionCmd, openCmd)
}
