```
GOOS=linux go build client.go
GOOS=linux go build server.go

./drop.sh 就是netfilter drop client 和 server 的连接数据，模拟底层网络不通的情况下，SetReadDeadline是否正常工作
./flush.sh 就删除iptables 规则, 模拟网络恢复。
```