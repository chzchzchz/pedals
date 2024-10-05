#!/bin/bash

MIDIPORT=${MIDIPORT:-"MIDI4x4 MIDI Out 4"}

# send stop message
midisend -p "$MIDIPORT" -s FC
killall -SIGINT jack_capture
