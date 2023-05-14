#! /bin/sh
#./start.sh eth2 9738 x.x.x.x
kill -9 `pidof zhuabao`
kill -9 `pidof tcpdump`
ps -ef|grep tcpdump
nohup ./zhuabao tcpdump -i $1 tcp port $2 and host $3  -s64 -c 300000&