#!/bin/bash

winid=`xdotool search " - Ardour"`
xdocmd="xdotool key --window $winid"

playstop() {
	if [ -e running ]; then
		rm running
		$xdocmd space
		echo ======stopping record===========
	else
		fn=`date "+%Y-%m-%d-%H:%M:%S"`
		touch running
		$xdocmd Shift+space
	fi
}

record() {
	$xdocmd Shift+R
}

home() {
	$xdocmd Home
}

$1
