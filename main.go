package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/gvalkov/golang-evdev"
)

const devInputPath = "/dev/input/by-id"
const cmdTimeout = 100 * time.Millisecond

type KeyConfig struct {
	Up   []string
	Down []string
	Hold []string
}

type DeviceConfig struct {
	Device string
	Keys   map[string]KeyConfig
}

var ev2key = map[int]string{
	evdev.KEY_A: "a",
	evdev.KEY_B: "b",
	evdev.KEY_C: "c",
}

func (dc *DeviceConfig) LookupKeyConfig(k int) *KeyConfig {
	s, ok := ev2key[k]
	if !ok {
		panic("can't translate")
	}
	kc, ok := dc.Keys[s]
	if !ok {
		return nil
	}
	return &kc
}

func mustLoadConfig(path string) (devs []DeviceConfig) {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, &devs); err != nil {
		panic(err)
	}
	return devs
}

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
	cmdc := make(chan *exec.Cmd, 3)
	go func() {
		for cmd := range cmdc {
			if err := cmd.Run(); err != nil {
				fmt.Println(err)
			}
		}
	}()

	// Launch io goroutine for every device.
	var wg sync.WaitGroup
	for _, dc := range mustLoadConfig(os.Args[1]) {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := RunDevice(dc, cmdc); err != nil {
				panic(err)
			}
		}()
	}
	wg.Wait()
}

func RunDevice(dc DeviceConfig, cmdc chan<- *exec.Cmd) error {
	devpath := filepath.Join(devInputPath, filepath.Base(dc.Device))
	log.Println("opening", devpath)
	kbd, err := evdev.Open(devpath)
	if err != nil {
		return err
	}
	defer kbd.File.Close()
	defer kbd.Release()
	kbd.Grab()
	for {
		ev, err := kbd.ReadOne()
		if err != nil {
			return err
		}
		if ev.Type != evdev.EV_KEY {
			continue
		}
		keyev := evdev.NewKeyEvent(ev)
		kc := dc.LookupKeyConfig(int(keyev.Scancode))
		if kc == nil {
			continue
		}
		var cmdArgs []string
		if keyev.State == evdev.KeyUp {
			cmdArgs = kc.Up
		} else if keyev.State == evdev.KeyDown {
			cmdArgs = kc.Down
		} else if keyev.State == evdev.KeyHold {
			cmdArgs = kc.Hold
		}
		if len(cmdArgs) == 0 {
			continue
		}
		ctx, _ := context.WithTimeout(context.TODO(), cmdTimeout)
		cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
		cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
		cmdc <- cmd
	}
}
