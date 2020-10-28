#!/bin/bash

# Make log directories
LOG_DIR="output"
if [ ! -d $LOG_DIR ]; then
  mkdir $LOG_DIR
fi

rm -r ${LOG_DIR:?}/*

# Kill orphans proc
netstat -tlpn 2>/dev/null | grep App | grep 160 | tr -s ' '| cut -d'/' -f1 | cut -d' ' -f7 | xargs -i kill {}

/usr/local/go/bin/go test -v globalSnapshot_test.go

# Merge & sort logs
#sort -k 3  ${LOG_DIR:?}/*.log  > ${LOG_DIR:?}/completeLog.log
cat ${LOG_DIR:?}/*.log  > ${LOG_DIR:?}/completeLog.log
cat ${LOG_DIR:?}/GoVector/LogFile*  >> ${LOG_DIR:?}/GoVector/temp_log.log
echo -e "(?<date>\d+) (?<host>\w*) (?<clock>.*)\\\n(?<event>.*)\n" > ${LOG_DIR:?}/GoVector/completeGoVectorLog.log
sed -n '/^.*}$/ {N; s/\n/$\^$/g; p;}' ${LOG_DIR:?}/GoVector/temp_log.log | sort | sed -n 's/\$\^\$/\n/g; p;' >> ${LOG_DIR:?}/GoVector/completeGoVectorLog.log
rm ${LOG_DIR:?}/GoVector/temp_log.log
