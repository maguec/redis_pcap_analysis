package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/google/gopacket/pcap"
)

var (
	handle *pcap.Handle
	err    error
)

type pktinfo struct {
	request      string
	response     string
	requestTime  time.Time
	responseTime time.Time
}

type pdata struct {
	count int
	pkts  []pktinfo
}

func main() {
	pcapFile := flag.String("file", "dump.pcap", "TCP Dump file")
	client := flag.String("client", "", "client IP address")
	server := flag.String("server", "", "server IP address")
	flag.Parse()

	handle, err = pcap.OpenOffline(*pcapFile)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	//	packets := make(map[uint32]*pdata)

	// Set filter
	var filter string = "tcp"
	err = handle.SetBPFFilter(filter)
	if err != nil {
		log.Fatal(err)
	}

	packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

	packetCount := 1

	for packet := range packetSource.Packets() {

		if tcpLayer := packet.Layer(layers.LayerTypeTCP); tcpLayer != nil {
			tcp, _ := tcpLayer.(*layers.TCP)
			if tcp.ACK && tcp.PSH {
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				myIP := ipLayer.(*layers.IPv4)
				if myIP.SrcIP.String() == *server {
					fmt.Printf("FromServer => Ack %d\n", tcp.Ack)
				}
				if myIP.SrcIP.String() == *client {
					fmt.Printf("FromClient => Seq %d\n", tcp.Seq)
				}
			}

		}
		packetCount++
	}
}
