// SPDX-License-Identifier: MIT

package cmd

import (
	"context"
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
	"github.com/openbunny/tickerbox-cli/internal/concurrent"
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

	doctorWifiConnectedStatus = wifiStatusConnected
	doctorNtpActiveStatus     = ntpStatusActive
	doctorApActiveStatus      = apStatusActive
	doctorApLingeringStatus   = apStatusLingering
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

func runDoctor(ctx context.Context, c *client.Client) doctorReport {
	var features featuresPayload
	var featuresErr error

	var sys systemStatusPayload
	var sysErr error

	var wifi dashboardWifiStatus
	var wifiErr error

	var ap dashboardApStatus
	var apErr error

	var ntp dashboardNtpStatus
	var ntpErr error

	concurrent.Run(deviceRequestConcurrency,
		func() { featuresErr = c.Get(ctx, "features", &features) },
		func() { sysErr = c.Get(ctx, "systemStatus", &sys) },
		func() { wifiErr = c.Get(ctx, "wifiStatus", &wifi) },
		func() { apErr = c.Get(ctx, "apStatus", &ap) },
		func() { ntpErr = c.Get(ctx, "ntpStatus", &ntp) },
	)

	checks := []doctorCheck{secretExposureCheck()}

	if featuresErr != nil {
		checks = append(checks, fetchFailureCheck("features", featuresErr))
	}

	if sysErr != nil {
		checks = append(checks, fetchFailureCheck("systemStatus", sysErr))
	} else {
		checks = append(checks, evaluateFreeHeap(sys.FreeHeap), evaluateFsHeadroom(sys.FsTotal, sys.FsUsed))
	}

	if wifiErr != nil {
		checks = append(checks, fetchFailureCheck("wifiStatus", wifiErr))
	} else {
		checks = append(checks, evaluateWifi(wifi.Status))
	}

	if apErr != nil {
		checks = append(checks, fetchFailureCheck("apStatus", apErr))
	} else {
		checks = append(checks, evaluateApExposure(ap.Status))
	}

	if ntpErr != nil {
		checks = append(checks, fetchFailureCheck("ntpStatus", ntpErr))
	} else {
		checks = append(checks, evaluateNtp(ntp.Status))
	}

	return doctorReport{Verdict: aggregateSeverity(checks).String(), Checks: checks}
}

func printDoctorReport(r doctorReport) error {
	fmt.Printf("%s - %s\n", r.Device, strings.ToUpper(r.Verdict))
	rows := make([][]string, len(r.Checks))
	for i, c := range r.Checks {
		rows[i] = []string{c.Name, c.Severity, c.Detail}
	}
	return output.Table([]string{"CHECK", "SEVERITY", "DETAIL"}, rows)
}

var doctorAll bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Read-only health check",
	Long: "Runs read-only checks: free heap, filesystem headroom, wifi connectivity, ntp sync, and ap " +
		"exposure, plus a standing note that the device's REST API has no authentication. Exits non-zero " +
		"if any check is critical. --all runs against every configured device.",
	Example: "  tickerbox doctor\n" +
		"  tickerbox doctor --all --json",
	Args: cobra.NoArgs,
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
			reports = make([]doctorReport, len(devices))
			tasks := make([]func(), len(devices))
			for i, d := range devices {
				tasks[i] = func() {
					c := client.New(restBase(d.Host), timeoutFlag)
					c.Retries = retryFlag
					report := runDoctor(cmdContext(cmd), c)
					report.Device = d.Name
					reports[i] = report
				}
			}
			concurrent.Run(deviceFanOutConcurrency, tasks...)
		} else {
			report := runDoctor(cmdContext(cmd), newClient())
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
				if err := printDoctorReport(r); err != nil {
					return err
				}
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

func terminalStdout() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

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
	Long: "Re-runs a read-only view on an interval until interrupted, clearing the screen each time. Only a " +
		"fixed set of read-only commands can be targeted (status, system info/features, wifi/ap/ntp status " +
		"and settings, display/clock settings, tz list, tickers list); with no arguments it re-runs status.",
	Example: "  tickerbox watch\n" +
		"  tickerbox watch --interval 5s wifi status",
	Args: cobra.ArbitraryArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		target, err := resolveWatchTarget(args)
		if err != nil {
			return err
		}

		ctx, stop := signal.NotifyContext(cmdContext(cmd), os.Interrupt)
		defer stop()
		target.SetContext(ctx)

		ticker := time.NewTicker(watchInterval)
		defer ticker.Stop()

		clearScreen := terminalStdout()
		for {
			if clearScreen {
				fmt.Print(ansiClearScreen)
			}
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
	Use:     "version",
	Short:   "Print the CLI version",
	Example: "  tickerbox version",
	Args:    cobra.NoArgs,
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

const (
	cmdExeName             = "cmd"
	cmdExeRunArg           = "/c"
	cmdExeStartBuiltin     = "start"
	cmdExeStartWindowTitle = ""
)

func browserOpenCommand(goos string) (name string, args []string, ok bool) {
	switch goos {
	case goosDarwin:
		return "open", nil, true
	case goosLinux:
		return "xdg-open", nil, true
	case goosWindows:
		return cmdExeName, []string{cmdExeRunArg, cmdExeStartBuiltin, cmdExeStartWindowTitle}, true
	default:
		return "", nil, false
	}
}

var openCmd = &cobra.Command{
	Use:     "open",
	Aliases: []string{"ui"},
	Short:   "Open the device's web UI in the default browser",
	Example: "  tickerbox open",
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
