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

type pdata struct {
	count          int
	request        string
	response       string
	requestPayload string
	requestTime    time.Time
	responseTime   time.Time
	requestSeq     int
	responseSeq    int
}

func main() {
	pcapFile := flag.String("file", "dump.pcap", "TCP Dump file")
	thresh := flag.Int64("threshold", 200, "Microseconds threshold")
	server := flag.String("server", "", "server IP address")
	flag.Parse()

	handle, err = pcap.OpenOffline(*pcapFile)
	if err != nil {
		log.Fatal(err)
	}
	defer handle.Close()

	packets := make(map[uint32]*pdata)

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
					_, ok := packets[tcp.Seq]
					if ok {
						packets[tcp.Seq].responseTime = packet.Metadata().Timestamp
						packets[tcp.Seq].responseSeq = packetCount
						packets[tcp.Seq].count++
					} else {
						packets[tcp.Seq] = &pdata{responseTime: packet.Metadata().Timestamp, responseSeq: packetCount, count: 1}
					}
				}
				if myIP.DstIP.String() == *server {
					_, ok := packets[tcp.Ack]
					if ok {
						packets[tcp.Ack].requestTime = packet.Metadata().Timestamp
						packets[tcp.Ack].requestSeq = packetCount
						packets[tcp.Ack].requestPayload = string(tcp.Payload)
						packets[tcp.Ack].count++
					} else {
						packets[tcp.Ack] = &pdata{requestTime: packet.Metadata().Timestamp, requestSeq: packetCount, requestPayload: string(tcp.Payload), count: 1}
					}
				}
			}

		}
		packetCount++
	}

	for k, pkt := range packets {
		if pkt.requestTime.Unix() > 0 && pkt.responseTime.Unix() > 0 {
			diffMs := pkt.responseTime.Sub(pkt.requestTime).Microseconds()
			if diffMs >= *thresh {
				fmt.Printf("%d -> %d us %d %d request:\n %s\n\n", k, diffMs, pkt.requestSeq, pkt.responseSeq, pkt.requestPayload)
			}
		}
	}

}
