// SPDX-License-Identifier: MIT

//go:build ignore

package main

import (
	"encoding/json"
	"log"
	"net/http"
)

const (
	mockAddr = "127.0.0.1:8765"

	mockCPUFreqMHz         = 240
	mockMaxAllocHeapBytes  = 160 * 1024
	mockPsramSizeBytes     = 2 * 1024 * 1024
	mockFreePsramBytes     = 1800 * 1024
	mockFreeHeapBytes      = 180 * 1024
	mockSketchSizeBytes    = 900 * 1024
	mockFreeSketchBytes    = 3 * 1024 * 1024
	mockFlashChipSizeBytes = 4 * 1024 * 1024
	mockFlashChipSpeedHz   = 40 * 1000 * 1000
	mockFsTotalBytes       = 2 * 1024 * 1024
	mockFsUsedBytes        = 1 * 1024 * 1024

	mockWifiStatusConnected = 3
	mockWifiRSSI            = -52
	mockWifiChannel         = 6

	mockAPStatusInactive = 1
	mockAPStationCount   = 0
	mockNTPStatusActive  = 1
	mockNTPUptimeSeconds = 123456

	mockBrightness     = 80
	mockChangeInterval = 10
	mockAnimationSpeed = 5

	mockTickerCount = 3
)

type mockFeatures struct {
	Project        bool `json:"project"`
	Ntp            bool `json:"ntp"`
	Ota            bool `json:"ota"`
	UploadFirmware bool `json:"upload_firmware"`
	Tickerbox      bool `json:"tickerbox"`
	Shopify        bool `json:"shopify"`
	ExtraEtf       bool `json:"extra_etf"`
}

type mockSystemStatus struct {
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

type mockWifiStatus struct {
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

type mockAPStatus struct {
	Status     int    `json:"status"`
	IPAddress  string `json:"ip_address"`
	MacAddress string `json:"mac_address"`
	StationNum int    `json:"station_num"`
}

type mockNTPStatus struct {
	Status    int    `json:"status"`
	UTCTime   string `json:"utc_time"`
	LocalTime string `json:"local_time"`
	Server    string `json:"server"`
	Uptime    int64  `json:"uptime"`
}

type mockSettingsState struct {
	Brightness     int    `json:"brightness"`
	ChangeInterval int    `json:"changeInterval"`
	SleepEnabled   bool   `json:"sleepEnabled"`
	SleepStart     string `json:"sleepStart"`
	SleepEnd       string `json:"sleepEnd"`
}

type mockClockSetupState struct {
	Enabled          bool   `json:"enabled"`
	TwelveHourFormat bool   `json:"twelweHourFormat"`
	AnimationSpeed   int    `json:"animationSpeed"`
	TzLabel          string `json:"tz_label"`
}

type mockCoinSetupState struct {
	Size     int    `json:"size"`
	Types    string `json:"types"`
	Tickers  string `json:"tickers"`
	Times    string `json:"times"`
	Currency string `json:"currency"`
}

func serveJSON(v any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(v); err != nil {
			log.Printf("encode response for %s: %v", r.URL.Path, err)
		}
	}
}

func main() {
	mux := http.NewServeMux()

	mux.Handle("/rest/features", serveJSON(mockFeatures{
		Project: true, Ntp: true, Ota: true, UploadFirmware: true,
		Tickerbox: true, Shopify: false, ExtraEtf: true,
	}))

	mux.Handle("/rest/systemStatus", serveJSON(mockSystemStatus{
		EspPlatform:     "esp32",
		MaxAllocHeap:    mockMaxAllocHeapBytes,
		PsramSize:       mockPsramSizeBytes,
		FreePsram:       mockFreePsramBytes,
		CPUFreqMHz:      mockCPUFreqMHz,
		FreeHeap:        mockFreeHeapBytes,
		SketchSize:      mockSketchSizeBytes,
		FreeSketchSpace: mockFreeSketchBytes,
		SdkVersion:      "v5.1.2",
		FlashChipSize:   mockFlashChipSizeBytes,
		FlashChipSpeed:  mockFlashChipSpeedHz,
		FsTotal:         mockFsTotalBytes,
		FsUsed:          mockFsUsedBytes,
	}))

	mux.Handle("/rest/wifiStatus", serveJSON(mockWifiStatus{
		Status:     mockWifiStatusConnected,
		LocalIP:    "192.0.2.10",
		MacAddress: "02:00:00:00:00:01",
		RSSI:       mockWifiRSSI,
		SSID:       "Demo-WiFi",
		BSSID:      "02:00:00:00:00:ff",
		Channel:    mockWifiChannel,
		SubnetMask: "255.255.255.0",
		GatewayIP:  "192.0.2.1",
		DNSIP1:     "192.0.2.1",
	}))

	mux.Handle("/rest/apStatus", serveJSON(mockAPStatus{
		Status:     mockAPStatusInactive,
		IPAddress:  "192.0.2.1",
		MacAddress: "02:00:00:00:00:02",
		StationNum: mockAPStationCount,
	}))

	mux.Handle("/rest/ntpStatus", serveJSON(mockNTPStatus{
		Status:    mockNTPStatusActive,
		UTCTime:   "2026-09-30T06:00:00Z",
		LocalTime: "2026-09-30T08:00:00+02:00",
		Server:    "pool.ntp.org",
		Uptime:    mockNTPUptimeSeconds,
	}))

	mux.Handle("/rest/settingsState", serveJSON(mockSettingsState{
		Brightness:     mockBrightness,
		ChangeInterval: mockChangeInterval,
		SleepEnabled:   false,
		SleepStart:     "22:00",
		SleepEnd:       "07:00",
	}))

	mux.Handle("/rest/clockSetupState", serveJSON(mockClockSetupState{
		Enabled:          true,
		TwelveHourFormat: false,
		AnimationSpeed:   mockAnimationSpeed,
		TzLabel:          "Europe/Berlin",
	}))

	mux.Handle("/rest/coinSetupState", serveJSON(mockCoinSetupState{
		Size:     mockTickerCount,
		Types:    "crypto,crypto,stocks",
		Tickers:  "BTC,ETH,AAPL",
		Times:    "5min,5min,1min",
		Currency: "USD,USD,USD",
	}))

	log.Printf("tickerbox demo mock listening on http://%s", mockAddr)
	if err := http.ListenAndServe(mockAddr, mux); err != nil {
		log.Fatalf("mock server: %v", err)
	}
}
