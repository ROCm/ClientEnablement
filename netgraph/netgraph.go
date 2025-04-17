package main

import (
    "context"
    "encoding/binary"
    "encoding/json"
    "flag"
    "fmt"
    "io"
    "log"
    "net"
    "os"
    "os/signal"
    "sync"
    "syscall"
    "time"

    "github.com/gopacket/gopacket"
    "github.com/gopacket/gopacket/layers"
    "github.com/gopacket/gopacket/pcap"
)

// debug enables extra logging for development/troubleshooting.
const debug = true

// Constants for EtherTypes:
const (
    lldpEtherType = 0x88CC
    cdpEtherType  = 0x2000
    arpEtherType  = 0x0806
)

// LLDP TLV type constants (minimal subset).
const (
    lldpTLVTypeEnd        = 0
    lldpTLVTypeChassisID  = 1
    lldpTLVTypePortID     = 2
    lldpTLVTypeTTL        = 3
    lldpTLVTypePortDesc   = 4
    lldpTLVTypeSystemName = 5
    // … add more if needed
)

// NeighborInfo holds basic info for discovered neighbors (for ARP/CDP).
type NeighborInfo struct {
    InterfaceName string
    SourceMAC     string
    Protocol      string
    Details       string // Could store more structured info
}

// Node represents one end of a link, e.g. (device=switch1, interface=Eth0/1, mac=aa:bb:cc...).
type Node struct {
    Device    string `json:"device"`
    Interface string `json:"interface"`
    MAC       string `json:"mac,omitempty"`
    // You could add more fields if needed (e.g., VLAN, IP, etc.)
}

// Edge links two Nodes (Local -> Remote).
type Edge struct {
    Local  Node `json:"local"`
    Remote Node `json:"remote"`
}

var (
    // edges holds discovered LLDP edges in a global slice.
    edges   []Edge
    edgesMu sync.Mutex

    // discoveredNeighbors is used for ARP/CDP logging.
    discoveredNeighbors sync.Map

    // localHostname caches our local hostname once for clarity.
    localHostname string
)

func main() {
    // Use flags for a user-specified duration and output file.
    outputFile := flag.String("out", "", "Output JSON file for edges")
    captureDuration := flag.Int("duration", 0, "Capture time in seconds (0 means run until Ctrl+C)")

    flag.Parse()

    // Attempt to get local hostname (defaults if fails).
    h, err := os.Hostname()
    if err != nil {
        h = "UnknownHost"
    }
    localHostname = h

    // Find all network devices.
    devices, err := pcap.FindAllDevs()
    if err != nil {
        log.Fatalf("Error finding devices: %v", err)
    }
    if len(devices) == 0 {
        log.Println("No devices found. Exiting.")
        return
    }

    // We want to capture LLDP (0x88cc), CDP (0x2000), and ARP (0x0806).
    // The BPF filter uses "or" for multiple EtherTypes.
    filter := "ether proto 0x88cc or ether proto 0x2000 or ether proto 0x0806"

    // Create a context that cancels on SIGINT/SIGTERM.
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Also, if captureDuration is set, stop after that many seconds.
    if *captureDuration > 0 {
        time.AfterFunc(time.Duration(*captureDuration)*time.Second, func() {
            log.Printf("Capture time (%d seconds) is up, stopping...\n", *captureDuration)
            cancel()
        })
    }

    // Listen for OS interrupt/kill signals to gracefully stop.
    go func() {
        sigChan := make(chan os.Signal, 1)
        signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
        <-sigChan
        log.Println("Received interrupt signal, stopping captures...")
        cancel()
    }()

    // Start capturing from all devices in parallel.
    // We'll close them gracefully once the context is cancelled.
    var wg sync.WaitGroup
    for _, dev := range devices {
        wg.Add(1)
        go func(d pcap.Interface) {
            defer wg.Done()
            capturePackets(ctx, d.Name, filter)
        }(dev)
    }

    // Let the user know how to stop or how long we run if *captureDuration>0.
    if *captureDuration > 0 {
        log.Printf("Capturing for %d seconds...\n", *captureDuration)
    } else {
        log.Println("Capturing until Ctrl+C...")
    }

    // Wait until context is done.
    <-ctx.Done()

    // Wait for all goroutines to exit cleanly.
    wg.Wait()

    // Print discovered neighbors for ARP/CDP
    fmt.Println("\nDiscovered Neighbors (ARP & CDP):")
    discoveredNeighbors.Range(func(key, value interface{}) bool {
        neighbor := value.(NeighborInfo)
        fmt.Printf("  Key: %s, Interface: %s, SrcMAC: %s, Protocol: %s, Details: %s\n",
            key, neighbor.InterfaceName, neighbor.SourceMAC, neighbor.Protocol, neighbor.Details)
        return true
    })
    fmt.Println()

    // Print discovered LLDP edges in text form:
    fmt.Println("Discovered LLDP Edges:")
    edgesMu.Lock()
    for _, e := range edges {
        // Just skip printing self-loops, if any
        if e.Local.Device != e.Remote.Device {
            fmt.Printf("  (%s, %s) -> (%s, %s)\n",
                e.Local.Device, e.Local.Interface,
                e.Remote.Device, e.Remote.Interface)
        }
    }
    edgesMu.Unlock()
    fmt.Println()

    // Also output edges in JSON form (to file or stdout).
    edgesMu.Lock()
    jsonData, err := json.MarshalIndent(edges, "", "  ")
    edgesMu.Unlock()
    if err != nil {
        log.Printf("Error marshaling edges to JSON: %v\n", err)
        return
    }

    if *outputFile != "" {
        if err := os.WriteFile(*outputFile, jsonData, 0644); err != nil {
            log.Printf("Error writing JSON to file '%s': %v\n", *outputFile, err)
        } else {
            fmt.Printf("Wrote LLDP edges JSON to %s\n", *outputFile)
        }
    } else {
        fmt.Println("LLDP Edges in JSON:")
        fmt.Println(string(jsonData))
    }
}

// capturePackets opens a pcap handle on the given interface, applies the BPF filter,
// and reads packets until the context is cancelled or an error occurs.
func capturePackets(ctx context.Context, deviceName, filter string) {
    // Use a short read timeout (1 second). This ensures we can periodically check the context and exit.
    handle, err := pcap.OpenLive(deviceName, 65535, true, 1*time.Second)
    if err != nil {
        log.Printf("pcap OpenLive failed on %s: %v", deviceName, err)
        return
    }
    defer handle.Close()

    if err := handle.SetBPFFilter(filter); err != nil {
        log.Printf("SetBPFFilter failed on %s: %v", deviceName, err)
        return
    }
    log.Printf("Capturing on interface %s with filter (%s)\n", deviceName, filter)

    packetSource := gopacket.NewPacketSource(handle, handle.LinkType())

    for {
        select {
        case <-ctx.Done():
            return
        default:
            // Use NextPacket() to handle timeouts.
            packet, err := packetSource.NextPacket()
            if err != nil {
                if err == io.EOF {
                    // No more packets (interface closed?), just exit.
                    return
                }
                if err == pcap.NextErrorTimeoutExpired {
                    // Timeout - check if context is done.
                    if ctx.Err() != nil {
                        return
                    }
                    // Otherwise continue to next iteration.
                    continue
                }
                // Some other error.
                log.Printf("Error reading packet on %s: %v", deviceName, err)
                return
            }
            // Got a valid packet
            if debug {
                log.Printf("NETGRAPH: got packet on %s (len=%d)\n",
                    deviceName, len(packet.Data()))
            }
            processPacket(deviceName, packet)
        }
    }
}

// processPacket routes packets to the right handler based on EtherType.
func processPacket(deviceName string, packet gopacket.Packet) {
    ethLayer := packet.Layer(layers.LayerTypeEthernet)
    if ethLayer == nil {
        return
    }
    eth, _ := ethLayer.(*layers.Ethernet)

    switch uint16(eth.EthernetType) {
    case lldpEtherType:
        handleLLDPPacket(deviceName, eth)
    case cdpEtherType:
        handleCDPPacket(deviceName, eth)
    case arpEtherType:
        handleARPPacket(deviceName, eth, packet)
    default:
        // Not LLDP, CDP, or ARP - ignore
    }
}

// ---- LLDP Handling ----

// LLDPFields holds fields we care about from the LLDP payload.
type LLDPFields struct {
    ChassisID  string
    PortID     string
    SystemName string
}

// handleLLDPPacket decodes the LLDP data and stores it as an edge in our graph.
func handleLLDPPacket(deviceName string, eth *layers.Ethernet) {
    payload := eth.Payload
    fields := parseLLDPFields(payload)

    // If we have no system name but a chassis, use the chassis ID as the "device name".
    remoteDeviceName := fields.SystemName
    if remoteDeviceName == "" {
        remoteDeviceName = fields.ChassisID
        if remoteDeviceName == "" {
            remoteDeviceName = "UnknownRemote"
        }
    }

    // Build our local and remote nodes:
    localNode := Node{
        Device:    localHostname,
        Interface: deviceName,
        MAC:       getInterfaceMAC(deviceName).String(),
    }
    remoteNode := Node{
        Device:    remoteDeviceName,
        Interface: fields.PortID,
        MAC:       eth.SrcMAC.String(),
    }

    // Store the edge in our global slice:
    edgesMu.Lock()
    edges = append(edges, Edge{Local: localNode, Remote: remoteNode})
    edgesMu.Unlock()
}

// parseLLDPFields does a minimal parse of the LLDP TLV structure
// and returns the key fields (ChassisID, PortID, SystemName).
func parseLLDPFields(payload []byte) LLDPFields {
    var fields LLDPFields
    offset := 0

    for offset < len(payload) {
        // Need at least 2 bytes for a TLV header.
        if len(payload[offset:]) < 2 {
            break
        }
        // 2-byte TLV header: [7 bits of Type | 9 bits of Length]
        tlvHeader := binary.BigEndian.Uint16(payload[offset : offset+2])
        offset += 2

        tlvType := tlvHeader >> 9
        tlvLen := tlvHeader & 0x1FF

        if tlvLen == 0 || offset+int(tlvLen) > len(payload) {
            break
        }
        tlvValue := payload[offset : offset+int(tlvLen)]
        offset += int(tlvLen)

        switch tlvType {
        case lldpTLVTypeEnd:
            // End of LLDPDU
            return fields
        case lldpTLVTypeChassisID:
            // Byte 0 is sub-type, so actual chassis ID data is after that
            if len(tlvValue) > 1 {
                fields.ChassisID = string(tlvValue[1:])
            }
        case lldpTLVTypePortID:
            // Byte 0 is sub-type, so actual port ID data is after that
            if len(tlvValue) > 1 {
                fields.PortID = string(tlvValue[1:])
            }
        case lldpTLVTypeSystemName:
            // System name is directly the entire TLV value
            fields.SystemName = string(tlvValue)
        // You can also parse TTL, Port Description, etc.
        }
    }
    return fields
}

// getInterfaceMAC attempts to look up the local interface's MAC address.
func getInterfaceMAC(ifName string) net.HardwareAddr {
    iface, err := net.InterfaceByName(ifName)
    if err != nil {
        return nil
    }
    return iface.HardwareAddr
}

// ---- CDP Handling ----

// handleCDPPacket logs CDP neighbor data. Optionally parse details to store in edges.
func handleCDPPacket(deviceName string, eth *layers.Ethernet) {
    payload := eth.Payload

    // Potentially parse the CDP payload (TLVs, device ID, port ID, etc.).
    // For now we just store minimal info in discoveredNeighbors.
    neighborKey := fmt.Sprintf("%s-CDP-%s", deviceName, eth.SrcMAC)
    details := fmt.Sprintf("CDP payload length: %d bytes", len(payload))

    neighbor := NeighborInfo{
        InterfaceName: deviceName,
        SourceMAC:     eth.SrcMAC.String(),
        Protocol:      "CDP",
        Details:       details,
    }
    discoveredNeighbors.Store(neighborKey, neighbor)

    // (Optional) You might add edges just like LLDP:
    // localNode := Node{ ... }
    // remoteNode := Node{ ... } // parse from CDP device ID, etc.
    // storeEdge(localNode, remoteNode)
}

// ---- ARP Handling ----

// handleARPPacket logs ARP neighbor data. We could also store them in a graph.
func handleARPPacket(deviceName string, eth *layers.Ethernet, packet gopacket.Packet) {
    arpLayer := packet.Layer(layers.LayerTypeARP)
    if arpLayer == nil {
        return
    }
    arp, _ := arpLayer.(*layers.ARP)

    neighborKey := fmt.Sprintf("%s-ARP-%s", deviceName, eth.SrcMAC)
    details := fmt.Sprintf("ARP: SenderIP=%s, SenderMAC=%s, TargetIP=%s, TargetMAC=%s",
        net.IP(arp.SourceProtAddress).String(),
        net.HardwareAddr(arp.SourceHwAddress).String(),
        net.IP(arp.DstProtAddress).String(),
        net.HardwareAddr(arp.DstHwAddress).String())

    neighbor := NeighborInfo{
        InterfaceName: deviceName,
        SourceMAC:     eth.SrcMAC.String(),
        Protocol:      "ARP",
        Details:       details,
    }
    discoveredNeighbors.Store(neighborKey, neighbor)

    // (Optional) you can unify with edges if you want to represent ARP as well.
}

// storeEdge is a helper function if you want a uniform approach for any discovered link.
// (Example usage: unify LLDP/CDP into the same adjacency structure.)
func storeEdge(local, remote Node) {
    edgesMu.Lock()
    defer edgesMu.Unlock()
    edges = append(edges, Edge{Local: local, Remote: remote})
}
