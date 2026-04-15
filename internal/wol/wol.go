// Package wol implements Wake-on-LAN magic packet transmission.
// A magic packet is 6 bytes of 0xFF followed by the target MAC address
// repeated 16 times (102 bytes total), sent as a UDP broadcast.
package wol

import (
	"fmt"
	"net"
)

// Send transmits a WoL magic packet to macAddr via UDP broadcast.
// broadcastIP can be a directed broadcast (e.g. "192.168.1.255") or
// the limited broadcast "255.255.255.255".
// port is typically 7 or 9.
// repeat controls how many times the packet is sent (UDP is unreliable).
func Send(macAddr, broadcastIP string, port, repeat int) error {
	mac, err := net.ParseMAC(macAddr)
	if err != nil {
		return fmt.Errorf("invalid MAC address %q: %w", macAddr, err)
	}

	packet := buildMagicPacket(mac)

	addr := &net.UDPAddr{
		IP:   net.ParseIP(broadcastIP),
		Port: port,
	}
	if addr.IP == nil {
		return fmt.Errorf("invalid broadcast IP %q", broadcastIP)
	}

	conn, err := net.DialUDP("udp", nil, addr)
	if err != nil {
		return fmt.Errorf("open UDP socket: %w", err)
	}
	defer conn.Close()

	var lastErr error
	for i := 0; i < repeat; i++ {
		if _, err := conn.Write(packet); err != nil {
			lastErr = err
		}
	}
	return lastErr
}

// buildMagicPacket constructs the 102-byte WoL magic packet for mac.
func buildMagicPacket(mac net.HardwareAddr) []byte {
	packet := make([]byte, 6+16*6)
	// Synchronisation stream: 6 × 0xFF
	for i := 0; i < 6; i++ {
		packet[i] = 0xFF
	}
	// Target MAC repeated 16 times
	for i := 0; i < 16; i++ {
		copy(packet[6+i*6:], mac)
	}
	return packet
}
