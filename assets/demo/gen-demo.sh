#!/bin/bash

set -o errexit
set -o pipefail

INTERACTIVE=0 asciinema rec --overwrite -c ./assets/demo/demo.sh ./assets/demo/demo.asciinema.json
asciicast2gif -w 90 -h 33 ./assets/demo/demo.asciinema.json ./assets/demo/demo.gif
rm ./assets/demo/demo.asciinema.json
