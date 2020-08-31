#!/bin/sh

nix-shell -p cacert cachix curl jq nix --run "sh startup_script.sh"