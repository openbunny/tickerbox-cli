package cmd

import (
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/openbunny/tickerbox-cli/internal/config"
)

func useTempDeviceConfigDir(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("AppData", dir)
}

func TestSubnetHosts(t *testing.T) {
	tests := []struct {
		name string
		ip   net.IP
		want int
	}{
		{name: "IPv4 excludes self", ip: net.ParseIP("192.168.10.89"), want: subnetLastHost - subnetFirstHost},
		{name: "IPv4 self at edge", ip: net.ParseIP("10.0.0.254"), want: subnetLastHost - subnetFirstHost},
		{name: "IPv6 unsupported", ip: net.ParseIP("2001:db8::1"), want: 0},
		{name: "nil unsupported", ip: nil, want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := subnetHosts(tt.ip)
			if len(got) != tt.want {
				t.Fatalf("subnetHosts(%v) len = %d, want %d", tt.ip, len(got), tt.want)
			}
			if tt.want == 0 {
				return
			}

			self := tt.ip.String()
			seen := make(map[string]bool, len(got))
			for _, h := range got {
				if h == self {
					t.Errorf("subnetHosts(%v) includes self %s", tt.ip, self)
				}
				if seen[h] {
					t.Errorf("subnetHosts(%v) duplicates %s", tt.ip, h)
				}
				seen[h] = true
				if strings.HasSuffix(h, ".0") || strings.HasSuffix(h, ".255") {
					t.Errorf("subnetHosts(%v) includes network/broadcast address %s", tt.ip, h)
				}
			}

			prefix := self[:strings.LastIndex(self, ".")+1]
			if !seen[prefix+"1"] {
				t.Errorf("subnetHosts(%v) missing %s1", tt.ip, prefix)
			}
			if self != prefix+"254" && !seen[prefix+"254"] {
				t.Errorf("subnetHosts(%v) missing %s254", tt.ip, prefix)
			}
		})
	}
}

func featuresServer(t *testing.T, status int, body string) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rest/features" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return strings.TrimPrefix(srv.URL, "http://")
}

func TestProbeTickerbox(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   bool
	}{
		{name: "tickerbox true", status: http.StatusOK, body: `{"tickerbox":true}`, want: true},
		{name: "tickerbox false", status: http.StatusOK, body: `{"tickerbox":false}`, want: false},
		{name: "tickerbox absent", status: http.StatusOK, body: `{"ntp":true}`, want: false},
		{name: "non-2xx status", status: http.StatusNotFound, body: `{"tickerbox":true}`, want: false},
		{name: "malformed json", status: http.StatusOK, body: `not json`, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			host := featuresServer(t, tt.status, tt.body)
			if got := probeTickerbox(host, time.Second); got != tt.want {
				t.Errorf("probeTickerbox() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProbeTickerboxUnreachable(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	host := strings.TrimPrefix(srv.URL, "http://")
	srv.Close()

	if probeTickerbox(host, 200*time.Millisecond) {
		t.Error("probeTickerbox() on closed server = true, want false")
	}
}

func TestScanSubnet(t *testing.T) {
	tickerbox1 := featuresServer(t, http.StatusOK, `{"tickerbox":true}`)
	tickerbox2 := featuresServer(t, http.StatusOK, `{"tickerbox":true}`)
	notTickerbox := featuresServer(t, http.StatusOK, `{"tickerbox":false}`)

	closedSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	closedHost := strings.TrimPrefix(closedSrv.URL, "http://")
	closedSrv.Close()

	hosts := []string{tickerbox1, tickerbox2, notTickerbox, closedHost}
	got := scanSubnet(hosts, 500*time.Millisecond, 4)

	gotHosts := make(map[string]bool, len(got))
	for _, d := range got {
		gotHosts[d.Host] = true
	}

	if len(got) != 2 {
		t.Fatalf("scanSubnet() found %d hosts, want 2: %v", len(got), got)
	}
	if !gotHosts[tickerbox1] || !gotHosts[tickerbox2] {
		t.Errorf("scanSubnet() = %v, want both %s and %s", got, tickerbox1, tickerbox2)
	}
	if gotHosts[notTickerbox] || gotHosts[closedHost] {
		t.Errorf("scanSubnet() incorrectly included a non-TickerBox host: %v", got)
	}
}

func TestPingDevice(t *testing.T) {
	t.Run("reachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{}`))
		}))
		defer srv.Close()

		r := pingDevice(config.Device{Name: "kitchen", Host: srv.URL})
		if !r.Reachable {
			t.Error("Reachable = false, want true")
		}
		if r.RTTMillis < 0 {
			t.Errorf("RTTMillis = %d, want >= 0", r.RTTMillis)
		}
		if r.Name != "kitchen" || r.Host != srv.URL {
			t.Errorf("got Name=%q Host=%q, want Name=%q Host=%q", r.Name, r.Host, "kitchen", srv.URL)
		}
	})

	t.Run("unreachable", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
		url := srv.URL
		srv.Close()

		r := pingDevice(config.Device{Name: "office", Host: url})
		if r.Reachable {
			t.Error("Reachable = true, want false")
		}
	})

	t.Run("server error", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer srv.Close()

		r := pingDevice(config.Device{Name: "garage", Host: srv.URL})
		if r.Reachable {
			t.Error("Reachable = true, want false")
		}
	})
}

func TestRunDeviceAddRmUseList(t *testing.T) {
	useTempDeviceConfigDir(t)

	if err := runDeviceAdd(nil, []string{"kitchen", "10.0.0.5"}); err != nil {
		t.Fatalf("runDeviceAdd() error = %v", err)
	}
	if err := runDeviceAdd(nil, []string{"office", "10.0.0.6"}); err != nil {
		t.Fatalf("runDeviceAdd() error = %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if len(cfg.Devices) != 2 {
		t.Fatalf("Devices = %v, want 2 entries", cfg.Devices)
	}

	if err := runDeviceUse(nil, []string{"office"}); err != nil {
		t.Fatalf("runDeviceUse() error = %v", err)
	}
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if cfg.Default != "office" {
		t.Errorf("Default = %q, want office", cfg.Default)
	}

	if err := runDeviceUse(nil, []string{"unknown"}); err == nil {
		t.Error("runDeviceUse(unknown) error = nil, want error")
	}

	if err := runDeviceList(nil, nil); err != nil {
		t.Fatalf("runDeviceList() error = %v", err)
	}

	if err := runDeviceRm(nil, []string{"office"}); err != nil {
		t.Fatalf("runDeviceRm() error = %v", err)
	}
	cfg, err = config.Load()
	if err != nil {
		t.Fatalf("config.Load() error = %v", err)
	}
	if _, ok := cfg.Devices["office"]; ok {
		t.Error("Devices still contains removed device office")
	}
	if cfg.Default != "" {
		t.Errorf("Default after removing default device = %q, want empty", cfg.Default)
	}

	if err := runDeviceRm(nil, []string{"office"}); err == nil {
		t.Error("runDeviceRm(office) twice: error = nil, want error")
	}
}

func TestRunDevicePing(t *testing.T) {
	useTempDeviceConfigDir(t)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	if err := runDeviceAdd(nil, []string{"kitchen", srv.URL}); err != nil {
		t.Fatalf("runDeviceAdd() error = %v", err)
	}

	t.Run("no name and no --all errors", func(t *testing.T) {
		devicePingAll = false
		if err := runDevicePing(nil, nil); err == nil {
			t.Error("runDevicePing() error = nil, want error")
		}
	})

	t.Run("unknown device errors", func(t *testing.T) {
		devicePingAll = false
		if err := runDevicePing(nil, []string{"missing"}); err == nil {
			t.Error("runDevicePing(missing) error = nil, want error")
		}
	})

	t.Run("--all with a name errors", func(t *testing.T) {
		devicePingAll = true
		defer func() { devicePingAll = false }()
		if err := runDevicePing(nil, []string{"kitchen"}); err == nil {
			t.Error("runDevicePing(--all, kitchen) error = nil, want error")
		}
	})

	t.Run("named device pings successfully", func(t *testing.T) {
		devicePingAll = false
		if err := runDevicePing(nil, []string{"kitchen"}); err != nil {
			t.Fatalf("runDevicePing(kitchen) error = %v", err)
		}
	})

	t.Run("--all pings every configured device", func(t *testing.T) {
		devicePingAll = true
		defer func() { devicePingAll = false }()
		if err := runDevicePing(nil, nil); err != nil {
			t.Fatalf("runDevicePing(--all) error = %v", err)
		}
	})
}
