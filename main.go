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
	"gopkg.in/fsnotify.v0"
)

const devInputPath = "/dev/input/by-id"
const cmdTimeout = 100 * time.Millisecond

type KeyConfig struct {
	Up   []string
	Down []string
	Hold []string
}

type DeviceConfig struct {
	Device     string
	Concurrent bool
	Keys       map[string]KeyConfig
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

	// Launch io goroutine for every device.
	var wg sync.WaitGroup
	for _, d := range mustLoadConfig(os.Args[1]) {
		wg.Add(1)
		go func(dc *DeviceConfig) {
			defer wg.Done()
			dc.RunLoop(cmdc)
		}(&d)
	}
	wg.Wait()
}

func (dc *DeviceConfig) path() string {
	return filepath.Join(devInputPath, filepath.Base(dc.Device))
}

func (dc *DeviceConfig) RunLoop(cmdc chan<- []string) error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	for {
		err := dc.RunDevice(cmdc)
		log.Printf("lost device: %v", err)
		err = watcher.WatchFlags(devInputPath, fsnotify.FSN_CREATE)
		if err != nil {
			return err
		}
		for {
			if _, err := os.Stat(dc.path()); err == nil {
				break
			} else if !os.IsNotExist(err) {
				return err
			}
			select {
			case <-watcher.Event:
			case err := <-watcher.Error:
				return err
			}
		}
		watcher.RemoveWatch(dc.path())
	}
}

func (dc *DeviceConfig) RunDevice(cmdc chan<- []string) error {
	var mu sync.Mutex
	var wg sync.WaitGroup
	cancels := make(map[*context.CancelFunc]struct{})

	defer func() {
		mu.Lock()
		for c := range cancels {
			(*c)()
		}
		mu.Unlock()
		wg.Wait()
	}()

	kbd, err := evdev.Open(dc.path())
	if err != nil {
		return err
	}
	defer kbd.File.Close()
	defer kbd.Release()
	log.Println("attached", dc.path())
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
		if dc.Concurrent {
			ctx, cancel := context.WithCancel(context.TODO())
			mu.Lock()
			cancels[&cancel] = struct{}{}
			mu.Unlock()
			wg.Add(1)
			go func() {
				defer func() {
					mu.Lock()
					delete(cancels, &cancel)
					mu.Unlock()
					wg.Done()
				}()
				cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Println(err)
				}
				cancel()
			}()
		} else {
			cmdc <- cmdArgs
		}
	}
}
