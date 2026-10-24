package vetopr

import (
	"fmt"
	"os"
	"path"
	"sync"
	"testing"
)

const (
	SNAPSHOTS_PATH = "../../snapshots"
)

var (
	findFirstUnzippedOnce sync.Once
	testPRPath            string
)

func getPR() (string, error) {
	var findErr error = nil
	findFirstUnzippedOnce.Do(func() {
		entries, err := os.ReadDir(SNAPSHOTS_PATH)
		if err != nil {
			findErr = err
			return
		}

		for _, entry := range entries {
			if entry.IsDir() {
				testPRPath = fmt.Sprintf("%s/%s", SNAPSHOTS_PATH, entry.Name())
			}
		}
	})

	return testPRPath, findErr
}

func getServiceCfg(service string, vetoPrConfig VetoprConfig) (ServiceConfig, error) {
	var zero ServiceConfig
	srvCfg, ok := vetoPrConfig.Services[service]
	if !ok {
		return zero, fmt.Errorf("Service %s is not a key on Config", service)
	}

	return srvCfg, nil
}

func TestReadConfig(t *testing.T) {
	prPath, err := getPR()
	if err != nil {
		t.Error(err)
		return
	}

	vetoPrConfig, err := readProjectConfig(path.Join(prPath, "vetopr.yaml"))
	if err != nil {
		t.Errorf("readProjectConfig: %+v", err)
		return
	}

	t.Logf("vetoPr: %+v\n", vetoPrConfig)
	srvCfg, ok := vetoPrConfig.Services["api"]
	if !ok {
		t.Errorf(`vetoPrConfig.Services["api"] = %q, %v`, vetoPrConfig, srvCfg)
		return
	}

	if srvCfg.Base != "./backend" {
		t.Errorf(`srvCfg.Base = %q, %v, want match for "./backend"`, srvCfg.Base, srvCfg)
		return
	}

	if srvCfg.Start != "npm run start" {
		t.Errorf(`srvCfg.Start = %q, %v, want match for "npm run start"`, srvCfg.Base, srvCfg)
		return
	}
}

func TestBuildProject(t *testing.T) {
	prPath, err := getPR()
	if err != nil {
		t.Error(err)
		return
	}

	vetoPrConfig, err := readProjectConfig(path.Join(prPath, "vetopr.yaml"))
	if err != nil {
		t.Errorf("readProjectConfig: %+v", err)
		return
	}

	apiCfg, err := getServiceCfg("api", vetoPrConfig)
	if err != nil {
		t.Errorf(`vgetServiceCfg("api", vetoPrConfig) =  %v`, err)
		return
	}

	buildFolderPath := path.Join(prPath, apiCfg.Base, "dist")
	if err := os.RemoveAll(buildFolderPath); err != nil {
		t.Errorf(`os.RemoveAll(path.Join(prPath, apiCfg.Base, "dist")) =  %v`, err)
		return
	}

	if err := buildProject(prPath, apiCfg); err != nil {
		t.Errorf(`buildProject(prPath, apiCfg) =  %v`, err)
		return
	}

	if _, err := os.Stat(buildFolderPath); err != nil {
		t.Errorf(`os.Stat(buildFolderPath) =  %v`, err)
		return
	}
}
