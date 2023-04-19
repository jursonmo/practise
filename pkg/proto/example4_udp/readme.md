
用了高效的udp 库，用上linux sendmsgs 批量读写，那么测试时，需要编译成linux 平台
GOOS=linux go build -o server server.go auth.go
GOOS=linux go build -o client client.go auth.go

test handshake (ping pong)
test auth ( use proto options)
test ReadDeadline