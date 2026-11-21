package vm

import "testing"

func TestSetupVM(t *testing.T) {
	if err := setupFirecrackerVM(1); err != nil {
		t.Fatal(err)
	}
}
