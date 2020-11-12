#!/bin/bash

./agent --hostname=10.254.24.2  --bgpconfigfile=server.toml --configfile=config.json

gobgpd -f server.toml -d -r