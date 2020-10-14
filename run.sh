#!/bin/bash

# ps aux | grep -E "Goland.*jediterm-bash.in" | grep -v grep | cut -d' ' -f8 | xargs -i kill {}

cd src || exit
go build globalState.go
for i in {0..3} ; do
    ./globalState "$i" &
    sleep 0.05
done

