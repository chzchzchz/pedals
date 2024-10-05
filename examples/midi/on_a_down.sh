#!/bin/bash

MIDIPORT=${MIDIPORT:-"MIDI4x4 MIDI Out 4"}
WAVDIR=${WAVDIR:-`pwd`}

if pgrep jack_capture ; then
	echo already capturing
	exit 0
fi
# send start message to midi playback
midisend -p "$MIDIPORT" -s FA
wavout=$WAVDIR/`date +%s.%N`.wav
jack_capture --no-stdin --channels 2 --port system:capture_1 --port system:capture_2 $wavout
