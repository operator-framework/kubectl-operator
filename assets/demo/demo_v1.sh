#!/usr/bin/env bash

INTERACTIVE=${INTERACTIVE:-"1"}

run() {
	type "kubectl operator olmv1 --help"
	type "kubectl operator olmv1 get catalog"
}

prompt() {
	echo ""
	echo -n "$ "
}

type() {
	prompt
	sleep 1
	for (( i=0; i<${#1}; i++ )); do
		echo -n "${1:$i:1}"
		sleep 0.06
	done
	echo ""
	sleep 0.25
	eval $1
	[[ "$INTERACTIVE" == "1" ]] && read -p ""
}

run
