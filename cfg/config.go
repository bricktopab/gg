package cfg

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"
)

const currentFormatVersion = 2

type Task struct {
	IssueID     string
	Title       string
	Description string
	Type        string
}

type Config struct {
	JiraUser    string `yaml:"jira_user"`
	JiraURL     string `yaml:"jira_url"`
	JiraToken   string `yaml:"jira_token"`
	JiraProject string `yaml:"jira_project"`

	FormatVersion int `yaml:"format_version"`

	Tasks map[string]Task `yaml:"tasks"`

	lock *sync.Mutex
}

type JiraConfig struct {
	JiraUser      string  `yaml:"jira_user"`
	JiraURL       string  `yaml:"jira_url"`
	JiraToken     string  `yaml:"jira_token"`
	JiraProject   string  `yaml:"jira_project"`
	JiraAccountID *string `yaml:"jira_account_id"`
}

type AskForConfig func() *Config

func NewConfg() Config {
	return Config{
		// FormatVersion will be set by LoadOrCreateConfig or Save
		Tasks: map[string]Task{},
		lock:  &sync.Mutex{},
	}
}

func LoadOrCreateConfig(askForConfig AskForConfig) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".gg")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			newCfg := askForConfig()
			// Initialize fields that NewConfg would, and set current format version
			newCfg.FormatVersion = currentFormatVersion
			if newCfg.lock == nil {
				newCfg.lock = &sync.Mutex{}
			}
			if newCfg.Tasks == nil {
				newCfg.Tasks = make(map[string]Task)
			}
			// Save will also ensure FormatVersion is currentFormatVersion
			return newCfg, newCfg.Save()
		}
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var config Config
	// Initialize non-yaml fields manually before decode
	config.lock = &sync.Mutex{}
	// Tasks map will be populated by decoder if present in YAML, otherwise needs init
	// config.Tasks = make(map[string]Task) // Let decoder handle this or init after.

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		// If decoding fails, it might be an empty or malformed file.
		// For this specific case, we might want to treat it as "not exist"
		// and create a new one, but the current logic is to error out.
		// For now, let's stick to the error.
		return nil, fmt.Errorf("failed to decode config file: %w", err)
	}

	// After successful decode, handle versioning and ensure essential maps/locks
	if config.FormatVersion == 0 { // If FormatVersion was missing in the file (or explicitly 0)
		config.FormatVersion = 1 // This is an old config (version 1)
	}

	// Ensure Tasks map is initialized if it was nil (e.g. empty or old config file)
	if config.Tasks == nil {
		config.Tasks = make(map[string]Task)
	}
	// Lock should have been initialized above.

	return &config, nil
}

func (c *Config) Save() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.FormatVersion = currentFormatVersion

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".gg")
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("failed to create config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("failed to encode config file: %w", err)
	}

	return nil
}

func (c *Config) AddTask(task *Task) {
	c.Tasks[task.IssueID] = *task
	_ = c.Save()
}

func (c *Config) GetTask(id string) *Task {
	task := c.Tasks[id]
	return &task
}
