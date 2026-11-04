package vetopr

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
)

func (vs *VetoprService) start() (*os.Process, error) {
	switch vs.Type {
	case UserType:
		return vs.startUserService()
	case DatabaseType:
		return vs.startDatabaseService()
	}

	return nil, fmt.Errorf("Invalid Service type %s\n", vs.Type)
}

func (vs *VetoprService) startUserService() (*os.Process, error) {
	switch vs.Type {
	case UserType:

	}
	if err := vs.build(); err != nil {
		return nil, err
	}

	workPath := path.Join(vs.Config.Workpath, vs.Config.Base)

	cmd := exec.Command("/bin/sh", "-c", vs.Config.Start)
	cmd.Dir = workPath

	p, err := vs.startCmd(cmd, false)
	if err != nil {
		return nil, fmt.Errorf("startCmd(cmd) = %v", err)
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

func (vs *VetoprService) startDatabaseService() (*os.Process, error) {

}
