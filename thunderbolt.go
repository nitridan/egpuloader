package main

import (
	"io/ioutil"
	"os"
	"path/filepath"

	udev "github.com/jochenvg/go-udev"
)

const (
	subsystem    = "thunderbolt"
	deviceType   = "thunderbolt_device"
	removeAction = "remove"
)

type deviceProvider interface {
	getDevices() chan ThunderboltDevice
	monitorDevices(done chan struct{}) chan ThunderboltDevice
}

type thunderboltDeviceProvider struct{}

type ThunderboltDevice struct {
	deviceName string
	vendorName string
	uniqueID   string
	authorized bool
	sysPath    string
	action     string
}

func createDeviceProvider() deviceProvider {
	return &thunderboltDeviceProvider{}
}

func (self *ThunderboltDevice) TryAuthorize() bool {
	if self.authorized || self.action == removeAction {
		return false
	}

	err := ioutil.WriteFile(filepath.Join(self.sysPath, "authorized"), []byte("1"), os.ModeDevice)
	if err != nil {
		panic(err)
	}

	return true
}

func (self *thunderboltDeviceProvider) toThunderboltDevice(device *udev.Device) ThunderboltDevice {
	thunderboltDevice := ThunderboltDevice{
		action:     device.Action(),
		sysPath:    device.Syspath(),
		deviceName: device.SysattrValue("device_name"),
		uniqueID:   device.SysattrValue("unique_id"),
		vendorName: device.SysattrValue("vendor_name"),
	}

	if thunderboltDevice.action == removeAction {
		return thunderboltDevice
	}

	authorized := device.SysattrValue("authorized")
	switch authorized {
	case "0":
		thunderboltDevice.authorized = false
	case "1":
		thunderboltDevice.authorized = true
	default:
		panic("Unknown value for authorized property: " + authorized)
	}

	return thunderboltDevice
}

func (self *thunderboltDeviceProvider) getDevices() chan ThunderboltDevice {
	u := udev.Udev{}
	e := u.NewEnumerate()
	dsp, err := e.Devices()
	if err != nil {
		panic(err)
	}

	thunderboltDevices := make(chan ThunderboltDevice)
	go func() {
		for _, device := range dsp {
			if device.Subsystem() != subsystem || device.PropertyValue("DEVTYPE") != deviceType {
				continue
			}

			thunderboltDevices <- self.toThunderboltDevice(device)
		}

		close(thunderboltDevices)
	}()

	return thunderboltDevices
}

func (self *thunderboltDeviceProvider) monitorDevices(done chan struct{}) chan ThunderboltDevice {
	udev := udev.Udev{}
	monitor := udev.NewMonitorFromNetlink("udev")
	e := monitor.FilterAddMatchSubsystemDevtype(subsystem, deviceType)
	if e != nil {
		panic(e)
	}

	c, e := monitor.DeviceChan(done)
	if e != nil {
		panic(e)
	}

	resultChan := make(chan ThunderboltDevice)
	go func() {
		select {
		case <-done:
			close(resultChan)
		default:
		}
	}()

	go func() {
		for device := range c {
			resultChan <- self.toThunderboltDevice(device)
		}
	}()

	return resultChan
}
