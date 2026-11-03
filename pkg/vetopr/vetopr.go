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
	Config   VetoprConfig
}

func NewVetoprD(workpath string, configPath string) (*VetoprD, error) {
	cfg, err := readProjectConfig(configPath)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Config: %+v\n", cfg)

	vd := &VetoprD{Services: []*VetoprService{}, Workpath: workpath}
	for _, v := range cfg.Services {
		vd.addService(v)
	}

	return vd, nil
}

func (vd *VetoprD) addService(cfg ServiceConfig) *VetoprService {
	if cfg.Workpath == "" {
		cfg.Workpath = vd.Workpath
	}

	vs := &VetoprService{Process: nil, Config: cfg}
	vd.Services = append(vd.Services, vs)
	return vs
}

func (vd *VetoprD) RunAll() error {
	for _, v := range vd.Services {
		_, err := vd.Run(v.Config.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (vd *VetoprD) Run(serviceKey string) (*os.Process, error) {
	s := vd.FindService(serviceKey)
	if s == nil {
		return nil, errServiceNotFound(serviceKey)
	}

	fmt.Printf("Building project %s...\n", s.Config.Name)
	if err := s.build(); err != nil {
		return nil, err
	}

	return s.start()
}

func (vd *VetoprD) Stop(serviceKey string) error {
	s := vd.FindService(serviceKey)
	if s == nil {
		return errServiceNotFound(serviceKey)
	}

	return s.Process.Kill()
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
	Process    *os.Process
	Config     ServiceConfig
	StdoutRead *io.ReadCloser
	StderrRead *io.ReadCloser
}

func (vs *VetoprService) start() (*os.Process, error) {
	workPath := path.Join(vs.Config.Workpath, vs.Config.Base)

	cmd := exec.Command("/bin/sh", "-c", vs.Config.Start)
	cmd.Dir = workPath

	p, err := vs.startCmd(cmd, false)
	if err != nil {
		return nil, fmt.Errorf("startCmd(cmd) = %v", err)
	}

	if err = cmd.Wait(); err != nil {
		return nil, fmt.Errorf("cmd.Wait() = %v", err)
	}

	go func() {
		reader := *vs.StdoutRead
		readerErr := *vs.StderrRead

		_, err = io.Copy(os.Stdout, reader)
		if err != nil {
			fmt.Printf("io.Copy(os.Stdin, reader) = %v", err)
		}
		defer reader.Close()

		_, err = io.Copy(os.Stderr, readerErr)
		if err != nil {
			fmt.Printf("io.Copy(os.Stderr, readerErr) = %v", err)
		}
		defer readerErr.Close()
	}()

	vs.Process = p
	return p, err
}

func (vs *VetoprService) build() error {
	workPath := path.Join(vs.Config.Workpath, vs.Config.Base)

	cmd := exec.Command("/bin/sh", "-c", vs.Config.Build)
	cmd.Dir = workPath

	_, err := vs.startCmd(cmd, true)
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

func (vs *VetoprService) startCmd(cmd *exec.Cmd, wait bool) (*os.Process, error) {
	reader, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("cmd.StdoutPipe() = %v", err)
	}
	vs.StdoutRead = &reader

	readerErr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("cmd.StderrPipe() = %v", err)
	}
	vs.StderrRead = &readerErr

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("cmd.Start() = %v", err)
	}

	if wait {
		return cmd.Process, cmd.Wait()
	}

	return cmd.Process, nil
}
