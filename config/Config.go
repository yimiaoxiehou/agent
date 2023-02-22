package config

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/docker/docker/api/types"
	"gopkg.in/yaml.v3"
	"log"
	"os"
)

type Config struct {
	Services map[string]Service
}

type Service struct {
	Name                string `yaml:"name"`
	CN                  string `yaml:"cn"`
	HealthCheckMode     string `yaml:"health-check-mode"`
	HealthCheckEndpoint string `yaml:"health-check-endpoint"`
	HealthCheckCmd      string `yaml:"health-check-cmd"`
	NacosNamespace      string `yaml:"nacos-namespace"`
	NacosUsername       string `yaml:"nacos-username"`
	NacosPassword       string `yaml:"nacos-password"`
}

func (s Service) HealthCheck(container types.ContainerJSON) {
	switch s.HealthCheckMode {
	case "http":
		break
	case "docker-command":

	}
}

func LoadConfig() *Config {
	cfgPath, err := ParseFlags()
	if err != nil {
		log.Fatal(err)
	}
	cfg, err := NewConfig(cfgPath, "")
	if err != nil {
		log.Fatal(err)
	}
	return cfg
}

// NewConfig returns a new decoded Config struct
func NewConfig(configPath string, prefix string) (*Config, error) {
	// Create config structure
	config := &Config{}

	// Open config file
	file, err := os.Open(configPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// Init new YAML decode
	d := yaml.NewDecoder(file)

	if prefix == "" {
		if err := d.Decode(&config); err != nil {
			return nil, err
		}
		return config, nil
	}

	tree := make(map[string]interface{})
	// Start YAML decoding from file
	if err := d.Decode(&tree); err != nil {
		return nil, err
	}

	if tree[prefix] == nil {
		return nil, err
	}
	buff := &bytes.Buffer{}
	e := yaml.NewEncoder(buff)
	if e.Encode(tree[prefix]) != nil {
		return nil, err
	}

	d = yaml.NewDecoder(buff)
	if d.Decode(&config) != nil {
		return nil, err
	}
	return config, err
}

// ValidateConfigPath just makes sure, that the path provided is a file,
// that can be read
func ValidateConfigPath(path string) error {
	s, err := os.Stat(path)
	if err != nil {
		return err
	}
	if s.IsDir() {
		return fmt.Errorf("'%s' is a directory, not a normal file", path)
	}
	return nil
}

// ParseFlags will create and parse the CLI flags
// and return the path to be used elsewhere
func ParseFlags() (string, error) {
	// String that contains the configured configuration path
	var configPath string

	// Set up a CLI flag called "-config" to allow users
	// to supply the configuration file
	flag.StringVar(&configPath, "config", "./config.yml", "path to config file")

	// Actually parse the flags
	flag.Parse()

	// Validate the path first
	if err := ValidateConfigPath(configPath); err != nil {
		return "", err
	}

	// Return the configuration path
	return configPath, nil
}
