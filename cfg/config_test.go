package cfg

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"gopkg.in/yaml.v2"
)

const testCurrentFormatVersion = 2

// Helper function to create a temporary config file
func createTempConfigFile(t *testing.T, content string) string {
	t.Helper()
	tempDir := t.TempDir()
	tmpFile, err := os.Create(filepath.Join(tempDir, ".gg"))
	if err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp config file: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("Failed to close temp config file: %v", err)
	}
	// Override user home dir to point to our temp dir for the duration of the test
	// so LoadOrCreateConfig and Save work with our temp file.
	originalHomeDir, present := os.LookupEnv("HOME")
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("Failed to set HOME env var: %v", err)
	}
	if present {
		t.Cleanup(func() {
			if err := os.Setenv("HOME", originalHomeDir); err != nil {
				t.Fatalf("Failed to restore HOME env var: %v", err)
			}
		})
	} else {
		t.Cleanup(func() {
			if err := os.Unsetenv("HOME"); err != nil {
				t.Fatalf("Failed to unset HOME env var: %v", err)
			}
		})
	}

	return tmpFile.Name()
}

// Helper function to read and unmarshal a config file
func loadConfigFromFile(t *testing.T, path string) *Config {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read config file %s: %v", path, err)
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("Failed to unmarshal config from %s: %v", path, err)
	}
	return &cfg
}

func TestLoadExistingConfigWithoutVersion(t *testing.T) {
	configContent := `
jira_user: testuser
jira_url: https://test.jira.com
jira_token: sometoken
jira_project: TEST
tasks: {}
`
	configFile := createTempConfigFile(t, configContent)

	mockAskFn := func() *Config {
		t.Error("AskForConfig should not be called when a config file exists")
		return &Config{FormatVersion: 99, lock: &sync.Mutex{}} // Should not happen
	}

	loadedCfg, err := LoadOrCreateConfig(mockAskFn)
	if err != nil {
		t.Fatalf("LoadOrCreateConfig failed: %v", err)
	}

	if loadedCfg.FormatVersion != 1 {
		t.Errorf("Expected FormatVersion to be 1 for old config, got %d", loadedCfg.FormatVersion)
	}

	// Ensure lock is initialized
	if loadedCfg.lock == nil {
		loadedCfg.lock = &sync.Mutex{}
	}
	
	err = loadedCfg.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	savedCfg := loadConfigFromFile(t, configFile)
	if savedCfg.FormatVersion != testCurrentFormatVersion {
		t.Errorf("Expected FormatVersion in saved file to be %d, got %d", testCurrentFormatVersion, savedCfg.FormatVersion)
	}
}

func TestCreateNewConfig(t *testing.T) {
	tempDir := t.TempDir()
	// Override user home dir to point to our temp dir for the duration of the test
	originalHomeDir, present := os.LookupEnv("HOME")
	if err := os.Setenv("HOME", tempDir); err != nil {
		t.Fatalf("Failed to set HOME env var: %v", err)
	}
	if present {
		t.Cleanup(func() {
			if err := os.Setenv("HOME", originalHomeDir); err != nil {
				t.Fatalf("Failed to restore HOME env var: %v", err)
			}
		})
	} else {
		t.Cleanup(func() {
			if err := os.Unsetenv("HOME"); err != nil {
				t.Fatalf("Failed to unset HOME env var: %v", err)
			}
		})
	}

	// Ensure no config file exists initially
	configFilePath := filepath.Join(tempDir, ".gg")
	if _, err := os.Stat(configFilePath); !os.IsNotExist(err) {
		t.Fatalf("Config file %s should not exist at the start of this test, but it does.", configFilePath)
	}

	var askFnCalled bool
	mockAskFn := func() *Config {
		askFnCalled = true
		// Create a new config like the main NewConfg would, including setting version
		return &Config{
			JiraUser:      "newuser",
			JiraURL:       "https://new.jira.com",
			JiraToken:     "newtoken",
			JiraProject:   "NEW",
			FormatVersion: testCurrentFormatVersion, // NewConfg now sets this
			Tasks:         map[string]Task{},
			lock:          &sync.Mutex{},
		}
	}

	createdCfg, err := LoadOrCreateConfig(mockAskFn)
	if err != nil {
		t.Fatalf("LoadOrCreateConfig failed: %v", err)
	}

	if !askFnCalled {
		t.Error("AskForConfig was not called when creating a new config")
	}

	if createdCfg.FormatVersion != testCurrentFormatVersion {
		t.Errorf("Expected FormatVersion to be %d for new config, got %d", testCurrentFormatVersion, createdCfg.FormatVersion)
	}
	
	// Ensure lock is initialized
	if createdCfg.lock == nil {
		createdCfg.lock = &sync.Mutex{}
	}

	err = createdCfg.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	savedCfg := loadConfigFromFile(t, configFilePath)
	if savedCfg.FormatVersion != testCurrentFormatVersion {
		t.Errorf("Expected FormatVersion in saved file to be %d, got %d", testCurrentFormatVersion, savedCfg.FormatVersion)
	}
	if savedCfg.JiraUser != "newuser" { // Check if other data from mockAskFn was saved
		t.Errorf("Expected JiraUser to be 'newuser', got '%s'", savedCfg.JiraUser)
	}
}

func TestLoadConfigWithCurrentVersion(t *testing.T) {
	configContent := fmt.Sprintf(`
jira_user: testuser
jira_url: https://test.jira.com
jira_token: sometoken
jira_project: TEST
format_version: %d
tasks: {}
`, testCurrentFormatVersion)
	configFile := createTempConfigFile(t, configContent)

	mockAskFn := func() *Config {
		t.Error("AskForConfig should not be called when a config file exists")
		return &Config{FormatVersion: 99, lock: &sync.Mutex{}} // Should not happen
	}

	loadedCfg, err := LoadOrCreateConfig(mockAskFn)
	if err != nil {
		t.Fatalf("LoadOrCreateConfig failed: %v", err)
	}

	if loadedCfg.FormatVersion != testCurrentFormatVersion {
		t.Errorf("Expected FormatVersion to be %d, got %d", testCurrentFormatVersion, loadedCfg.FormatVersion)
	}

	// Ensure lock is initialized
	if loadedCfg.lock == nil {
		loadedCfg.lock = &sync.Mutex{}
	}
	
	err = loadedCfg.Save()
	if err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	savedCfg := loadConfigFromFile(t, configFile)
	if savedCfg.FormatVersion != testCurrentFormatVersion {
		t.Errorf("Expected FormatVersion in saved file to be %d, got %d", testCurrentFormatVersion, savedCfg.FormatVersion)
	}
}
