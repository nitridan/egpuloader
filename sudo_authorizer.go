package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
)

type deviceLoader interface {
	loadSudo(sysPath string)
	loadDevice(sysPath string)
}

type thunderboltDeviceLoader struct{}

func createDeviceLoader() deviceLoader {
	return &thunderboltDeviceLoader{}
}

func (self *thunderboltDeviceLoader) loadSudo(sysPath string) {
	ex, err := os.Executable()
	if err != nil {
		panic(err)
	}

	cmd := exec.Command("/usr/bin/pkexec", "--user", "root", ex, "--loaddevice", "--sysPath", sysPath)
	out, _ := self.runCommand(cmd)
	fmt.Println("PKexec output: " + out.String())
}

func (self *thunderboltDeviceLoader) loadDevice(sysPath string) {
	deviceProvider := createDeviceProvider()
	var d *ThunderboltDevice
	for device := range deviceProvider.getDevices() {
		if device.sysPath == sysPath {
			d = &device
			break
		}
	}

	if d == nil {
		fmt.Println("Specified device: '" + sysPath + "' not found")
		return
	}

	if d.TryAuthorize() {
		cmd := exec.Command("/usr/sbin/service", "lightdm", "restart")
		out, e := self.runCommand(cmd)
		fmt.Println("Service output: " + out.String())
		if e != nil {
			panic(e)
		}
	}
}

func (self *thunderboltDeviceLoader) runCommand(cmd *exec.Cmd) (bytes.Buffer, error) {
	var out bytes.Buffer
	cmd.Stdout = &out
	e := cmd.Run()
	if e != nil {
		fmt.Println(e)
	}

	return out, e
}
