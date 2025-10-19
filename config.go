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
	evdev.KEY_D: "d",
	evdev.KEY_E: "e",
	evdev.KEY_F: "f",
	evdev.KEY_G: "g",
	evdev.KEY_H: "h",
	evdev.KEY_I: "i",
	evdev.KEY_J: "j",
	evdev.KEY_K: "k",
	evdev.KEY_L: "l",
	evdev.KEY_M: "m",
	evdev.KEY_N: "n",
	evdev.KEY_O: "o",
	evdev.KEY_P: "p",
	evdev.KEY_Q: "q",
	evdev.KEY_R: "r",
	evdev.KEY_S: "s",
	evdev.KEY_T: "t",
	evdev.KEY_U: "u",
	evdev.KEY_V: "v",
	evdev.KEY_X: "x",
	evdev.KEY_Y: "y",
	evdev.KEY_Z: "z",

	evdev.KEY_0:   "0",
	evdev.KEY_1:   "1",
	evdev.KEY_2:   "2",
	evdev.KEY_3:   "3",
	evdev.KEY_4:   "4",
	evdev.KEY_5:   "5",
	evdev.KEY_6:   "6",
	evdev.KEY_7:   "7",
	evdev.KEY_8:   "8",
	evdev.KEY_9:   "9",
	evdev.KEY_KP0: "0",
	evdev.KEY_KP1: "1",
	evdev.KEY_KP2: "2",
	evdev.KEY_KP3: "3",
	evdev.KEY_KP4: "4",
	evdev.KEY_KP5: "5",
	evdev.KEY_KP6: "6",
	evdev.KEY_KP7: "7",
	evdev.KEY_KP8: "8",
	evdev.KEY_KP9: "9",

	evdev.KEY_KPENTER:    "enter",
	evdev.KEY_KPPLUS:     "plus",
	evdev.KEY_KPMINUS:    "minus",
	evdev.KEY_KPSLASH:    "slash",
	evdev.KEY_KPASTERISK: "asterisk",
	evdev.KEY_KPDOT:      "dot",

	evdev.KEY_BACKSPACE: "backspace",
	evdev.KEY_HOMEPAGE:  "homepage",
	evdev.KEY_MAIL:      "mail",
	evdev.KEY_EMAIL:     "mail",
	evdev.KEY_TAB:       "tab",
	evdev.KEY_CALC:      "calc",
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
