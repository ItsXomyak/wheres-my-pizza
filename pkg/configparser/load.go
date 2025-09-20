package configparser

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadYamlFile reads a YAML file and loads variables into the environment
func LoadYamlFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open YAML file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	currentIndent := 0
	prefixStack := []string{}

	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), " ")

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Calculate indentation
		indent := 0
		for _, ch := range line {
			if ch != ' ' {
				break
			}
			indent++
		}

		// Update prefix stack based on indentation
		for indent < currentIndent && len(prefixStack) > 0 {
			prefixStack = prefixStack[:len(prefixStack)-1]
			currentIndent -= 2 // Standard YAML indent is 2 spaces
		}
		currentIndent = indent

		// Parse the line
		content := strings.TrimSpace(line)
		if strings.HasSuffix(content, ":") {
			// This is a new section
			sectionName := strings.TrimSuffix(content, ":")
			prefixStack = append(prefixStack, sectionName)
			currentIndent += 2
		} else {
			// This is a key-value pair
			parts := strings.SplitN(content, ":", 2)
			if len(parts) != 2 {
				continue // Skip malformed lines
			}

			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Build the full env var name with prefixes
			fullKey := strings.Join(append(prefixStack, key), "_")
			fullKey = strings.ToUpper(fullKey)

			// Remove quotes if present
			value = strings.Trim(value, `"'`)

			// Handle empty values (YAML allows just "key:")
			if value == "" {
				value = "true" // Default value for empty YAML fields
			}

			// Set the environment variable
			if err := os.Setenv(fullKey, value); err != nil {
				return fmt.Errorf("could not set env var %s: %w", fullKey, err)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading YAML file: %w", err)
	}

	return nil
}
