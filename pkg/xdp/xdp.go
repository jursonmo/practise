package xdp

//go:generate bpf2go -target bpf xdp ../../cmd/xdp_learn/ebpf/xconnect.c -- -I/root/goworkspace/practise/cmd/xdp_learn/ebpf/include -O2 -Wall

// func aa() {
// 	specs, err := newXdpSpecs()
// 	if err != nil {
// 		return nil, err
// 	} //go run github.com/cilium/ebpf/cmd/

// 	_, err := specs.Load(nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("Can't load objects:%s", err)
// 	}
// }
