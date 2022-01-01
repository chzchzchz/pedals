package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"sync"
	"time"
)

const devInputPath = "/dev/input/by-id"
const cmdTimeout = 100 * time.Millisecond
const holdDuration = 500 * time.Millisecond

func listDevices() {
	fmt.Printf("devices (%s):\n", devInputPath)
	des, err := os.ReadDir(devInputPath)
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}
		if ty := info.Mode() & fs.ModeType; ty == fs.ModeSymlink {
			fmt.Println(de.Name())
		}
	}
	if err != nil {
		panic(err)
	}
}

func main() {
	/* I see:
	   usb-1a86_e026-event-if00 -> ../event12
	   usb-1a86_e026-event-if01 -> ../event13
	   usb-1a86_e026-event-joystick -> ../event11
	   usb-1a86_e026-event-kbd -> ../event9
	   usb-1a86_e026-event-mouse -> ../event10
	   usb-1a86_e026-joystick -> ../js0
	   usb-1a86_e026-mouse -> ../mouse2
	   and endpoints show up in lsusb but X maps it as a,b,c only(?)
	*/
	if len(os.Args) < 2 {
		fmt.Println("usage: pedals config.json")
		listDevices()
		os.Exit(1)
	}

	// Serialize all commands through cmdc.
	cmdc := make(chan []string, 3)
	defer close(cmdc)
	go func() {
		for cmdArgs := range cmdc {
			ctx, cancel := context.WithTimeout(context.TODO(), cmdTimeout)
			cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
			cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
			err := cmd.Run()
			cancel()
			if err != nil {
				fmt.Println(err)
			}
		}
	}()

	// Launch io goroutine for every device configuration.
	var wg sync.WaitGroup
	for _, d := range mustLoadConfig(os.Args[1]) {
		wg.Add(1)
		go func(dc *DeviceConfig) {
			defer wg.Done()
			dev := Device{DeviceConfig: dc, cmdc: cmdc}
			dev.RunLoop()
		}(&d)
	}
	wg.Wait()
}
