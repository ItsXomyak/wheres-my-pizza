package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config represents the application's configuration loaded from YAML.
type Config struct {
	Database DatabaseConfig `yaml:"database"`
	RabbitMQ RabbitMQConfig `yaml:"rabbitmq"`
}

type DatabaseConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Database string `yaml:"database"`
}

type RabbitMQConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

// Load reads a YAML file from path and unmarshals it into Config.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}
	// Parse the (simple) YAML into a generic map without pulling an external yaml
	// dependency. This parser supports basic mappings and integer/boolean/scalar
	// values which is sufficient for this project's config.yaml.
	m, err := parseSimpleYAMLToMap(string(b))
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	jb, err := json.Marshal(m)
	if err != nil {
		return nil, fmt.Errorf("marshal intermediate json: %w", err)
	}

	var c Config
	if err := json.Unmarshal(jb, &c); err != nil {
		return nil, fmt.Errorf("unmarshal config json: %w", err)
	}

	// Basic validation
	if c.Database.Host == "" || c.Database.Port == 0 {
		return nil, fmt.Errorf("invalid database configuration: %+v", c.Database)
	}
	if c.RabbitMQ.Host == "" || c.RabbitMQ.Port == 0 {
		return nil, fmt.Errorf("invalid rabbitmq configuration: %+v", c.RabbitMQ)
	}

	return &c, nil
}

// parseSimpleYAMLToMap parses a very small subset of YAML sufficient for
// the repository's config.yaml: top-level mappings, nested mappings (by
// indentation), and scalar values (strings, integers, booleans). It does
// not support sequences, anchors, or complex types.
func parseSimpleYAMLToMap(s string) (map[string]interface{}, error) {
	lines := strings.Split(s, "\n")
	root := map[string]interface{}{}

	type frame struct {
		indent int
		node   map[string]interface{}
	}

	// stack holds the nested maps with their indentation level. Start with root
	// at indent -1 so top-level keys (indent 0) attach to it.
	stack := []frame{{indent: -1, node: root}}

	for _, raw := range lines {
		// trim CR for Windows files and skip empty/comment lines
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		ts := strings.TrimSpace(line)
		if strings.HasPrefix(ts, "#") {
			continue
		}

		// count leading spaces (assume spaces used for indentation)
		i := 0
		for i < len(line) && line[i] == ' ' {
			i++
		}
		indent := i

		// split key: value
		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)
		key := strings.TrimSpace(parts[0])
		var valStr string
		if len(parts) > 1 {
			valStr = strings.TrimSpace(parts[1])
		}

		// find parent frame: the most recent frame with indent < current indent
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1].node

		if valStr == "" {
			// a new nested mapping
			nm := map[string]interface{}{}
			parent[key] = nm
			stack = append(stack, frame{indent: indent, node: nm})
			continue
		}

		// parse scalar value: try int, bool, then string
		if iv, err := strconv.Atoi(valStr); err == nil {
			parent[key] = iv
			continue
		}
		if valStr == "true" || valStr == "false" {
			parent[key] = (valStr == "true")
			continue
		}
		// strip optional quotes
		if len(valStr) >= 2 && ((valStr[0] == '\'' && valStr[len(valStr)-1] == '\'') || (valStr[0] == '"' && valStr[len(valStr)-1] == '"')) {
			parent[key] = valStr[1 : len(valStr)-1]
			continue
		}
		parent[key] = valStr
	}

	return root, nil
}
