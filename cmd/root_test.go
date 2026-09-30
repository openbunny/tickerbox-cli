// SPDX-License-Identifier: MIT

package cmd

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"

	"github.com/openbunny/tickerbox-cli/internal/config"
)

func TestRestBase(t *testing.T) {
	tests := []struct {
		name string
		host string
		want string
	}{
		{name: "no trailing slash", host: "http://tickerbox.local", want: "http://tickerbox.local/rest/"},
		{name: "trailing slash", host: "http://tickerbox.local/", want: "http://tickerbox.local/rest/"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := restBase(tt.host); got != tt.want {
				t.Errorf("restBase(%q) = %q; want %q", tt.host, got, tt.want)
			}
		})
	}
}

func TestNewClientAppliesFlags(t *testing.T) {
	origHost, origTimeout, origRetry := resolvedHost, timeoutFlag, retryFlag
	defer func() { resolvedHost, timeoutFlag, retryFlag = origHost, origTimeout, origRetry }()

	resolvedHost = "http://example.invalid"
	timeoutFlag = 5 * time.Second
	retryFlag = 4

	c := newClient()
	if c.Retries != retryFlag {
		t.Errorf("got Retries %d; want %d", c.Retries, retryFlag)
	}
}

func TestCommandTargetsDevice(t *testing.T) {
	tests := []struct {
		name string
		cmd  *cobra.Command
		want bool
	}{
		{"a leaf under deviceCmd returns false", deviceUseCmd, false},
		{"deviceCmd itself returns false", deviceCmd, false},
		{"tz list returns false", tzListCmd, false},
		{"tzCmd itself returns true, since tz set is a sibling that needs a device", tzCmd, true},
		{"tz set returns true", tzSetCmd, true},
		{"profile list returns false", profileListCmd, false},
		{"profile show returns false", profileShowCmd, false},
		{"profile rm returns false", profileRmCmd, false},
		{"version returns false", versionCmd, false},
		{"profile save returns true", profileSaveCmd, true},
		{"profile apply returns true", profileApplyCmd, true},
		{"profile diff returns true", profileDiffCmd, true},
		{"a representative device-needing leaf returns true", statusCmd, true},
		{"wifi status returns true", wifiStatusCmd, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := commandTargetsDevice(tt.cmd); got != tt.want {
				t.Errorf("commandTargetsDevice(%s) = %v; want %v", tt.cmd.Name(), got, tt.want)
			}
		})
	}
}

func TestPersistentPreRunESkipsDeviceIndependentCommands(t *testing.T) {
	useTempDeviceConfigDir(t)

	origResolvedHost := resolvedHost
	t.Cleanup(func() { resolvedHost = origResolvedHost })
	resolvedHost = "unchanged"

	if err := rootCmd.PersistentPreRunE(deviceListCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE(deviceListCmd) = %v; want nil (device-independent commands skip resolution)", err)
	}
	if resolvedHost != "unchanged" {
		t.Errorf("resolvedHost = %q; want unchanged, since deviceListCmd never resolves a target", resolvedHost)
	}
}

func TestPersistentPreRunEBanner(t *testing.T) {
	tests := []struct {
		name           string
		devices        map[string]config.Device
		defaultName    string
		json           bool
		wantBanner     bool
		wantSource     config.HostSource
		wantDeviceName string
	}{
		{
			name: "multi-device config prints the banner",
			devices: map[string]config.Device{
				"kitchen": {Name: "kitchen", Host: "http://10.0.0.1"},
				"office":  {Name: "office", Host: "http://10.0.0.2"},
			},
			defaultName:    "kitchen",
			wantBanner:     true,
			wantSource:     config.SourceDefault,
			wantDeviceName: "kitchen",
		},
		{
			name:           "single-device config: no banner",
			devices:        map[string]config.Device{"kitchen": {Name: "kitchen", Host: "http://10.0.0.1"}},
			defaultName:    "kitchen",
			wantBanner:     false,
			wantSource:     config.SourceDefault,
			wantDeviceName: "kitchen",
		},
		{
			name: "multi-device config under --json: no banner",
			devices: map[string]config.Device{
				"kitchen": {Name: "kitchen", Host: "http://10.0.0.1"},
				"office":  {Name: "office", Host: "http://10.0.0.2"},
			},
			defaultName:    "kitchen",
			json:           true,
			wantBanner:     false,
			wantSource:     config.SourceDefault,
			wantDeviceName: "kitchen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useTempDeviceConfigDir(t)
			cfg := &config.Config{Devices: tt.devices, Default: tt.defaultName}
			if err := cfg.Save(); err != nil {
				t.Fatalf("Save() error = %v", err)
			}

			origHostFlag, origDeviceFlag, origJSON := hostFlag, deviceFlag, jsonFlag
			origHost, origSource, origDeviceName := resolvedHost, resolvedSource, resolvedDeviceName
			t.Cleanup(func() {
				hostFlag, deviceFlag, jsonFlag = origHostFlag, origDeviceFlag, origJSON
				resolvedHost, resolvedSource, resolvedDeviceName = origHost, origSource, origDeviceName
			})
			hostFlag, deviceFlag, jsonFlag = "", "", tt.json

			var preErr error
			stdout := captureStdout(t, func() {
				preErr = rootCmd.PersistentPreRunE(statusCmd, nil)
			})
			if preErr != nil {
				t.Fatalf("PersistentPreRunE() = %v", preErr)
			}

			gotBanner := strings.Contains(stdout, "device: ")
			if gotBanner != tt.wantBanner {
				t.Errorf("banner printed = %v (stdout %q); want %v", gotBanner, stdout, tt.wantBanner)
			}
			if resolvedSource != tt.wantSource {
				t.Errorf("resolvedSource = %v; want %v", resolvedSource, tt.wantSource)
			}
			if resolvedDeviceName != tt.wantDeviceName {
				t.Errorf("resolvedDeviceName = %q; want %q", resolvedDeviceName, tt.wantDeviceName)
			}
		})
	}
}

func TestPersistentPreRunEBannerAbsentForDeviceIndependentCommand(t *testing.T) {
	useTempDeviceConfigDir(t)
	cfg := &config.Config{Devices: map[string]config.Device{
		"kitchen": {Name: "kitchen", Host: "http://10.0.0.1"},
		"office":  {Name: "office", Host: "http://10.0.0.2"},
	}, Default: "kitchen"}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	origHostFlag, origDeviceFlag := hostFlag, deviceFlag
	t.Cleanup(func() { hostFlag, deviceFlag = origHostFlag, origDeviceFlag })
	hostFlag, deviceFlag = "", ""

	var preErr error
	stdout := captureStdout(t, func() {
		preErr = rootCmd.PersistentPreRunE(deviceListCmd, nil)
	})
	if preErr != nil {
		t.Fatalf("PersistentPreRunE(deviceListCmd) = %v", preErr)
	}
	if strings.Contains(stdout, "device: ") {
		t.Errorf("stdout = %q; want no banner for a device-independent command", stdout)
	}
}

func TestPersistentPreRunEResolvesHostForTzSet(t *testing.T) {
	useTempDeviceConfigDir(t)

	origHostFlag, origDeviceFlag, origJSON := hostFlag, deviceFlag, jsonFlag
	origHost, origSource, origDeviceName := resolvedHost, resolvedSource, resolvedDeviceName
	t.Cleanup(func() {
		hostFlag, deviceFlag, jsonFlag = origHostFlag, origDeviceFlag, origJSON
		resolvedHost, resolvedSource, resolvedDeviceName = origHost, origSource, origDeviceName
	})
	hostFlag, deviceFlag, jsonFlag = "http://1.2.3.4", "", false
	resolvedHost = ""

	if err := rootCmd.PersistentPreRunE(tzSetCmd, nil); err != nil {
		t.Fatalf("PersistentPreRunE(tzSetCmd) = %v", err)
	}
	if resolvedHost != "http://1.2.3.4" {
		t.Errorf("resolvedHost = %q; want %q, since tz set targets a device like any other device-needing command", resolvedHost, "http://1.2.3.4")
	}
}

func TestCommandTargetsDeviceExcludesReservedCobraNames(t *testing.T) {
	for _, name := range []string{"help", "completion", "__complete", "__completeNoDesc"} {
		t.Run(name, func(t *testing.T) {
			c := &cobra.Command{Use: name}
			if commandTargetsDevice(c) {
				t.Errorf("commandTargetsDevice(%s) = true; want false, since it is a reserved cobra command name", name)
			}
		})
	}
}

func TestBuiltinCommandsSucceedWithZeroDevices(t *testing.T) {
	useTempDeviceConfigDir(t)

	tests := [][]string{
		{"help", "device"},
		{"completion", "bash"},
		{"__complete", ""},
	}
	for _, args := range tests {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			var out bytes.Buffer
			rootCmd.SetOut(&out)
			rootCmd.SetErr(&out)
			rootCmd.SetArgs(args)
			t.Cleanup(func() { rootCmd.SetArgs(nil) })

			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("rootCmd.Execute(%v) = %v; want nil with zero devices configured", args, err)
			}
		})
	}
}
