#!/bin/bash

# ps aux | grep -E "Goland.*jediterm-bash.in" | grep -v grep | cut -d' ' -f8 | xargs -i kill {}
LOG_DIR="output"
rm -r ${LOG_DIR:?}/*
if [ ! -d $LOG_DIR ]; then
  mkdir $LOG_DIR
fi

# Kill orphans proc
netstat -tlpn 2>/dev/null | grep app | tr -s ' '| cut -d'/' -f1 | cut -d' ' -f7 | xargs -i kill {}

cd test || exit
go build app.go
for i in {0..3} ; do
    ./app "$i" &
    sleep 0.05
done

sleep 10
cd .. && sort -k 3  ${LOG_DIR:?}/*.log  > ${LOG_DIR:?}/completeLog.log
echo -e "(?<host>\w*) (?<clock>.*)\\\n(?<event>.*)\n" > ${LOG_DIR:?}/GoVector/completeGoVectorLog.log
cat ${LOG_DIR:?}/GoVector/*.txt  >> ${LOG_DIR:?}/GoVector/completeGoVectorLog.log
