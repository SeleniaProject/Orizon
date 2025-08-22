// Package kernel provides networking stack for OS development
package kernel

import (
	"fmt"
	"net"
	"sync"
	"time"
)

// ============================================================================
// Network Protocol Stack
// ============================================================================

// ProtocolType represents network protocol types
type ProtocolType uint16

const (
	ProtocolIP   ProtocolType = 0x0800
	ProtocolARP  ProtocolType = 0x0806
	ProtocolIPv6 ProtocolType = 0x86DD
	ProtocolTCP  ProtocolType = 6
	ProtocolUDP  ProtocolType = 17
	ProtocolICMP ProtocolType = 1
)

// NetworkInterface represents a network interface
type NetworkInterface struct {
	Name      string
	Index     int
	MAC       [6]byte
	IP        net.IP
	Netmask   net.IPMask
	Gateway   net.IP
	MTU       int
	Flags     uint32
	TxPackets uint64
	RxPackets uint64
	TxBytes   uint64
	RxBytes   uint64
	TxErrors  uint64
	RxErrors  uint64
	mutex     sync.RWMutex
}

// PacketBuffer represents a network packet
type PacketBuffer struct {
	Data      []byte
	Length    int
	Interface *NetworkInterface
	Protocol  ProtocolType
	Timestamp time.Time

	// Layer pointers
	EthHeader  *EthernetHeader
	IPHeader   *IPHeader
	TCPHeader  *TCPHeader
	UDPHeader  *UDPHeader
	ICMPHeader *ICMPHeader
}

// EthernetHeader represents an Ethernet frame header
type EthernetHeader struct {
	DstMAC    [6]byte
	SrcMAC    [6]byte
	EtherType uint16
}

// IPHeader represents an IPv4 header
type IPHeader struct {
	Version    uint8
	IHL        uint8
	TOS        uint8
	Length     uint16
	ID         uint16
	Flags      uint16
	FragOffset uint16
	TTL        uint8
	Protocol   uint8
	Checksum   uint16
	SrcIP      [4]byte
	DstIP      [4]byte
	Options    []byte
}

// TCPHeader represents a TCP header
type TCPHeader struct {
	SrcPort    uint16
	DstPort    uint16
	SeqNum     uint32
	AckNum     uint32
	DataOffset uint8
	Flags      uint8
	Window     uint16
	Checksum   uint16
	UrgPtr     uint16
	Options    []byte
}

// UDPHeader represents a UDP header
type UDPHeader struct {
	SrcPort  uint16
	DstPort  uint16
	Length   uint16
	Checksum uint16
}

// ICMPHeader represents an ICMP header
type ICMPHeader struct {
	Type     uint8
	Code     uint8
	Checksum uint16
	ID       uint16
	Sequence uint16
}

// NetworkStack manages the network protocol stack
type NetworkStack struct {
	interfaces   map[int]*NetworkInterface
	routingTable *RoutingTable
	arpTable     *ARPTable
	socketMgr    *SocketManager
	packetQueue  chan *PacketBuffer
	running      bool
	mutex        sync.RWMutex
}

// GlobalNetworkStack provides global access to networking
var GlobalNetworkStack *NetworkStack

// InitializeNetworkStack initializes the network stack
func InitializeNetworkStack() error {
	if GlobalNetworkStack != nil {
		return fmt.Errorf("network stack already initialized")
	}

	GlobalNetworkStack = &NetworkStack{
		interfaces:   make(map[int]*NetworkInterface),
		routingTable: NewRoutingTable(),
		arpTable:     NewARPTable(),
		socketMgr:    NewSocketManager(),
		packetQueue:  make(chan *PacketBuffer, 1024),
		running:      false,
	}

	// Create loopback interface
	loopback := &NetworkInterface{
		Name:    "lo",
		Index:   1,
		MAC:     [6]byte{0, 0, 0, 0, 0, 0},
		IP:      net.ParseIP("127.0.0.1"),
		Netmask: net.CIDRMask(8, 32),
		MTU:     65536,
		Flags:   0x49, // UP | LOOPBACK | RUNNING
	}

	GlobalNetworkStack.interfaces[1] = loopback

	return nil
}

// StartNetworkStack starts the network processing
func (ns *NetworkStack) Start() error {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	if ns.running {
		return fmt.Errorf("network stack already running")
	}

	ns.running = true
	go ns.packetProcessor()

	return nil
}

// packetProcessor processes incoming packets
func (ns *NetworkStack) packetProcessor() {
	for ns.running {
		select {
		case packet := <-ns.packetQueue:
			ns.processPacket(packet)
		case <-time.After(100 * time.Millisecond):
			// Timeout to check running status
		}
	}
}

// processPacket processes a single packet
func (ns *NetworkStack) processPacket(packet *PacketBuffer) {
	if packet == nil || len(packet.Data) == 0 {
		return
	}

	// Parse Ethernet header
	if len(packet.Data) < 14 {
		return
	}

	packet.EthHeader = &EthernetHeader{
		EtherType: uint16(packet.Data[12])<<8 | uint16(packet.Data[13]),
	}
	copy(packet.EthHeader.DstMAC[:], packet.Data[0:6])
	copy(packet.EthHeader.SrcMAC[:], packet.Data[6:12])

	// Update interface statistics
	if packet.Interface != nil {
		packet.Interface.mutex.Lock()
		packet.Interface.RxPackets++
		packet.Interface.RxBytes += uint64(packet.Length)
		packet.Interface.mutex.Unlock()
	}

	// Process by protocol
	switch ProtocolType(packet.EthHeader.EtherType) {
	case ProtocolIP:
		ns.processIPPacket(packet)
	case ProtocolARP:
		ns.processARPPacket(packet)
	case ProtocolIPv6:
		ns.processIPv6Packet(packet)
	}
}

// processIPPacket processes IPv4 packets
func (ns *NetworkStack) processIPPacket(packet *PacketBuffer) {
	if len(packet.Data) < 34 { // Ethernet + minimum IP header
		return
	}

	ipData := packet.Data[14:] // Skip Ethernet header
	if len(ipData) < 20 {
		return
	}

	packet.IPHeader = &IPHeader{
		Version:  (ipData[0] >> 4) & 0xF,
		IHL:      ipData[0] & 0xF,
		TOS:      ipData[1],
		Length:   uint16(ipData[2])<<8 | uint16(ipData[3]),
		ID:       uint16(ipData[4])<<8 | uint16(ipData[5]),
		Flags:    uint16(ipData[6])<<8 | uint16(ipData[7]),
		TTL:      ipData[8],
		Protocol: ipData[9],
		Checksum: uint16(ipData[10])<<8 | uint16(ipData[11]),
	}

	copy(packet.IPHeader.SrcIP[:], ipData[12:16])
	copy(packet.IPHeader.DstIP[:], ipData[16:20])

	// Process by IP protocol
	switch packet.IPHeader.Protocol {
	case uint8(ProtocolTCP):
		ns.processTCPPacket(packet)
	case uint8(ProtocolUDP):
		ns.processUDPPacket(packet)
	case uint8(ProtocolICMP):
		ns.processICMPPacket(packet)
	}
}

// processTCPPacket processes TCP packets
func (ns *NetworkStack) processTCPPacket(packet *PacketBuffer) {
	headerLen := int(packet.IPHeader.IHL) * 4
	tcpOffset := 14 + headerLen // Ethernet + IP headers

	if len(packet.Data) < tcpOffset+20 {
		return
	}

	tcpData := packet.Data[tcpOffset:]
	packet.TCPHeader = &TCPHeader{
		SrcPort:    uint16(tcpData[0])<<8 | uint16(tcpData[1]),
		DstPort:    uint16(tcpData[2])<<8 | uint16(tcpData[3]),
		SeqNum:     uint32(tcpData[4])<<24 | uint32(tcpData[5])<<16 | uint32(tcpData[6])<<8 | uint32(tcpData[7]),
		AckNum:     uint32(tcpData[8])<<24 | uint32(tcpData[9])<<16 | uint32(tcpData[10])<<8 | uint32(tcpData[11]),
		DataOffset: (tcpData[12] >> 4) & 0xF,
		Flags:      tcpData[13],
		Window:     uint16(tcpData[14])<<8 | uint16(tcpData[15]),
		Checksum:   uint16(tcpData[16])<<8 | uint16(tcpData[17]),
		UrgPtr:     uint16(tcpData[18])<<8 | uint16(tcpData[19]),
	}

	// Deliver to socket manager
	ns.socketMgr.DeliverTCPPacket(packet)
}

// processUDPPacket processes UDP packets
func (ns *NetworkStack) processUDPPacket(packet *PacketBuffer) {
	headerLen := int(packet.IPHeader.IHL) * 4
	udpOffset := 14 + headerLen

	if len(packet.Data) < udpOffset+8 {
		return
	}

	udpData := packet.Data[udpOffset:]
	packet.UDPHeader = &UDPHeader{
		SrcPort:  uint16(udpData[0])<<8 | uint16(udpData[1]),
		DstPort:  uint16(udpData[2])<<8 | uint16(udpData[3]),
		Length:   uint16(udpData[4])<<8 | uint16(udpData[5]),
		Checksum: uint16(udpData[6])<<8 | uint16(udpData[7]),
	}

	// Deliver to socket manager
	ns.socketMgr.DeliverUDPPacket(packet)
}

// processICMPPacket processes ICMP packets
func (ns *NetworkStack) processICMPPacket(packet *PacketBuffer) {
	headerLen := int(packet.IPHeader.IHL) * 4
	icmpOffset := 14 + headerLen

	if len(packet.Data) < icmpOffset+8 {
		return
	}

	icmpData := packet.Data[icmpOffset:]
	packet.ICMPHeader = &ICMPHeader{
		Type:     icmpData[0],
		Code:     icmpData[1],
		Checksum: uint16(icmpData[2])<<8 | uint16(icmpData[3]),
		ID:       uint16(icmpData[4])<<8 | uint16(icmpData[5]),
		Sequence: uint16(icmpData[6])<<8 | uint16(icmpData[7]),
	}

	// Handle ping requests
	if packet.ICMPHeader.Type == 8 { // Echo Request
		ns.sendICMPReply(packet)
	}
}

// processARPPacket processes ARP packets
func (ns *NetworkStack) processARPPacket(packet *PacketBuffer) {
	// Simple ARP processing
	ns.arpTable.ProcessARP(packet)
}

// processIPv6Packet processes IPv6 packets
func (ns *NetworkStack) processIPv6Packet(packet *PacketBuffer) {
	// IPv6 processing would go here
}

// sendICMPReply sends an ICMP echo reply
func (ns *NetworkStack) sendICMPReply(originalPacket *PacketBuffer) {
	// Create reply packet
	replyData := make([]byte, len(originalPacket.Data))
	copy(replyData, originalPacket.Data)

	// Swap IP addresses
	copy(replyData[26:30], originalPacket.IPHeader.SrcIP[:]) // Dst IP
	copy(replyData[30:34], originalPacket.IPHeader.DstIP[:]) // Src IP

	// Change ICMP type to Echo Reply
	headerLen := int(originalPacket.IPHeader.IHL) * 4
	icmpOffset := 14 + headerLen
	replyData[icmpOffset] = 0 // Echo Reply

	// Recalculate checksums (simplified)
	// In a real implementation, proper checksum calculation would be needed

	// Send the packet
	ns.SendPacket(replyData, originalPacket.Interface)
}

// SendPacket sends a packet through the network stack
func (ns *NetworkStack) SendPacket(data []byte, iface *NetworkInterface) error {
	if iface == nil {
		return fmt.Errorf("no interface specified")
	}

	// Update interface statistics
	iface.mutex.Lock()
	iface.TxPackets++
	iface.TxBytes += uint64(len(data))
	iface.mutex.Unlock()

	// In a real implementation, this would send to hardware
	// For now, we simulate successful transmission
	return nil
}

// ============================================================================
// Routing Table
// ============================================================================

// RoutingEntry represents a routing table entry
type RoutingEntry struct {
	Destination net.IPNet
	Gateway     net.IP
	Interface   *NetworkInterface
	Metric      int
}

// RoutingTable manages network routing
type RoutingTable struct {
	entries []RoutingEntry
	mutex   sync.RWMutex
}

// NewRoutingTable creates a new routing table
func NewRoutingTable() *RoutingTable {
	return &RoutingTable{
		entries: make([]RoutingEntry, 0),
	}
}

// AddRoute adds a route to the table
func (rt *RoutingTable) AddRoute(dest net.IPNet, gateway net.IP, iface *NetworkInterface, metric int) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	entry := RoutingEntry{
		Destination: dest,
		Gateway:     gateway,
		Interface:   iface,
		Metric:      metric,
	}

	rt.entries = append(rt.entries, entry)
}

// FindRoute finds the best route for a destination
func (rt *RoutingTable) FindRoute(dest net.IP) *RoutingEntry {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()

	var bestMatch *RoutingEntry
	bestPrefixLen := -1

	for i := range rt.entries {
		entry := &rt.entries[i]
		if entry.Destination.Contains(dest) {
			prefixLen, _ := entry.Destination.Mask.Size()
			if prefixLen > bestPrefixLen {
				bestPrefixLen = prefixLen
				bestMatch = entry
			}
		}
	}

	return bestMatch
}

// ============================================================================
// ARP Table
// ============================================================================

// ARPEntry represents an ARP table entry
type ARPEntry struct {
	IP        net.IP
	MAC       [6]byte
	Interface *NetworkInterface
	Timestamp time.Time
}

// ARPTable manages ARP resolution
type ARPTable struct {
	entries map[string]ARPEntry
	mutex   sync.RWMutex
}

// NewARPTable creates a new ARP table
func NewARPTable() *ARPTable {
	return &ARPTable{
		entries: make(map[string]ARPEntry),
	}
}

// AddEntry adds an ARP entry
func (at *ARPTable) AddEntry(ip net.IP, mac [6]byte, iface *NetworkInterface) {
	at.mutex.Lock()
	defer at.mutex.Unlock()

	entry := ARPEntry{
		IP:        ip,
		MAC:       mac,
		Interface: iface,
		Timestamp: time.Now(),
	}

	at.entries[ip.String()] = entry
}

// LookupMAC looks up MAC address for IP
func (at *ARPTable) LookupMAC(ip net.IP) ([6]byte, bool) {
	at.mutex.RLock()
	defer at.mutex.RUnlock()

	entry, exists := at.entries[ip.String()]
	if !exists {
		return [6]byte{}, false
	}

	// Check if entry is still valid (5 minutes)
	if time.Since(entry.Timestamp) > 5*time.Minute {
		return [6]byte{}, false
	}

	return entry.MAC, true
}

// ProcessARP processes ARP packets
func (at *ARPTable) ProcessARP(packet *PacketBuffer) {
	// Simplified ARP processing
	// In a real implementation, this would parse ARP packets
	// and update the ARP table accordingly
}

// ============================================================================
// Socket Management
// ============================================================================

// SocketType represents socket types
type SocketType int

const (
	SocketTCP SocketType = iota
	SocketUDP
	SocketRaw
)

// SocketState represents socket states
type SocketState int

const (
	SocketClosed SocketState = iota
	SocketListen
	SocketConnected
	SocketConnecting
)

// Socket represents a network socket
type Socket struct {
	ID         uint32
	Type       SocketType
	State      SocketState
	LocalIP    net.IP
	LocalPort  uint16
	RemoteIP   net.IP
	RemotePort uint16
	RecvBuffer []byte
	SendBuffer []byte
	mutex      sync.RWMutex
}

// SocketManager manages network sockets
type SocketManager struct {
	sockets map[uint32]*Socket
	nextID  uint32
	mutex   sync.RWMutex
}

// NewSocketManager creates a new socket manager
func NewSocketManager() *SocketManager {
	return &SocketManager{
		sockets: make(map[uint32]*Socket),
		nextID:  1,
	}
}

// CreateSocket creates a new socket
func (sm *SocketManager) CreateSocket(sockType SocketType) *Socket {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	socket := &Socket{
		ID:         sm.nextID,
		Type:       sockType,
		State:      SocketClosed,
		RecvBuffer: make([]byte, 0, 4096),
		SendBuffer: make([]byte, 0, 4096),
	}

	sm.sockets[sm.nextID] = socket
	sm.nextID++

	return socket
}

// DeliverTCPPacket delivers a TCP packet to the appropriate socket
func (sm *SocketManager) DeliverTCPPacket(packet *PacketBuffer) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	dstIP := net.IP(packet.IPHeader.DstIP[:])
	dstPort := packet.TCPHeader.DstPort

	for _, socket := range sm.sockets {
		if socket.Type == SocketTCP &&
			socket.LocalIP.Equal(dstIP) &&
			socket.LocalPort == dstPort {

			socket.mutex.Lock()
			// Add data to receive buffer
			tcpDataOffset := 14 + int(packet.IPHeader.IHL)*4 + int(packet.TCPHeader.DataOffset)*4
			if tcpDataOffset < len(packet.Data) {
				data := packet.Data[tcpDataOffset:]
				socket.RecvBuffer = append(socket.RecvBuffer, data...)
			}
			socket.mutex.Unlock()
			break
		}
	}
}

// DeliverUDPPacket delivers a UDP packet to the appropriate socket
func (sm *SocketManager) DeliverUDPPacket(packet *PacketBuffer) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	dstIP := net.IP(packet.IPHeader.DstIP[:])
	dstPort := packet.UDPHeader.DstPort

	for _, socket := range sm.sockets {
		if socket.Type == SocketUDP &&
			socket.LocalIP.Equal(dstIP) &&
			socket.LocalPort == dstPort {

			socket.mutex.Lock()
			// Add data to receive buffer
			udpDataOffset := 14 + int(packet.IPHeader.IHL)*4 + 8
			if udpDataOffset < len(packet.Data) {
				data := packet.Data[udpDataOffset:]
				socket.RecvBuffer = append(socket.RecvBuffer, data...)
			}
			socket.mutex.Unlock()
			break
		}
	}
}

// ============================================================================
// Network Driver Interface
// ============================================================================

// NetworkDriver represents a network device driver
type NetworkDriver interface {
	Initialize() error
	SendPacket(data []byte) error
	ReceivePacket() ([]byte, error)
	GetMAC() [6]byte
	GetMTU() int
}

// RegisterNetworkInterface registers a network interface
func (ns *NetworkStack) RegisterNetworkInterface(name string, driver NetworkDriver) (*NetworkInterface, error) {
	ns.mutex.Lock()
	defer ns.mutex.Unlock()

	err := driver.Initialize()
	if err != nil {
		return nil, err
	}

	index := len(ns.interfaces) + 1
	iface := &NetworkInterface{
		Name:  name,
		Index: index,
		MAC:   driver.GetMAC(),
		MTU:   driver.GetMTU(),
		Flags: 0x43, // UP | BROADCAST | RUNNING
	}

	ns.interfaces[index] = iface

	// Start receiving packets from this interface
	go ns.interfaceReceiver(iface, driver)

	return iface, nil
}

// interfaceReceiver receives packets from a network interface
func (ns *NetworkStack) interfaceReceiver(iface *NetworkInterface, driver NetworkDriver) {
	for ns.running {
		data, err := driver.ReceivePacket()
		if err != nil {
			time.Sleep(10 * time.Millisecond)
			continue
		}

		if len(data) > 0 {
			packet := &PacketBuffer{
				Data:      data,
				Length:    len(data),
				Interface: iface,
				Timestamp: time.Now(),
			}

			select {
			case ns.packetQueue <- packet:
			default:
				// Queue full, drop packet
				iface.mutex.Lock()
				iface.RxErrors++
				iface.mutex.Unlock()
			}
		}
	}
}

// ============================================================================
// Kernel API functions for networking
// ============================================================================

// KernelCreateSocket creates a network socket
func KernelCreateSocket(sockType int) uint32 {
	if GlobalNetworkStack == nil {
		return 0
	}

	socket := GlobalNetworkStack.socketMgr.CreateSocket(SocketType(sockType))
	return socket.ID
}

// KernelBindSocket binds a socket to an address
func KernelBindSocket(socketID uint32, ip [4]byte, port uint16) bool {
	if GlobalNetworkStack == nil {
		return false
	}

	GlobalNetworkStack.socketMgr.mutex.RLock()
	socket, exists := GlobalNetworkStack.socketMgr.sockets[socketID]
	GlobalNetworkStack.socketMgr.mutex.RUnlock()

	if !exists {
		return false
	}

	socket.mutex.Lock()
	socket.LocalIP = net.IP(ip[:])
	socket.LocalPort = port
	socket.mutex.Unlock()

	return true
}

// KernelSendData sends data through a socket
func KernelSendData(socketID uint32, data []byte) int {
	if GlobalNetworkStack == nil {
		return -1
	}

	GlobalNetworkStack.socketMgr.mutex.RLock()
	socket, exists := GlobalNetworkStack.socketMgr.sockets[socketID]
	GlobalNetworkStack.socketMgr.mutex.RUnlock()

	if !exists {
		return -1
	}

	socket.mutex.Lock()
	socket.SendBuffer = append(socket.SendBuffer, data...)
	sent := len(data)
	socket.mutex.Unlock()

	// In a real implementation, this would actually send the data
	return sent
}

// KernelReceiveData receives data from a socket
func KernelReceiveData(socketID uint32, buffer []byte) int {
	if GlobalNetworkStack == nil {
		return -1
	}

	GlobalNetworkStack.socketMgr.mutex.RLock()
	socket, exists := GlobalNetworkStack.socketMgr.sockets[socketID]
	GlobalNetworkStack.socketMgr.mutex.RUnlock()

	if !exists {
		return -1
	}

	socket.mutex.Lock()
	defer socket.mutex.Unlock()

	if len(socket.RecvBuffer) == 0 {
		return 0
	}

	copyLen := len(buffer)
	if copyLen > len(socket.RecvBuffer) {
		copyLen = len(socket.RecvBuffer)
	}

	copy(buffer, socket.RecvBuffer[:copyLen])
	socket.RecvBuffer = socket.RecvBuffer[copyLen:]

	return copyLen
}

// KernelGetNetworkStats returns network interface statistics
func KernelGetNetworkStats(interfaceIndex int) (txPackets, rxPackets, txBytes, rxBytes, txErrors, rxErrors uint64) {
	if GlobalNetworkStack == nil {
		return 0, 0, 0, 0, 0, 0
	}

	GlobalNetworkStack.mutex.RLock()
	iface, exists := GlobalNetworkStack.interfaces[interfaceIndex]
	GlobalNetworkStack.mutex.RUnlock()

	if !exists {
		return 0, 0, 0, 0, 0, 0
	}

	iface.mutex.RLock()
	defer iface.mutex.RUnlock()

	return iface.TxPackets, iface.RxPackets, iface.TxBytes, iface.RxBytes, iface.TxErrors, iface.RxErrors
}
