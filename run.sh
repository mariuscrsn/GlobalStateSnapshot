#!/bin/bash

# ps aux | grep -E "Goland.*jediterm-bash.in" | grep -v grep | cut -d' ' -f8 | xargs -i kill {}
LOG_DIR="output"
if [ ! -d $LOG_DIR ]; then
  mkdir $LOG_DIR
fi

# Kill orphans proc
netstat -tlpn 2>/dev/null | grep globalState | tr -s ' '| cut -d'/' -f1 | cut -d' ' -f7 | xargs -i kill {}
cd src || exit
go build globalState.go
for i in {0..3} ; do
    ./globalState "$i" &
    sleep 0.05
done

