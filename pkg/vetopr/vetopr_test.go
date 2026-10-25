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

	if srvCfg.Name != "api" {
		t.Errorf(`srvCfg.Name = %q, %v, want match for "api"`, srvCfg.Name, srvCfg)
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
