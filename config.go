package main

import (
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/gvalkov/golang-evdev"
)

var ev2key = map[int]string{
	evdev.KEY_A: "a",
	evdev.KEY_B: "b",
	evdev.KEY_C: "c",
}

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

func (dc *DeviceConfig) path() string {
	if dc.Device[0] == '/' {
		return dc.Device
	}
	return filepath.Join(devInputPath, filepath.Base(dc.Device))
}
