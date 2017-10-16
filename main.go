package main

import (
	"flag"
	"fmt"

	"github.com/getlantern/systray"
)

func main() {
	isLoad := flag.Bool("loaddevice", false, "should be provided in order to load device")
	syspath := flag.String("sysPath", "", "name of device to load")
	flag.Parse()
	fmt.Println(*isLoad)
	fmt.Println(*syspath)
	if *isLoad {
		loader := createDeviceLoader()
		loader.loadDevice(*syspath)
	} else {
		tray := createEgpuTray()
		systray.Run(tray.onReady, tray.onExit)
	}
}
