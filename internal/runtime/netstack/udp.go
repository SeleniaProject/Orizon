package netstack

import (
    "net"
    "time"
)

// UDPEndpoint provides simple UDP send/recv helpers.
type UDPEndpoint struct {
    conn *net.UDPConn
}

func ListenUDP(addr string) (*UDPEndpoint, error) {
    a, err := net.ResolveUDPAddr("udp", addr)
    if err != nil { return nil, err }
    c, err := net.ListenUDP("udp", a)
    if err != nil { return nil, err }
    return &UDPEndpoint{conn: c}, nil
}

func DialUDP(addr string) (*UDPEndpoint, error) {
    r, err := net.ResolveUDPAddr("udp", addr)
    if err != nil { return nil, err }
    c, err := net.DialUDP("udp", nil, r)
    if err != nil { return nil, err }
    return &UDPEndpoint{conn: c}, nil
}

func (e *UDPEndpoint) Close() error { return e.conn.Close() }

func (e *UDPEndpoint) SetDeadline(t time.Time) error      { return e.conn.SetDeadline(t) }
func (e *UDPEndpoint) SetReadDeadline(t time.Time) error  { return e.conn.SetReadDeadline(t) }
func (e *UDPEndpoint) SetWriteDeadline(t time.Time) error { return e.conn.SetWriteDeadline(t) }

func (e *UDPEndpoint) ReadFrom(b []byte) (int, *net.UDPAddr, error) {
    n, addr, err := e.conn.ReadFromUDP(b)
    return n, addr, err
}

func (e *UDPEndpoint) WriteTo(b []byte, addr *net.UDPAddr) (int, error) {
    return e.conn.WriteToUDP(b, addr)
}


