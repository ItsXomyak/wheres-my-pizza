package config

import (
	"path/filepath"
	"testing"
)

func TestLoad_ConfigYAML(t *testing.T) {
	path := filepath.Join("..", "..", "config.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Database.Host == "" {
		t.Fatalf("expected database.host to be set")
	}
	if cfg.RabbitMQ.Port == 0 {
		t.Fatalf("expected rabbitmq.port to be set")
	}
}
