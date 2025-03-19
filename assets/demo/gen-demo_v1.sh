#!/bin/bash

set -o errexit
set -o pipefail

INTERACTIVE=0 asciinema rec --overwrite -c ./assets/demo/demo_v1.sh ./assets/demo/demo.v1.asciinema.json
asciicast2gif -w 102 -h 34 ./assets/demo/demo.v1.asciinema.json ./assets/demo/demo-v1.gif
rm ./assets/demo/demo.v1.asciinema.json
