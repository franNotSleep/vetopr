package vetopr

import (
	"fmt"
	"io"
	"os"

	"github.com/goccy/go-yaml"
)

type ServiceConfig struct {
	Base  string `yaml:"base"`
	Build string `yaml:"build"`
	Start string `yaml:"start"`
}

// type VetoprConfig map[any]any
type VetoprConfig struct {
	Services map[any]ServiceConfig `yaml:"services"`
}

func handle(workPath string) error {
	cfg, err := readProjectConfig(workPath + "/vetopr.yaml")
	if err != nil {
		return err
	}
	fmt.Printf("CFG: %+v\n", cfg)
	return nil
}

// 1. read the config on how to turn each project
func readProjectConfig(path string) (VetoprConfig, error) {
	f, err := os.Open(path)
	var cfg VetoprConfig

	if err != nil {
		return cfg, err
	}
	bytes, err := io.ReadAll(f)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(bytes, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// 2. turn each project
func startProject() {

}
