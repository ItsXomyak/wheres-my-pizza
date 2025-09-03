package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all configuration for the restaurant system
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

// RabbitMQConfig holds RabbitMQ connection configuration
type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Load reads configuration from a YAML file
func Load(filename string) (*Config, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open config file: %w", err)
	}
	defer file.Close()

	config := &Config{}
	scanner := bufio.NewScanner(file)
	
	var currentSection string
	
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		
		// Check for section headers
		if strings.HasSuffix(line, ":") && !strings.Contains(line, " ") {
			currentSection = strings.TrimSuffix(line, ":")
			continue
		}
		
		// Parse key-value pairs
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			
			if err := config.setValue(currentSection, key, value); err != nil {
				return nil, fmt.Errorf("failed to set config value %s.%s: %w", currentSection, key, err)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}
	
	return config, nil
}

// setValue sets a configuration value based on section and key
func (c *Config) setValue(section, key, value string) error {
	switch section {
	case "database":
		return c.setDatabaseValue(key, value)
	case "rabbitmq":
		return c.setRabbitMQValue(key, value)
	default:
		return fmt.Errorf("unknown section: %s", section)
	}
}

// setDatabaseValue sets database configuration values
func (c *Config) setDatabaseValue(key, value string) error {
	switch key {
	case "host":
		c.Database.Host = value
	case "port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port value: %w", err)
		}
		c.Database.Port = port
	case "user":
		c.Database.User = value
	case "password":
		c.Database.Password = value
	case "database":
		c.Database.Database = value
	default:
		return fmt.Errorf("unknown database key: %s", key)
	}
	return nil
}

// setRabbitMQValue sets RabbitMQ configuration values
func (c *Config) setRabbitMQValue(key, value string) error {
	switch key {
	case "host":
		c.RabbitMQ.Host = value
	case "port":
		port, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid port value: %w", err)
		}
		c.RabbitMQ.Port = port
	case "user":
		c.RabbitMQ.User = value
	case "password":
		c.RabbitMQ.Password = value
	default:
		return fmt.Errorf("unknown rabbitmq key: %s", key)
	}
	return nil
}

// DatabaseURL returns a PostgreSQL connection URL
func (c *Config) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		c.Database.User, c.Database.Password, c.Database.Host, c.Database.Port, c.Database.Database)
}

// RabbitMQURL returns an AMQP connection URL
func (c *Config) RabbitMQURL() string {
	return fmt.Sprintf("amqp://%s:%s@%s:%d/",
		c.RabbitMQ.User, c.RabbitMQ.Password, c.RabbitMQ.Host, c.RabbitMQ.Port)
}
