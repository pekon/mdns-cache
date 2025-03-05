package main

import (
	"context"
	"golang.org/x/net/dns/dnsmessage"
	"golang.org/x/net/ipv4"
	"log"
	"net"
	"syscall"
	// "x/sys/unix"
)

func reusePort(network, address string, conn syscall.RawConn) error {
	return conn.Control(func(descriptor uintptr) {
		syscall.SetsockoptInt(int(descriptor), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	})
}

func main() {
	cfg := &net.ListenConfig{Control: reusePort}

	c, err := cfg.ListenPacket(context.Background(), "udp4", "0.0.0.0:5353") // mDNS over UDP
	if err != nil {
		log.Fatal(err)
	}
	defer c.Close()
	p := ipv4.NewPacketConn(c)

	en0, err := net.InterfaceByName("enp8s0")
	if err != nil {
		log.Fatal(err)
	}
	mgmt, err := net.InterfaceByName("mgmt")
	if err != nil {
		log.Fatal(err)
	}
	mDNSLinkLocal := net.UDPAddr{IP: net.IPv4(224, 0, 0, 251)}
	if err := p.JoinGroup(en0, &mDNSLinkLocal); err != nil {
		log.Fatal(err)
	}
	defer p.LeaveGroup(en0, &mDNSLinkLocal)
	if err := p.JoinGroup(mgmt, &mDNSLinkLocal); err != nil {
		log.Fatal(err)
	}
	defer p.LeaveGroup(mgmt, &mDNSLinkLocal)
	if err := p.SetControlMessage(ipv4.FlagDst, true); err != nil {
		log.Fatal(err)
	}

	b := make([]byte, 1500)
	for {
		n, cm, peer, err := p.ReadFrom(b)
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("%d %v %v", n, cm, peer)
		var p dnsmessage.Parser
		if h, err := p.Start(b); err != nil {
			log.Printf("Parse err: %v", err)
			continue
		} else {
			log.Printf("header: %v", h.GoString())
		}
		for {
			q, err := p.Question()
			if err == dnsmessage.ErrSectionDone {
				break
			}
                        if err != nil {
				log.Printf("Question err: %v", err)
				continue
			}
			log.Printf("Question: %v", q)
		}
		for {
			h, err := p.AnswerHeader()
			if err == dnsmessage.ErrSectionDone {
				break
			}
                        if err != nil {
				log.Printf("Answer header err: %v", err)
				continue
			}
			log.Printf("ah: %v", h.GoString())
			if err := p.SkipAnswer(); err != nil {
				log.Printf("se: %v", err)
			}
		}
		
/*
		if !cm.Dst.IsMulticast() || !cm.Dst.Equal(mDNSLinkLocal.IP) {
			continue
		}
		answers := []byte("FAKE-MDNS-ANSWERS") // fake mDNS answers, you need to implement this
		if _, err := p.WriteTo(answers, nil, peer); err != nil {
			log.Fatal(err)
		}
*/
	}
}
