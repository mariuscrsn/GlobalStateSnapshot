#!/bin/bash

# ps aux | grep -E "Goland.*jediterm-bash.in" | grep -v grep | cut -d' ' -f8 | xargs -i kill {}
LOG_DIR="output"
if [ ! -d $LOG_DIR ]; then
  mkdir $LOG_DIR
fi

# Kill orphans proc
netstat -tlpn 2>/dev/null | grep app | tr -s ' '| cut -d'/' -f1 | cut -d' ' -f7 | xargs -i kill {}
rm output/*
cd test || exit
go build app.go
for i in {0..3} ; do
    ./app "$i" &
    sleep 0.05
done

sleep 10
cd .. && sort -k 3  output/*  > output/completeLog.log

