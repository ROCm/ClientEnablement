package main

import (
    "encoding/json"
    "fmt"
    "io/ioutil"
    "log"
    "path/filepath"
    "sort"
    "strings"
)

// Edge represents one LLDP edge (local -> remote).
type Edge struct {
    Local  Node `json:"local"`
    Remote Node `json:"remote"`
}

// Node is one end of an LLDP link.
type Node struct {
    Device    string `json:"device"`
    Interface string `json:"interface"`
    MAC       string `json:"mac"`
}

// DeviceInfo holds the data for each device from devices.json.
type DeviceInfo struct {
    Device  string `json:"device"`
    Type    string `json:"type"`
    Subtype string `json:"subtype,omitempty"`
}

func main() {
    // 1) Parse devices.json to build a map of device -> DeviceInfo
    deviceMap, err := parseDevicesInfo("./data/devices.json")
    if err != nil {
        log.Fatalf("Failed to parse devices.json: %v", err)
    }

    // 2) Read all JSON topology snippet files in a given directory
    inputDir := "./data" // Change to your directory path
    files, err := ioutil.ReadDir(inputDir)
    if err != nil {
        log.Fatalf("Error reading input directory: %v", err)
    }

    // We'll store all edges in a slice:
    var allEdges []Edge

    for _, f := range files {
        if f.IsDir() {
            continue
        }
        if !strings.HasSuffix(f.Name(), ".json") {
            continue
        }
        fullPath := filepath.Join(inputDir, f.Name())
        edges, err := parseEdgesFromJSON(fullPath)
        if err != nil {
            log.Printf("Skipping file %s due to parse error: %v\n", f.Name(), err)
            continue
        }
        allEdges = append(allEdges, edges...)
    }

    // 3) Build sets of device interfaces so we know which interfaces/ports belong to each device.
    //    Use nested maps: device -> map[interfaceName]bool
    deviceInterfaces := make(map[string]map[string]bool)
    for _, e := range allEdges {
        if _, ok := deviceInterfaces[e.Local.Device]; !ok {
            deviceInterfaces[e.Local.Device] = make(map[string]bool)
        }
        deviceInterfaces[e.Local.Device][e.Local.Interface] = true

        if _, ok := deviceInterfaces[e.Remote.Device]; !ok {
            deviceInterfaces[e.Remote.Device] = make(map[string]bool)
        }
        deviceInterfaces[e.Remote.Device][e.Remote.Interface] = true
    }

    // 4) Generate the DOT output
    dot := generateDOT(deviceMap, deviceInterfaces, allEdges)

    // 5) Print to stdout (or write to a file if you prefer)
    fmt.Println(dot)
}

// parseEdgesFromJSON opens a JSON file with an array of Edge objects and returns them.
func parseEdgesFromJSON(filePath string) ([]Edge, error) {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, err
    }
    var edges []Edge
    if err := json.Unmarshal(data, &edges); err != nil {
        return nil, err
    }
    return edges, nil
}

// parseDevicesInfo opens the devices.json file, parses it, and builds a map from device name to DeviceInfo.
func parseDevicesInfo(filePath string) (map[string]DeviceInfo, error) {
    // Read the devices.json file
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return nil, err
    }

    var deviceList []DeviceInfo
    if err := json.Unmarshal(data, &deviceList); err != nil {
        return nil, err
    }

    // Convert slice to map for easy lookups by device name
    deviceMap := make(map[string]DeviceInfo)
    for _, d := range deviceList {
        deviceMap[d.Device] = d
    }
    return deviceMap, nil
}

// generateDOT returns a string containing the Graphviz DOT for all devices and edges.
func generateDOT(
    deviceMap map[string]DeviceInfo,
    deviceInterfaces map[string]map[string]bool,
    edges []Edge,
) string {
    var sb strings.Builder

    // Increase ranksep / nodesep for more horizontal/vertical space
    // rankdir=LR gives a left-to-right layout.
    sb.WriteString(`digraph G {
  graph [ rankdir=LR; fontsize=10; labelloc="t"; label="Network Topology"; nodesep=1.0; ranksep=3.0 ];
  node [shape=record, fontsize=9, style=filled, fillcolor=lightgrey, width=2.5, height=1.0];
  edge [fontsize=8];

`)

    // Gather devices by bucket:
    //   - frontend switches (left)
    //   - servers (middle)
    //   - backend switches (right)
    var frontendSwitches, backendSwitches, servers []string

    for device := range deviceInterfaces {
        info, found := deviceMap[device]
        if found && strings.ToLower(info.Type) == "switch" {
            // If it's a switch, check subtype:
            if strings.ToLower(info.Subtype) == "backend" {
                backendSwitches = append(backendSwitches, device)
            } else {
                // anything not "backend" is treated as "frontend" here
                frontendSwitches = append(frontendSwitches, device)
            }
        } else {
            // default to server if not found or type != switch
            servers = append(servers, device)
        }
    }

    // Sort each bucket for consistent labeling
    sort.Strings(frontendSwitches)
    sort.Strings(servers)
    sort.Strings(backendSwitches)

    // 1) Subgraph for frontend switches
    sb.WriteString("  subgraph cluster_frontend {\n")
    sb.WriteString(`    rank=source; label="Frontend Switches"; style=dotted; color=gray;`)
    sb.WriteString("\n")

    for _, dev := range frontendSwitches {
        sb.WriteString(generateRecordNode(deviceMap, dev, deviceInterfaces[dev]))
    }
    sb.WriteString("  }\n\n")

    // 2) Subgraph for servers in the middle
    sb.WriteString("  subgraph cluster_servers {\n")
    sb.WriteString(`    label="Servers"; style=dotted; color=gray;`)
    sb.WriteString("\n")

    for _, dev := range servers {
        sb.WriteString(generateRecordNode(deviceMap, dev, deviceInterfaces[dev]))
    }
    sb.WriteString("  }\n\n")

    // 3) Subgraph for backend switches
    sb.WriteString("  subgraph cluster_backend {\n")
    sb.WriteString(`    rank=sink; label="Backend Switches"; style=dotted; color=gray;`)
    sb.WriteString("\n")

    for _, dev := range backendSwitches {
        sb.WriteString(generateRecordNode(deviceMap, dev, deviceInterfaces[dev]))
    }
    sb.WriteString("  }\n\n")

    // 4) Finally, define edges for each local->remote link
    for _, e := range edges {
        localPort := sanitizePort(e.Local.Interface)
        remotePort := sanitizePort(e.Remote.Interface)

        localID := sanitizeID(e.Local.Device)
        remoteID := sanitizeID(e.Remote.Device)

        sb.WriteString(fmt.Sprintf("  %s:%s -> %s:%s;\n", localID, localPort, remoteID, remotePort))
    }

    sb.WriteString("}\n")
    return sb.String()
}

// generateRecordNode creates the DOT record-based label for one device node.
func generateRecordNode(
    deviceMap map[string]DeviceInfo,
    device string,
    ifaceMap map[string]bool,
) string {
    var sb strings.Builder

    sanitizedDeviceID := sanitizeID(device)
    info, found := deviceMap[device]

    // Default styling is "server" if not in deviceMap
    deviceType := "server"
    fillColor := "lightgreen"

    if found {
        deviceType = strings.ToLower(info.Type)
        switch deviceType {
        case "switch":
            fillColor = "lightblue"
        case "server":
            fillColor = "lightgreen"
        default:
            fillColor = "lightgrey"
        }
    }

    // Collect and sort interfaces for consistent ordering
    var interfaces []string
    for iface := range ifaceMap {
        interfaces = append(interfaces, iface)
    }
    sort.Strings(interfaces)

    // Switch: ports on left; server: ports on right
    if deviceType == "switch" {
        sb.WriteString(fmt.Sprintf("    %s [label=\"{ ", sanitizedDeviceID))
        for i, iface := range interfaces {
            portName := sanitizePort(iface)
            sb.WriteString(fmt.Sprintf("<%s> %s", portName, iface))
            if i < len(interfaces)-1 {
                sb.WriteString(" | ")
            }
        }
        sb.WriteString(fmt.Sprintf(" | %s }\", fillcolor=%s];\n", device, fillColor))
    } else {
        // server
        sb.WriteString(fmt.Sprintf("    %s [label=\"{ %s", sanitizedDeviceID, device))
        for _, iface := range interfaces {
            portName := sanitizePort(iface)
            sb.WriteString(fmt.Sprintf(" | <%s> %s", portName, iface))
        }
        sb.WriteString(fmt.Sprintf(" }\", fillcolor=%s];\n", fillColor))
    }

    return sb.String()
}

// sanitizePort replaces special characters like '/' with '_' so they can be used as port labels in DOT.
func sanitizePort(port string) string {
    replacer := strings.NewReplacer(
        "/", "_",
        "-", "_",
        ".", "_",
        ":", "_",
        " ", "_",
    )
    return replacer.Replace(port)
}

// sanitizeID ensures device names are safe for DOT IDs (no spaces, punctuation, etc.).
func sanitizeID(id string) string {
    replacer := strings.NewReplacer(
        "-", "_",
        ".", "_",
        ":", "_",
        "/", "_",
        " ", "_",
    )
    return replacer.Replace(id)
}
