#!/bin/bash

# send stop message
midisend -p "Midi Through Port-0" -s FC
killall -9 jack_capture
