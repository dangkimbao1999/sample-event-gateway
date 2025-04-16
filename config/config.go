package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the services
type Config struct {
	Gateway GatewayConfig `mapstructure:"gateway"`
	Node    NodeConfig    `mapstructure:"node"`
	Consul  ConsulConfig  `mapstructure:"consul"`
	Log     LogConfig     `mapstructure:"log"`
}

// GatewayConfig holds gateway service configuration
type GatewayConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// NodeConfig holds node service configuration
type NodeConfig struct {
	ID          string            `mapstructure:"id"`
	Port        int               `mapstructure:"port"`
	HealthCheck HealthCheckConfig `mapstructure:"health_check"`
	Chains      []string          `mapstructure:"chains"` // List of blockchain chains this node handles
}

// HealthCheckConfig holds health check configuration
type HealthCheckConfig struct {
	Path     string `mapstructure:"path"`
	Port     int    `mapstructure:"port"`
	Interval string `mapstructure:"interval"`
	Timeout  string `mapstructure:"timeout"`
}

// ConsulConfig holds Consul configuration
type ConsulConfig struct {
	Host string `mapstructure:"host"`
	Port int    `mapstructure:"port"`
}

// LogConfig holds logging configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// LoadConfig loads configuration from file and environment variables
func LoadConfig(configPath string) (*Config, error) {
	v := viper.New()

	// Set default values
	setDefaults(v)

	// Read config file if provided
	if configPath != "" {
		v.SetConfigFile(configPath)
		log.Printf("Loading configuration from file: %s", configPath)
	} else {
		// Look for config in the current directory
		v.AddConfigPath(".")
		v.AddConfigPath("./config")
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		log.Printf("Looking for config file in . and ./config directories")
	}

	// Read environment variables
	v.SetEnvPrefix("EVENT_CATCHER")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	// Read the config file
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
		log.Printf("No config file found, using defaults and environment variables")
	} else {
		log.Printf("Using config file: %s", v.ConfigFileUsed())
	}

	var config Config
	if err := v.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Log the loaded configuration
	log.Printf("Loaded configuration: %+v", config)

	return &config, nil
}

// setDefaults sets default values for the configuration
func setDefaults(v *viper.Viper) {
	// Gateway defaults
	v.SetDefault("gateway.host", "0.0.0.0")
	v.SetDefault("gateway.port", 50051)

	// Node defaults
	v.SetDefault("node.id", "node1")
	v.SetDefault("node.port", 50052)
	v.SetDefault("node.health_check.path", "/health")
	v.SetDefault("node.health_check.port", 50053)
	v.SetDefault("node.health_check.interval", "10s")
	v.SetDefault("node.health_check.timeout", "5s")
	v.SetDefault("node.chains", []string{"ethereum", "bitcoin"})

	// Consul defaults
	v.SetDefault("consul.host", "localhost")
	v.SetDefault("consul.port", 8500)

	// Log defaults
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "text")
}

// GetGatewayAddr returns the full gateway address
func (c *Config) GetGatewayAddr() string {
	return fmt.Sprintf("%s:%d", c.Gateway.Host, c.Gateway.Port)
}

// GetConsulAddr returns the full Consul address
func (c *Config) GetConsulAddr() string {
	return fmt.Sprintf("%s:%d", c.Consul.Host, c.Consul.Port)
}

// GetNodeAddr returns the full node address
func (c *Config) GetNodeAddr() string {
	return fmt.Sprintf(":%d", c.Node.Port)
}

// GetNodeHealthAddr returns the full node health check address
func (c *Config) GetNodeHealthAddr() string {
	return fmt.Sprintf(":%d", c.Node.HealthCheck.Port)
}
