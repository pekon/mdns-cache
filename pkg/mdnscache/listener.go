package mdnscache

import (
	"context"
	"log"
	"net"
	"syscall"

	"github.com/miekg/dns"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
	"golang.org/x/sys/unix"
)

var (
	ipv4addr = &net.UDPAddr{
		IP:   net.IPv4(224, 0, 0, 251),
		Port: 5353,
	}
	ipv6addr = &net.UDPAddr{
		IP:   net.ParseIP("ff02::fb"),
		Port: 5353,
	}
)

func reusePort(network, address string, conn syscall.RawConn) error {
	var opErr error
	err := conn.Control(func(fd uintptr) {
		opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEPORT, 1)
		if opErr != nil {
			return
		}
		opErr = unix.SetsockoptInt(int(fd), unix.SOL_SOCKET, unix.SO_REUSEADDR, 1)

	})
	if err != nil {
		return err
	}
	return opErr
}

type Listener struct {
	closed       bool
	conn, p6, p4 net.PacketConn
}

func (l *Listener) Close() {
	if l.p6 != nil {
		l.p6.Close()
	}
	if l.p4 != nil {
		l.p4.Close()
	}
	if l.conn != nil {
		l.conn.Close()
	}
}

func NewListener(iface string) (*Listener, error) {
	ifc, err := net.InterfaceByName(iface)
	if err != nil {
		return nil, err
	}

	cfg := &net.ListenConfig{Control: reusePort}
	l := &Listener{}

	conn, err := cfg.ListenPacket(context.TODO(), "udp", "0.0.0.0:5353") // mDNS over UDP
	if err != nil {
		return nil, err
	}
	p4 := ipv4.NewPacketConn(conn)
	if err := p4.JoinGroup(ifc, ipv4addr); err != nil {
		return nil, err
	}
	p6 := ipv6.NewPacketConn(conn)
	if err := p6.JoinGroup(ifc, ipv6addr); err != nil {
		return nil, err
	}

	return l, nil
}

type Msg struct {
	dns.Msg
	Addr net.Addr
}

func nextMsg(pc net.PacketConn, out chan<- *Msg) {
	buf := make([]byte, 1600)
	n, addr, err := pc.ReadFrom(buf)
	msg := Msg{Addr: addr}
	if err := msg.Unpack(buf[:n]); err != nil {
		log.Printf("msg.Unpack: %v", err)
		return
	} else {
		out <- &msg
	}
	if err != nil {
		log.Printf("nextMsg: %v", err)
	}
}

func (l *Listener) Listen(ctx context.Context, out chan<- *Msg) {
	go func() {
		<-ctx.Done()
		l.Close()
	}()

	go func() {
		for !l.closed {
			nextMsg(l.p4, out)
		}
	}()
	for !l.closed {
		nextMsg(l.p6, out)
	}

}
