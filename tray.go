package main

import (
	"fmt"
	"os"

	"github.com/esiqveland/notify"
	"github.com/getlantern/systray"
	"github.com/godbus/dbus"
)

type namedMenuItem struct {
	name      string
	menuItem  *systray.MenuItem
	isEnabled bool
}

type egpuTray struct {
	monitorChannel chan struct{}
	menuItemMap    map[string]*namedMenuItem
}

type egpuTrayWrapper interface {
	onReady()
	onExit()
}

func createEgpuTray() egpuTrayWrapper {
	return &egpuTray{
		monitorChannel: make(chan struct{}, 3),
		menuItemMap:    make(map[string]*namedMenuItem),
	}
}

func (tray *egpuTray) printDevice(device *ThunderboltDevice) {
	fmt.Println("Action:")
	fmt.Println(device.action)
	fmt.Println("System path:")
	fmt.Println(device.sysPath)
	fmt.Println("Is device authorized:")
	fmt.Println(device.authorized)
	fmt.Println("Device name:")
	fmt.Println(device.deviceName)
	fmt.Println("Vendor name:")
	fmt.Println(device.vendorName)
	fmt.Println("Id:")
	fmt.Println(device.uniqueID)
}

func (tray *egpuTray) onStart() {
	deviceProvider := createDeviceProvider()
	for device := range deviceProvider.getDevices() {
		if device.authorized {
			continue
		}

		tray.printDevice(&device)
		tray.menuItemMap[device.sysPath] = tray.getOrCreateMenuItem(&device)
	}

	tray.refreshVisisilityState()
}

func (tray *egpuTray) handleRemove(device *ThunderboltDevice) {
	if device.action != removeAction {
		return
	}

	item := tray.menuItemMap[device.sysPath]
	if item != nil {
		item.isEnabled = false
		tray.showNotification(device, false)
		tray.refreshVisisilityState()
	}
}

func (tray *egpuTray) refreshVisisilityState() {
	for _, v := range tray.menuItemMap {
		if v.isEnabled {
			v.menuItem.Enable()
			v.menuItem.Show()
		} else {
			v.menuItem.Disable()
			v.menuItem.Hide()
		}
	}
}

func (tray *egpuTray) showNotification(device *ThunderboltDevice, isAdded bool) {
	conn, e := dbus.SessionBus()
	if e != nil {
		panic(e)
	}

	n := notify.Notification{
		AppName:       "egpuloader",
		AppIcon:       "video-display",
		Hints:         map[string]dbus.Variant{},
		ExpireTimeout: int32(5000),
	}

	if isAdded {
		n.Summary = "Device: " + device.deviceName + " was connected"
		n.Body = "Please authorize connected device and restart desktop manager (requires root)"
	} else {
		n.Summary = "Device: " + tray.menuItemMap[device.sysPath].name + " was disconnected"
	}

	_, err := notify.SendNotification(conn, n)
	if err != nil {
		fmt.Println("Error sending push notification: " + err.Error())
	}
}

func (tray *egpuTray) getOrCreateMenuItem(device *ThunderboltDevice) *namedMenuItem {
	item := tray.menuItemMap[device.sysPath]
	if item != nil {
		item.isEnabled = true
		return item
	}

	sysPath := device.sysPath
	deviceMount := systray.AddMenuItem("Load: "+device.deviceName, "Authorizes video adapter")
	go func() {
		<-deviceMount.ClickedCh
		fmt.Println("Mounting device: " + sysPath)
		loader := createDeviceLoader()
		loader.loadSudo(sysPath)
	}()

	return &namedMenuItem{
		menuItem:  deviceMount,
		name:      device.deviceName,
		isEnabled: true,
	}
}

func (tray *egpuTray) handleAdd(device *ThunderboltDevice) {
	if device.action != "add" || device.authorized {
		return
	}

	tray.showNotification(device, true)
	tray.menuItemMap[device.sysPath] = tray.getOrCreateMenuItem(device)
	tray.refreshVisisilityState()
}

func (tray *egpuTray) onReady() {
	go func() {
		deviceProvider := createDeviceProvider()
		for device := range deviceProvider.monitorDevices(tray.monitorChannel) {
			tray.printDevice(&device)
			tray.handleAdd(&device)
			tray.handleRemove(&device)
		}
	}()

	systray.SetIcon(gpuIconData)
	systray.SetTitle("External GPU loader widget")
	systray.SetTooltip("External GPU loader")
	mQuitOrig := systray.AddMenuItem("Quit", "Quit the whole app")
	go func() {
		<-mQuitOrig.ClickedCh
		fmt.Println("Requesting quit")
		var s struct{}
		tray.monitorChannel <- s
		close(tray.monitorChannel)
		systray.Quit()
		fmt.Println("Finished quitting")
		os.Exit(0)
	}()

	tray.onStart()
}

func (tray *egpuTray) onExit() {
	var s struct{}
	tray.monitorChannel <- s
	close(tray.monitorChannel)
	fmt.Println("Finished onExit")
}
