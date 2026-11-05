package vetopr

import (
	"fmt"
	"io"
	"os"
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
	if err := vs.build(); err != nil {
		return nil, err
	}

	for _, cmd := range vs.Cmds {
		wait := true
		if cmd.id == "build" {
			wait = false
		}

		p, err := vs.startCmd(cmd.cmd, wait)
		if err != nil {
			return nil, fmt.Errorf("startCmd(cmd) = %v", err)
		}

		if cmd.id == "start" {
			vs.Process = p
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
	}

	return vs.Process, nil
}

func (vs *VetoprService) startDatabaseService() (*os.Process, error) {
	return nil, nil
}
