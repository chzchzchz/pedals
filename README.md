# pedals

A little program for using pedals I bought off aliexpress.

## Build and Run 

```sh
go get github.com/chzchzchz/pedals
pedals # lists devices
pedals your_config.json
```

## json example

Bind to device `usb-1a86_e026-event-kbd` and register handlers on key "a":

```json
[
{
  "device" : "usb-1a86_e026-event-kbd",
  "keys" : {"a" : {"up" : ["on_up.sh"], "down" : ["on_down.sh", "arg1"], "hold" : ["abc"]  }}
}
]
```