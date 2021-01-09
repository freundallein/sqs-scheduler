package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

// AppConfig ...
type AppConfig struct {
	Storage struct {
		DSN string `yaml:"dsn"`
	}
	AWS struct {
		Region             string `yaml:"region"`
		CredentialsFile    string `yaml:"credentialsFile"`
		CredentialsProfile string `yaml:"credentialsProfile"`
	}
	Submitter struct {
		Queuesrc struct {
			Name    string `yaml:"name"`
			URL     string `yaml:"url"`
			Retries int    `yaml:"readRetries"`
		}
		Workers  int    `yaml:"workers"`
		LogLevel string `yaml:"loglevel"`
	}
	Scheduler struct {
		Queuedst struct {
			Name    string `yaml:"name"`
			URL     string `yaml:"url"`
			Retries int    `yaml:"readRetries"`
		}
		Workers  int    `yaml:"workers"`
		LogLevel string `yaml:"loglevel"`
	}
	Worker struct {
		Queuesrc struct {
			Name    string `yaml:"name"`
			URL     string `yaml:"url"`
			Retries int    `yaml:"readRetries"`
		}
		Queuedst struct {
			Name    string `yaml:"name"`
			URL     string `yaml:"url"`
			Retries int    `yaml:"readRetries"`
		}
		Workers  int    `yaml:"workers"`
		LogLevel string `yaml:"loglevel"`
	}
	Resulter struct {
		Queuesrc struct {
			Name    string `yaml:"name"`
			URL     string `yaml:"url"`
			Retries int    `yaml:"readRetries"`
		}
		Workers  int    `yaml:"workers"`
		LogLevel string `yaml:"loglevel"`
	}
	Supervisor struct {
		Workers         int    `yaml:"workers"`
		LogLevel        string `yaml:"loglevel"`
		StaleTimeout    int    `yaml:"staleTimeout"`
		RepairBatchSize int    `yaml:"repairBatchSize"`
		Expiration      int    `yaml:"expiration"`
	}
}

// Read ...
func Read() (*AppConfig, error) {
	filename := os.Getenv("CFG_PATH")
	cfg := &AppConfig{}
	buff, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal(buff, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}
