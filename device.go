package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/gvalkov/golang-evdev"
)

type Device struct {
	*DeviceConfig
	cmdc chan<- []string
	mu   sync.Mutex
	wg   sync.WaitGroup
}

func (d *Device) RunLoop() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()
	for {
		err := d.Run()
		log.Printf("lost device: %v", err)
		err = watcher.Add(devInputPath)
		if err != nil {
			return err
		}
		for {
			var ev fsnotify.Event
			select {
			case ev = <-watcher.Events:
			case err := <-watcher.Errors:
				return err
			}
			if !ev.Has(fsnotify.Create) {
				continue
			}
			if _, err := os.Stat(d.path()); err == nil {
				break
			} else if !os.IsNotExist(err) {
				return err
			}
		}
		watcher.Remove(devInputPath)
		for len(watcher.Events) > 0 {
			<-watcher.Events
		}
	}
}

func (d *Device) Run() error {
	cancels := make(map[*context.CancelFunc]struct{})

	defer func() {
		d.mu.Lock()
		for c := range cancels {
			(*c)()
		}
		d.mu.Unlock()
		d.wg.Wait()
	}()

	kbd, err := evdev.Open(d.path())
	if err != nil {
		return err
	}
	defer kbd.File.Close()
	defer kbd.Release()
	log.Println("attached", d.path())
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
		kc := d.LookupKeyConfig(int(keyev.Scancode))
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
		if d.Concurrent {
			ctx, cancel := context.WithCancel(context.TODO())
			d.mu.Lock()
			cancels[&cancel] = struct{}{}
			d.mu.Unlock()
			d.wg.Add(1)
			go func() {
				defer func() {
					d.mu.Lock()
					delete(cancels, &cancel)
					d.mu.Unlock()
					d.wg.Done()
				}()
				cmd := exec.CommandContext(ctx, cmdArgs[0], cmdArgs[1:]...)
				cmd.Stdout, cmd.Stderr = os.Stdout, os.Stderr
				if err := cmd.Run(); err != nil {
					fmt.Println(err)
				}
				cancel()
			}()
		} else {
			d.cmdc <- cmdArgs
		}
	}
}
