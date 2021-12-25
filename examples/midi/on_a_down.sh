#!/bin/bash

# send start message to midi playback
midisend -p "Midi Through Port-0" -s FA
wavout=`date +%s.%N`.wav
jack_capture --no-stdin --channels 1 --port system:playback_1 $wavout
