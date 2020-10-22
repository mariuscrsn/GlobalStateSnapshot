#!/bin/bash

# Make log directories
LOG_DIR="output"
rm -r ${LOG_DIR:?}/*
if [ ! -d $LOG_DIR ]; then
  mkdir $LOG_DIR
fi

# Kill orphans proc
netstat -tlpn 2>/dev/null | grep App | grep 160 | tr -s ' '| cut -d'/' -f1 | cut -d' ' -f7 | xargs -i kill {}

/usr/local/go/bin/go test -v globalSnapshot_test.go

# Merge logs
sort -k 3  ${LOG_DIR:?}/*.log  > ${LOG_DIR:?}/completeLog.log
echo -e "(?<host>\w*) (?<clock>.*)\\\n(?<event>.*)\n" > ${LOG_DIR:?}/GoVector/completeGoVectorLog.log
cat ${LOG_DIR:?}/GoVector/LogFileP*.txt  >> ${LOG_DIR:?}/GoVector/completeGoVectorLog.log
