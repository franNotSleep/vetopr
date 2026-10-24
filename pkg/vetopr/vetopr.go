package vetopr

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

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

// 2. build project
func buildProject(workPath string, serviceCfg ServiceConfig) error {

	workPath = path.Join(workPath, serviceCfg.Base)

	cmd := exec.Command("/bin/sh", "-c", serviceCfg.Build)
	cmd.Dir = workPath

	err := startCmd(cmd)
	if err != nil {
		return fmt.Errorf("startCmd(cmd) = %v", err)
	}

	//	readerErr, err := cmd.StderrPipe()
	//	if err != nil {
	//		return fmt.Errorf("cmd.StderrPipe() = %v", err)
	//	}
	//
	//	err = cmd.Start()
	//	if err != nil {
	//		return fmt.Errorf("cmd.Start() = %v", err)
	//	}
	//
	//	_, err = io.Copy(os.Stdout, reader)
	//	if err != nil {
	//		return fmt.Errorf("io.Copy(os.Stdin, reader) = %v", err)
	//	}
	//
	//	_, err = io.Copy(os.Stdout, reader)
	//	if err != nil {
	//		return fmt.Errorf("io.Copy(os.Stdin, reader) = %v", err)
	//	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("cmd.Wait() = %v", err)
	}

	return nil
}

func ls(workpath string) (string, error) {
	cmd := exec.Command("ls")
	cmd.Dir = workpath
	output, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}
	if err := cmd.Start(); err != nil {
		return "", err
	}

	buf := make([]byte, 4096)
	_, err = output.Read(buf)
	if err != nil {
		return "", err
	}

	if err := cmd.Wait(); err != nil {
		return "", err
	}

	return string(buf), nil
}

// 3. turn project
func startProject() {

}

func startCmd(cmd *exec.Cmd) error {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("cmd.StdoutPipe() = %v", err)
	}

	readerErr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("cmd.StderrPipe() = %v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("cmd.Start() = %v", err)
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return fmt.Errorf("io.Copy(os.Stdin, reader) = %v", err)
	}

	_, err = io.Copy(os.Stderr, readerErr)
	if err != nil {
		return fmt.Errorf("io.Copy(os.Stderr, readerErr) = %v", err)
	}

	return nil
}
