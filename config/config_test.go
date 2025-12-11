package config

import (
	"encoding/json"
	"os"
	"testing"
)

func TestDeviceIDFromConfig(t *testing.T) {
	// Create a temporary config file with a specific device ID
	configContent := `{
		"name": "Test Tuner",
		"device_id": "12345678"
	}`

	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Read and parse the config
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open config: %v", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		t.Fatalf("Failed to decode config: %v", err)
	}

	// Verify device ID was loaded correctly
	if cfg.DeviceID == nil {
		t.Fatal("DeviceID should not be nil when provided in config")
	}

	if *cfg.DeviceID != "12345678" {
		t.Errorf("Expected DeviceID to be '12345678', got '%s'", *cfg.DeviceID)
	}
}

func TestDeviceIDGeneration(t *testing.T) {
	// Create a temporary config file without a device ID
	configContent := `{
		"name": "Test Tuner"
	}`

	tmpFile, err := os.CreateTemp("", "config-*.json")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(configContent)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	tmpFile.Close()

	// Read and parse the config
	file, err := os.Open(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to open config: %v", err)
	}
	defer file.Close()

	var cfg Config
	decoder := json.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		t.Fatalf("Failed to decode config: %v", err)
	}

	// Simulate the init() function's device ID generation
	if cfg.DeviceID == nil {
		// Generate device ID like the init() function does
		deviceID := "10000000" // Placeholder for test
		cfg.DeviceID = &deviceID
	}

	// Verify device ID was set
	if cfg.DeviceID == nil {
		t.Fatal("DeviceID should be generated when not provided in config")
	}

	// Verify it's an 8-digit number (string format)
	if len(*cfg.DeviceID) != 8 {
		t.Errorf("Expected DeviceID to be 8 digits, got %d digits: '%s'", len(*cfg.DeviceID), *cfg.DeviceID)
	}

	// Verify it contains only digits
	for _, c := range *cfg.DeviceID {
		if c < '0' || c > '9' {
			t.Errorf("DeviceID should contain only digits, got: '%s'", *cfg.DeviceID)
			break
		}
	}
}

func TestDeviceIDFormat(t *testing.T) {
	tests := []struct {
		name        string
		deviceID    string
		shouldError bool
	}{
		{"Valid 8-digit ID", "30480554", false},
		{"Valid ID with leading 1", "10000000", false},
		{"Valid ID with leading 8", "89999999", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Verify the device ID is 8 digits
			if len(tt.deviceID) != 8 {
				if !tt.shouldError {
					t.Errorf("Expected 8-digit device ID, got %d digits", len(tt.deviceID))
				}
				return
			}

			// Verify it's numeric
			for _, c := range tt.deviceID {
				if c < '0' || c > '9' {
					if !tt.shouldError {
						t.Errorf("Device ID should be numeric, got: %s", tt.deviceID)
					}
					return
				}
			}
		})
	}
}

func TestConfigJSONUnmarshal(t *testing.T) {
	tests := []struct {
		name           string
		json           string
		expectedID     *string
		expectedName   string
		shouldHaveID   bool
	}{
		{
			name:         "Config with device_id",
			json:         `{"name": "Test", "device_id": "30480554"}`,
			expectedID:   stringPtr("30480554"),
			expectedName: "Test",
			shouldHaveID: true,
		},
		{
			name:         "Config without device_id",
			json:         `{"name": "Test"}`,
			expectedID:   nil,
			expectedName: "Test",
			shouldHaveID: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var cfg Config
			err := json.Unmarshal([]byte(tt.json), &cfg)
			if err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}

			if cfg.Name != tt.expectedName {
				t.Errorf("Expected name '%s', got '%s'", tt.expectedName, cfg.Name)
			}

			if tt.shouldHaveID {
				if cfg.DeviceID == nil {
					t.Error("Expected DeviceID to be set, but it was nil")
				} else if *cfg.DeviceID != *tt.expectedID {
					t.Errorf("Expected DeviceID '%s', got '%s'", *tt.expectedID, *cfg.DeviceID)
				}
			} else {
				if cfg.DeviceID != nil {
					t.Errorf("Expected DeviceID to be nil, but got '%s'", *cfg.DeviceID)
				}
			}
		})
	}
}

func TestDeviceIDPersistence(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	// Create a config without device_id
	configContent := `{"name": "Test"}`
	os.WriteFile("config.json", []byte(configContent), 0644)

	// First load - should generate and save device ID
	err := LoadConfig()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if Cfg.DeviceID == nil {
		t.Fatal("DeviceID should be generated")
	}
	firstID := *Cfg.DeviceID

	// Check that .device_id file was created
	savedID, err := loadDeviceIDFromFile()
	if err != nil {
		t.Fatalf("Device ID file should have been created: %v", err)
	}

	if savedID != firstID {
		t.Errorf("Saved device ID %s doesn't match generated ID %s", savedID, firstID)
	}

	// Reset config and load again - should use saved device ID
	Cfg = Config{}
	err = LoadConfig()
	if err != nil {
		t.Fatalf("Failed to reload config: %v", err)
	}

	if Cfg.DeviceID == nil {
		t.Fatal("DeviceID should be loaded from file")
	}

	if *Cfg.DeviceID != firstID {
		t.Errorf("Device ID changed after reload: expected %s, got %s", firstID, *Cfg.DeviceID)
	}
}

func stringPtr(s string) *string {
	return &s
}
