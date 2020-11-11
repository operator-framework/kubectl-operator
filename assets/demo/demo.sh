#!/usr/bin/env bash

INTERACTIVE=${INTERACTIVE:-"1"}

run() {
	type "kubectl operator catalog list -A"
	type "kubectl operator list-available cockroachdb"
	type "kubectl operator install cockroachdb --create-operator-group -v 2.1.1 -c stable"
	type "kubectl operator list"
	type "kubectl operator upgrade cockroachdb"
	type "kubectl operator list"
	type "kubectl operator uninstall cockroachdb --delete-crds --delete-operator-groups"
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
	$1
	[[ "$INTERACTIVE" == "1" ]] && read -p ""
}

run
