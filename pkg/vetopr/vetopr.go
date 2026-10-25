package vetopr

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/goccy/go-yaml"
)

func errServiceNotFound(key ...string) error {
	if len(key) == 0 {
		return fmt.Errorf("Service not found\n")
	}
	return fmt.Errorf("Service with key \"%s\" was not found\n", key[0])
}

type ServiceConfig struct {
	Base     string `yaml:"base"`
	Build    string `yaml:"build"`
	Start    string `yaml:"start"`
	Workpath string
	Name     string
}

// type VetoprConfig map[any]any
type VetoprConfig struct {
	Services map[any]ServiceConfig `yaml:"services"`
}

type VetoprD struct {
	Services []*VetoprService
	Workpath string
}

func NewVetoprD(workpath string) *VetoprD {
	return &VetoprD{Services: []*VetoprService{}, Workpath: workpath}
}

func (vd *VetoprD) AddService(cfg ServiceConfig) *VetoprService {
	if cfg.Workpath == "" {
		cfg.Workpath = vd.Workpath
	}

	return &VetoprService{PID: nil, Config: cfg}
}

func (vd *VetoprD) Run(serviceKey string) (int, error) {
	s := vd.FindService(serviceKey)
	if s == nil {
		return 0, errServiceNotFound(serviceKey)
	}

	fmt.Printf("Building project %s...\n", s.Config.Name)
	if err := s.build(); err != nil {
		return 0, err
	}

	return s.start()
}

func (vd *VetoprD) FindService(serviceKey string) *VetoprService {
	for _, s := range vd.Services {
		if s.Config.Name == serviceKey {
			return s
		}
	}
	return nil
}

type VetoprService struct {
	PID    *int
	Config ServiceConfig
}

func (vs *VetoprService) start() (int, error) {
	workPath := path.Join(vs.Config.Workpath, vs.Config.Base)

	cmd := exec.Command("/bin/sh", "-c", vs.Config.Start)
	cmd.Dir = workPath
	cmd.Start()

	pid, err := startCmd(cmd)
	if err != nil {
		return 0, fmt.Errorf("startCmd(cmd) = %v", err)
	}

	if err = cmd.Wait(); err != nil {
		return 0, fmt.Errorf("cmd.Wait() = %v", err)
	}

	vs.PID = &pid
	return pid, err
}

func (vs *VetoprService) build() error {
	workPath := path.Join(vs.Config.Workpath, vs.Config.Base)

	cmd := exec.Command("/bin/sh", "-c", vs.Config.Build)
	cmd.Dir = workPath

	_, err := startCmd(cmd)
	if err != nil {
		return fmt.Errorf("startCmd(cmd) = %v", err)
	}

	if err = cmd.Wait(); err != nil {
		return fmt.Errorf("cmd.Wait() = %v", err)
	}

	return nil
}

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

	for k, v := range cfg.Services {
		name, ok := k.(string)
		if ok {
			v.Name = name
			cfg.Services[k] = v
		}
	}

	return cfg, nil
}

func handle(workPath string) error {
	cfg, err := readProjectConfig(workPath + "/vetopr.yaml")
	if err != nil {
		return err
	}

	fmt.Printf("CFG: %+v\n", cfg)
	return nil
}

func startCmd(cmd *exec.Cmd) (int, error) {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return 0, fmt.Errorf("cmd.StdoutPipe() = %v", err)
	}

	readerErr, err := cmd.StderrPipe()
	if err != nil {
		return 0, fmt.Errorf("cmd.StderrPipe() = %v", err)
	}

	if err := cmd.Start(); err != nil {
		return 0, fmt.Errorf("cmd.Start() = %v", err)
	}

	go func(reader io.Reader, readerErr io.Reader) {
		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			fmt.Printf("io.Copy(os.Stdin, reader) = %v", err)
		}

		_, err = io.Copy(os.Stderr, readerErr)
		if err != nil {
			fmt.Printf("io.Copy(os.Stderr, readerErr) = %v", err)
		}
	}(reader, readerErr)

	return cmd.Process.Pid, nil
}
