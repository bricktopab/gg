package cfg

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v2"
)

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

	Tasks map[string]Task `yaml:"tasks"`

	lock *sync.Mutex
}

type AskForConfig func() *Config

func NewConfg() Config {
	return Config{
		Tasks: map[string]Task{},
		lock:  &sync.Mutex{},
	}
}

func LoadOrCreateConfig(askForConfig AskForConfig) (*Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("Failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".gg")
	file, err := os.Open(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			config := askForConfig()
			return config, config.Save()
		}
		log.Fatalf("Failed to open config file: %v", err)
	}
	defer func() {
		_ = file.Close()
	}()

	var config Config = NewConfg()
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&config); err != nil {
		return nil, fmt.Errorf("Failed to decode config file: %w", err)
	}

	return &config, nil
}

func (c *Config) Save() error {
	c.lock.Lock()
	defer c.lock.Unlock()

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("Failed to get home directory: %w", err)
	}

	configPath := filepath.Join(homeDir, ".gg")
	file, err := os.Create(configPath)
	if err != nil {
		return fmt.Errorf("Failed to create config file: %w", err)
	}
	defer func() {
		_ = file.Close()
	}()

	encoder := yaml.NewEncoder(file)
	if err := encoder.Encode(c); err != nil {
		return fmt.Errorf("Failed to encode config file: %w", err)
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
