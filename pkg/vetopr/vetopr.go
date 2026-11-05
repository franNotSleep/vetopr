package vetopr

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/moby/moby/client"
)

func errServiceNotFound(key ...string) error {
	if len(key) == 0 {
		return fmt.Errorf("Service not found\n")
	}
	return fmt.Errorf("Service with key \"%s\" was not found\n", key[0])
}

func NewVetoprD(workpath string, configPath string) (*VetoprD, error) {
	dockerClient, err := client.New(client.FromEnv)
	if err != nil {
		return nil, err
	}

	cfg, err := readProjectConfig(configPath)
	if err != nil {
		return nil, err
	}

	vd := &VetoprD{Services: []*VetoprService{}, Workpath: workpath}
	for _, v := range cfg.Services {
		if v.Workpath == "" {
			v.Workpath = vd.Workpath
		}

		workpath := path.Join(v.Workpath, v.Base)

		buildCommand := exec.Command("/bin/sh", "-c", v.Build)
		buildCommand.Dir = workpath

		startCommand := exec.Command("/bin/sh", "-c", v.Start)
		startCommand.Dir = workpath

		vs := &VetoprService{
			Process: nil,
			Name:    v.Name,
			Type:    UserType,
			Cmds: []*VetoprServiceCommand{
				&VetoprServiceCommand{
					cmd: buildCommand,
					id:  "build",
				},
				&VetoprServiceCommand{
					cmd: startCommand,
					id:  "start",
				},
			}}

		vd.Services = append(vd.Services, vs)
	}

	for _, v := range cfg.Databases {
		vs := &VetoprService{Process: nil, Name: v.Name, Workpath: "", Type: DatabaseType}
		vd.Services = append(vd.Services, vs)
	}

	vd.dockerClient = dockerClient

	return vd, nil
}

func (vd *VetoprD) startBox() error {
	dclient := vd.dockerClient
	c, err := dclient.ContainerCreate(context.Background(), client.ContainerCreateOptions{
		Image: "alpine:3.14",
		Name:  fmt.Sprintf("vetopr-%s", time.Now().UnixMilli()),
	})

	if err != nil {
		return err
	}
	vd.boxID = c.ID
	return nil
}

func (vd *VetoprD) RunAll() error {
	for _, v := range vd.Services {
		_, err := vd.Run(v.Name)
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

	return s.start()
}

func (vd *VetoprD) Stop(serviceKey string) error {
	s := vd.FindService(serviceKey)
	if s == nil {
		return errServiceNotFound(serviceKey)
	}

	fmt.Printf("Stoppping %s service\n", serviceKey)
	return s.Process.Kill()
}

func (vd *VetoprD) FindService(serviceKey string) *VetoprService {
	for _, s := range vd.Services {
		if s.Name == serviceKey {
			return s
		}
	}
	return nil
}

func (vs *VetoprService) build() error {
	cmd := exec.Command("/bin/sh", "-c", vs.Config.Build)
	cmd.Dir = vs.Workpath

	_, err := vs.startCmd(cmd, true)
	if err != nil {
		return fmt.Errorf("startCmd(cmd) = %v", err)
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
		err := cmd.Wait()
		if err != nil {
			return nil, fmt.Errorf("cmd.Wait() = %v", err)
		}

		return cmd.Process, nil
	}

	return cmd.Process, nil
}
