package routes

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/duncanleo/plex-dvr-hls/config"
	"github.com/gin-gonic/gin"
)

func TestDiscoverDeviceIDStability(t *testing.T) {
	// Set up a test device ID
	testDeviceID := "30480554"
	config.Cfg.DeviceID = &testDeviceID
	config.Cfg.Name = "Test Tuner"

	// Initialize empty channels to avoid nil pointer
	config.Channels = []config.Channel{}

	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/discover.json", Discover)

	// Make multiple requests
	deviceIDs := make([]string, 3)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/discover.json", nil)
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Fatalf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}

		var dvr DVR
		err := json.Unmarshal(w.Body.Bytes(), &dvr)
		if err != nil {
			t.Fatalf("Request %d: Failed to parse JSON response: %v", i+1, err)
		}

		deviceIDs[i] = dvr.DeviceID
	}

	// Verify all device IDs are the same
	for i := 1; i < len(deviceIDs); i++ {
		if deviceIDs[i] != deviceIDs[0] {
			t.Errorf("Device ID changed between requests: first='%s', request %d='%s'",
				deviceIDs[0], i+1, deviceIDs[i])
		}
	}

	// Verify it matches our test device ID
	if deviceIDs[0] != testDeviceID {
		t.Errorf("Expected device ID '%s', got '%s'", testDeviceID, deviceIDs[0])
	}
}

func TestDiscoverResponse(t *testing.T) {
	// Set up test config
	testDeviceID := "12345678"
	testName := "Amazing Tuner"
	config.Cfg.DeviceID = &testDeviceID
	config.Cfg.Name = testName
	config.Channels = []config.Channel{}

	// Create a test router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/discover.json", Discover)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/discover.json", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Expected status 200, got %d", w.Code)
	}

	var dvr DVR
	err := json.Unmarshal(w.Body.Bytes(), &dvr)
	if err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify response fields
	if dvr.DeviceID != testDeviceID {
		t.Errorf("Expected DeviceID '%s', got '%s'", testDeviceID, dvr.DeviceID)
	}

	if dvr.FriendlyName != testName {
		t.Errorf("Expected FriendlyName '%s', got '%s'", testName, dvr.FriendlyName)
	}

	if dvr.ModelNumber != "HDTC-2US" {
		t.Errorf("Expected ModelNumber 'HDTC-2US', got '%s'", dvr.ModelNumber)
	}

	if dvr.Manufacturer != "Silicondust" {
		t.Errorf("Expected Manufacturer 'Silicondust', got '%s'", dvr.Manufacturer)
	}

	if dvr.FirmwareName != "hdhomeruntc_atsc" {
		t.Errorf("Expected FirmwareName 'hdhomeruntc_atsc', got '%s'", dvr.FirmwareName)
	}

	if dvr.FirmwareVersion != "20150826" {
		t.Errorf("Expected FirmwareVersion '20150826', got '%s'", dvr.FirmwareVersion)
	}

	if dvr.DeviceAuth != "test1234" {
		t.Errorf("Expected DeviceAuth 'test1234', got '%s'", dvr.DeviceAuth)
	}
}

func TestDiscoverTunerCount(t *testing.T) {
	testDeviceID := "12345678"
	config.Cfg.DeviceID = &testDeviceID
	config.Cfg.Name = "Test"

	tests := []struct {
		name              string
		channels          int
		configTunerCount  *int
		expectedTunerCount int
	}{
		{
			name:              "Default tuner count (3x channels)",
			channels:          5,
			configTunerCount:  nil,
			expectedTunerCount: 15,
		},
		{
			name:              "Custom tuner count",
			channels:          5,
			configTunerCount:  intPtr(8),
			expectedTunerCount: 8,
		},
		{
			name:              "No channels with custom tuner count",
			channels:          0,
			configTunerCount:  intPtr(3),
			expectedTunerCount: 3,
		},
	}

	gin.SetMode(gin.TestMode)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up channels
			config.Channels = make([]config.Channel, tt.channels)
			config.Cfg.TunerCount = tt.configTunerCount

			router := gin.New()
			router.GET("/discover.json", Discover)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/discover.json", nil)
			router.ServeHTTP(w, req)

			var dvr DVR
			json.Unmarshal(w.Body.Bytes(), &dvr)

			if dvr.TunerCount != tt.expectedTunerCount {
				t.Errorf("Expected TunerCount %d, got %d", tt.expectedTunerCount, dvr.TunerCount)
			}
		})
	}
}

func intPtr(i int) *int {
	return &i
}
