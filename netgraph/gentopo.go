package main

import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "os"
    "path/filepath"
    "sort"
    "strings"
)

// Node represents one end of an LLDP link.
type Node struct {
    Device    string `json:"device"`
    Interface string `json:"interface"`
    MAC       string `json:"mac"`
}

// Edge represents one LLDP edge (local -> remote).
type Edge struct {
    Local  Node `json:"local"`
    Remote Node `json:"remote"`
}

// DeviceInfo holds the data for each device from devices.json.
type DeviceInfo struct {
    Device  string `json:"device"`
    Type    string `json:"type"`
    Subtype string `json:"subtype,omitempty"`
    Rack    string `json:"rack,omitempty"`
}

// PositionedDevice holds a DeviceInfo along with its (x,y) coordinates.
type PositionedDevice struct {
    DeviceInfo
    X float64
    Y float64
}

// deviceRow decides vertical placement based on Type/Subtype.
func deviceRow(d DeviceInfo) int {
    switch d.Type {
    case "server":
        return 1
    case "switch":
        if d.Subtype == "frontend" {
            return 2
        }
        return 0
    default:
        return 1
    }
}

// buildAdjacency builds undirected adjacency from edges.
func buildAdjacency(edges []Edge) map[string][]string {
    adj := make(map[string][]string)
    for _, e := range edges {
        adj[e.Local.Device] = append(adj[e.Local.Device], e.Remote.Device)
        adj[e.Remote.Device] = append(adj[e.Remote.Device], e.Local.Device)
    }
    return adj
}

// assignRacksByConnectivity labels connected components as racks.
func assignRacksByConnectivity(adj map[string][]string, devInfos []DeviceInfo) {
    deviceMap := make(map[string]*DeviceInfo)
    for i := range devInfos {
        deviceMap[devInfos[i].Device] = &devInfos[i]
    }
    visited := make(map[string]bool)
    rackCounter := 0
    for _, d := range devInfos {
        if visited[d.Device] {
            continue
        }
        rackCounter++
        queue := []string{d.Device}
        visited[d.Device] = true
        for len(queue) > 0 {
            cur := queue[0]
            queue = queue[1:]
            deviceMap[cur].Rack = fmt.Sprintf("%d", rackCounter)
            for _, nb := range adj[cur] {
                if !visited[nb] {
                    visited[nb] = true
                    queue = append(queue, nb)
                }
            }
        }
    }
}

// indexOf returns index of s in slice, or -1.
func indexOf(s string, slice []string) int {
    for i, v := range slice {
        if v == s {
            return i
        }
    }
    return -1
}

func main() {
    log.SetOutput(os.Stdout)
    viewMode := flag.String("v", "flat", "view mode: 'flat' (default) or 'pan'")
    dataDir  := flag.String("d", "data",  "data directory")
    flag.Parse()

    // Load LLDP edges
    pattern := filepath.Join(*dataDir, "netgraph*.json")
    edgeFiles, err := filepath.Glob(pattern)
    if err != nil {
        log.Fatalf("Error globbing LLDP JSON files (%s): %v", pattern, err)
    }
    var edges []Edge
    for _, f := range edgeFiles {
        data, err := ioutil.ReadFile(f)
        if err != nil {
            log.Fatalf("Read error %s: %v", f, err)
        }
        var tmp []Edge
        if err := json.Unmarshal(data, &tmp); err != nil {
            log.Fatalf("Unmarshal error %s: %v", f, err)
        }
        edges = append(edges, tmp...)
    }
    log.Printf("Parsed %d edges", len(edges))

    // Load devices
    devPath := filepath.Join(*dataDir, "devices.json")
    devData, err := ioutil.ReadFile(devPath)
    if err != nil {
        log.Fatalf("Error reading devices.json (%s): %v", devPath, err)
    }
    var devInfos []DeviceInfo
    if err := json.Unmarshal(devData, &devInfos); err != nil {
        log.Fatalf("Unmarshal devices.json: %v", err)
    }

    // Assign racks by connectivity
    adj := buildAdjacency(edges)
    assignRacksByConnectivity(adj, devInfos)

    // Group devices into rows
    rowMap := make(map[int][]DeviceInfo)
    for _, d := range devInfos {
        rowMap[deviceRow(d)] = append(rowMap[deviceRow(d)], d)
    }
    // Sort backend switches and servers normally
    for r := 0; r <= 1; r++ {
        sort.Slice(rowMap[r], func(i, j int) bool {
            a, b := rowMap[r][i], rowMap[r][j]
            if a.Rack != b.Rack {
                return a.Rack < b.Rack
            }
            return a.Device < b.Device
        })
    }
    // Sort frontend switches by first-connection order across all servers
    frontend := rowMap[2]
    servers  := rowMap[1]
    // build server order slice
    serverOrder := make([]string, len(servers))
    for i, s := range servers {
        serverOrder[i] = s.Device
    }
    sort.SliceStable(frontend, func(i, j int) bool {
        a, b := frontend[i], frontend[j]
        // find earliest server index they connect to
        rankA, rankB := len(serverOrder), len(serverOrder)
        for idx, srv := range serverOrder {
            if indexOf(srv, adj[a.Device]) >= 0 {
                rankA = idx
                break
            }
        }
        for idx, srv := range serverOrder {
            if indexOf(srv, adj[b.Device]) >= 0 {
                rankB = idx
                break
            }
        }
        if rankA != rankB {
            return rankA < rankB
        }
        return a.Device < b.Device
    })
    rowMap[2] = frontend

    // Canvas dimensions: width by max row length
    spacingX := 200.0  // increased spacing for wider layout
    maxCount := 0
    for r := 0; r < 3; r++ {
        if cnt := len(rowMap[r]); cnt > maxCount {
            maxCount = cnt
        }
    }
    totalWidth := spacingX * float64(maxCount+1)
    // ensure a generous minimum width
    if totalWidth < 1200 {
        totalWidth = 1200
    }
    width  := int(totalWidth)
    // Height: 3 rows
    spacingY := 300.0
    height := int(spacingY * 4)

    // Compute positions
    positions := make(map[string]PositionedDevice)
    for r := 0; r < 3; r++ {
        devs := rowMap[r]
        if len(devs) == 0 {
            continue
        }
        step := totalWidth / float64(len(devs)+1)
        y    := spacingY * float64(r+1)
        for i, d := range devs {
            positions[d.Device] = PositionedDevice{d, step*float64(i+1), y}
        }
    }

    // Create SVG file
    out, err := os.Create("network_topology.svg")
    if err != nil {
        log.Fatalf("SVG create error: %v", err)
    }
    defer out.Close()

    // SVG prolog
    fmt.Fprintln(out, `<?xml version="1.0" standalone="no"?>`)
    fmt.Fprintln(out, `<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN" "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">`)
    fmt.Fprintf(out, `<svg width="%d" height="%d" viewBox="0 0 %d %d" xmlns="http://www.w3.org/2000/svg">`, width, height, width, height)

    // Definitions: filters & gradients
    fmt.Fprintln(out, `<defs>`)
    fmt.Fprintln(out, `<filter id="shadow" x="-20%" y="-20%" width="140%" height="140%">`)
    fmt.Fprintln(out, `<feGaussianBlur in="SourceAlpha" stdDeviation="3"/>`)
    fmt.Fprintln(out, `<feOffset dx="2" dy="2" result="offsetblur"/>`)
    fmt.Fprintln(out, `<feMerge><feMergeNode/><feMergeNode in="SourceGraphic"/></feMerge>`)
    fmt.Fprintln(out, `</filter>`)
    fmt.Fprintln(out, `<linearGradient id="grad_server" x1="0%" y1="0%" x2="0%" y2="100%">`)
    fmt.Fprintln(out, `<stop offset="0%" stop-color="#007c97"/>`)
    fmt.Fprintln(out, `<stop offset="100%" stop-color="#000000"/>`)
    fmt.Fprintln(out, `</linearGradient>`)
    fmt.Fprintln(out, `<linearGradient id="grad_switch" x1="0%" y1="0%" x2="0%" y2="100%">`)
    fmt.Fprintln(out, `<stop offset="0%" stop-color="#f26522"/>`)
    fmt.Fprintln(out, `<stop offset="100%" stop-color="#ed1c24"/>`)
    fmt.Fprintln(out, `</linearGradient>`)
    fmt.Fprintln(out, `</defs>`)    

    // Styles
    fmt.Fprintln(out, `<style><![CDATA[`)  
    fmt.Fprintln(out, `.node text { font-size:14px; text-anchor:middle; pointer-events:none; fill:#fff }`)  
    fmt.Fprintln(out, `.edge { stroke:#999; stroke-width:2px }`)  
    fmt.Fprintln(out, `.control-button { cursor:pointer; font-family:sans-serif; font-size:18px; fill:#333; user-select:none }`)  
    fmt.Fprintln(out, `.control-button:hover { fill:red }`)  
    fmt.Fprintln(out, `]]></style>`)  

    // Pan controls wrapper
    if *viewMode == "pan" {
        fmt.Fprintln(out, `<g id="controls"><text x="20" y="30" class="control-button" onclick="panLeft()">◀</text><text x="60" y="30" class="control-button" onclick="panRight()">▶</text></g><g id="panGroup">`)
    } else {
        fmt.Fprintln(out, `<g id="flatGroup">`)
    }

    // Draw edges with onclick alerts for server-switch
    for _, e := range edges {
        lpos := positions[e.Local.Device]
        rpos := positions[e.Remote.Device]
        isSS := (lpos.Type == "server" && rpos.Type == "switch") || (lpos.Type == "switch" && rpos.Type == "server")
        if isSS {
            var si, sw string
            if lpos.Type == "server" {
                si = e.Local.Interface
                sw = e.Remote.Interface
            } else {
                si = e.Remote.Interface
                sw = e.Local.Interface
            }
            fmt.Fprintf(out, `<line class="edge" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" onclick="alert('Interfaces: (%s, %s)')"/>`,
                lpos.X, lpos.Y, rpos.X, rpos.Y, si, sw)
        } else {
            fmt.Fprintf(out, `<line class="edge" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`, lpos.X, lpos.Y, rpos.X, rpos.Y)
        }
    }

    // Draw nodes
    for _, pd := range positions {
        parts := strings.Split(pd.Device, ".")
        label := parts[0]
        gradID := "grad_switch"
        if pd.Type == "server" {
            gradID = "grad_server"
        }
        fmt.Fprintf(out, `<g class="node" transform="translate(%.1f,%.1f)">`, pd.X, pd.Y)
        fmt.Fprintf(out, `<rect x="-60" y="-30" width="120" height="60" rx="5" ry="5" filter="url(#shadow)" fill="url(#%s)"/>`, gradID)
        fmt.Fprintf(out, `<title>Device: %s
Type: %s
Rack: %s</title>`, pd.Device, pd.Type, pd.Rack)
        fmt.Fprintf(out, `<text dy="6">%s</text>`, label)
        fmt.Fprintln(out, `</g>`)  
    }

    // Close wrapper and SVG
    fmt.Fprintln(out, `</g></svg>`)  

    log.Printf("Generated network_topology.svg (%dx%d) in %s mode", width, height, *viewMode)
    fmt.Println("network_topology.svg created.")
}