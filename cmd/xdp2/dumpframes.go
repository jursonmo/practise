// Copyright 2019 Asavie Technologies Ltd. All rights reserved.
//
// Use of this source code is governed by a BSD-style license
// that can be found in the LICENSE file in the root of the source
// tree.

/*
dumpframes demostrates how to receive frames from a network link using
github.com/asavie/xdp package, it sets up an XDP socket attached to a
particular network link and dumps all frames it receives to standard output.
*/
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"

	"github.com/asavie/xdp"
	"github.com/asavie/xdp/examples/dumpframes/ebpf"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
)

func main() {
	var linkName string
	var queueID int
	var protocol int64

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	flag.StringVar(&linkName, "linkname", "enp3s0", "The network link on which rebroadcast should run on.")
	flag.IntVar(&queueID, "queueid", 0, "The ID of the Rx queue to which to attach to on the network link.")
	flag.Int64Var(&protocol, "ip-proto", 0, "If greater than 0 and less than or equal to 255, limit xdp bpf_redirect_map to packets with the specified IP protocol number.")
	flag.Parse()

	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("error: failed to fetch the list of network interfaces on the system: %v\n", err)
		return
	}

	Ifindex := -1
	for _, iface := range interfaces {
		if iface.Name == linkName {
			Ifindex = iface.Index
			break
		}
	}
	if Ifindex == -1 {
		fmt.Printf("error: couldn't find a suitable network interface to attach to\n")
		return
	}

	var program *xdp.Program

	// Create a new XDP eBPF program and attach it to our chosen network link.
	if protocol == 0 {
		program, err = xdp.NewProgram(queueID + 1)
	} else {
		program, err = ebpf.NewIPProtoProgram(uint32(protocol), nil)
	}
	if err != nil {
		fmt.Printf("error: failed to create xdp program: %v\n", err)
		return
	}
	defer program.Close()
	if err := program.Attach(Ifindex); err != nil {
		fmt.Printf("error: failed to attach xdp program to interface: %v\n", err)
		return
	}
	defer program.Detach(Ifindex)

	// Create and initialize an XDP socket attached to our chosen network
	// link.
	xsk, err := xdp.NewSocket(Ifindex, queueID, nil)
	if err != nil {
		fmt.Printf("error: failed to create an XDP socket: %v\n", err)
		return
	}

	// Register our XDP socket file descriptor with the eBPF program so it can be redirected packets
	if err := program.Register(queueID, xsk.FD()); err != nil {
		fmt.Printf("error: failed to register socket in BPF map: %v\n", err)
		return
	}
	defer program.Unregister(queueID)

	for {
		// If there are any free slots on the Fill queue...
		if n := xsk.NumFreeFillSlots(); n > 0 {
			// ...then fetch up to that number of not-in-use
			// descriptors and push them onto the Fill ring queue
			// for the kernel to fill them with the received
			// frames.
			xsk.Fill(xsk.GetDescs(n))
		}

		// Wait for receive - meaning the kernel has
		// produced one or more descriptors filled with a received
		// frame onto the Rx ring queue.
		log.Printf("waiting for frame(s) to be received...")
		numRx, _, err := xsk.Poll(-1)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		if numRx > 0 {
			// Consume the descriptors filled with received frames
			// from the Rx ring queue.
			rxDescs := xsk.Receive(numRx)

			// Print the received frames and also modify them
			// in-place replacing the destination MAC address with
			// broadcast address.
			for i := 0; i < len(rxDescs); i++ {
				pktData := xsk.GetFrame(rxDescs[i])
				pkt := gopacket.NewPacket(pktData, layers.LayerTypeEthernet, gopacket.Default)
				log.Printf("received frame:\n%s%+v", hex.Dump(pktData[:]), pkt)
			}
		}
	}
}

/*
root@ubuntu:~# ./dumpframes -linkname="eth0" -ip-proto=1
2022/01/23 15:55:14.191706 waiting for frame(s) to be received...
2022/01/23 15:55:14.383184 received frame:
00000000  00 e2 69 3d fb 20 00 90  27 f0 7e 64 08 00 45 00  |..i=. ..'.~d..E.|
00000010  00 54 16 98 00 00 40 01  d9 5c c0 a8 04 fd c0 a8  |.T....@..\......|
00000020  04 67 00 00 ca fc 18 48  00 24 16 cc d7 24 28 6f  |.g.....H.$...$(o|
00000030  ee 1e 01 01 01 01 01 01  01 01 01 01 01 01 01 01  |................|
00000040  01 01 01 01 01 01 01 01  01 01 01 01 01 01 01 01  |................|
00000050  01 01 01 01 01 01 01 01  01 01 01 01 01 01 01 01  |................|
00000060  01 01                                             |..|
PACKET: 98 bytes
- Layer 1 (14 bytes) = Ethernet	{Contents=[..14..] Payload=[..84..] SrcMAC=00:90:27:f0:7e:64 DstMAC=00:e2:69:3d:fb:20 EthernetType=IPv4 Length=0}
- Layer 2 (20 bytes) = IPv4	{Contents=[..20..] Payload=[..64..] Version=4 IHL=5 TOS=0 Length=84 Id=5784 Flags= FragOffset=0 TTL=64 Protocol=ICMPv4 Checksum=55644 SrcIP=192.168.4.253 DstIP=192.168.4.103 Options=[] Padding=[]}
- Layer 3 (08 bytes) = ICMPv4	{Contents=[..8..] Payload=[..56..] TypeCode=EchoReply Checksum=51964 Id=6216 Seq=36}
- Layer 4 (56 bytes) = Payload	56 byte(s)
2022/01/23 15:55:14.383294 waiting for frame(s) to be received...
*/
