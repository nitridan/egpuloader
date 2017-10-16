package main

import (
	"flag"

	"github.com/getlantern/systray"
)

func main() {
	isLoad := flag.Bool("loaddevice", false, "should be provided in order to load device")
	syspath := flag.String("sysPath", "", "name of device to load")
	flag.Parse()

	if *isLoad {
		loader := createDeviceLoader()
		loader.loadDevice(*syspath)
	} else {
		tray := createEgpuTray()
		systray.Run(tray.onReady, tray.onExit)
	}
}
