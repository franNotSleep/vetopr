package vm

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"golang.org/x/crypto/ssh"
)

type VM struct {
	sshClient *ssh.Client
}

func StartVetoprVM() (*VM, error) {
	vm := &VM{}
	sshClient, err := getSSHClient()
	if err != nil {
		return vm, err
	}
	vm.sshClient = sshClient

	return vm, nil
}

func setupFirecrackerVM(vmID int) error {
	tap := fmt.Sprintf("tap%d", vmID)

	tapIPfirstPart := (4*vmID + 1) / 256
	tapIPSecondPart := (4*vmID + 1) % 256
	tapIP := fmt.Sprintf("172.16.%d.%d", tapIPfirstPart, tapIPSecondPart)

	guestIPfirstPart := (4*vmID + 2) / 256
	guestIPSecondPart := (4*vmID + 2) % 256
	guestIP := fmt.Sprintf("172.16.%d.%d", guestIPfirstPart, guestIPSecondPart)

	var macFirstPart = fmt.Sprintf("%x", guestIPfirstPart)
	if guestIPfirstPart <= 15 {
		macFirstPart = "0" + macFirstPart
	}

	var macSecondPart = fmt.Sprintf("%x", guestIPSecondPart)
	if guestIPSecondPart <= 15 {
		macSecondPart = "0" + macSecondPart
	}

	mac := fmt.Sprintf("06:00:AC:10:%s:%s", macFirstPart, macSecondPart)

	fmt.Printf("tap: %s\ntapIP: %s\nguestIP:%s\nmac: %s\n", tap, tapIP, guestIP, mac)

	cmd := exec.Command("./setup-vm.sh", fmt.Sprintf("%d", vmID), tap, tapIP, guestIP, mac)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}

	if err := cmd.Start(); err != nil {
		return err
	}

	_, err = io.Copy(os.Stdout, stdout)
	if err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func getSSHClient() (*ssh.Client, error) {
	var hostKey ssh.PublicKey
	key, err := os.ReadFile("/home/frannotsleep/dev/firecracker/ubuntu-24.04.id_rsa")
	if err != nil {
		return nil, err
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User: "root",
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.FixedHostKey(hostKey),
	}

	client, err := ssh.Dial("tcp", "172.16.0.2:22", sshConfig)
	return client, err
}
