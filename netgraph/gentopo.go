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
// We have 'rack' as an optional field, but it may be missing if we want to auto-generate.
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

// deviceRow decides the row based on both Type and Subtype.
// - If Type is "server", the device goes to row 1 (middle).
// - If Type is "switch", then a Subtype of "frontend" goes to row 2 (bottom)
//   and "backend" to row 0 (top). If Subtype is missing, we default to row 0.
func deviceRow(d DeviceInfo) int {
    if d.Type == "server" {
        return 1
    } else if d.Type == "switch" {
        if d.Subtype == "frontend" {
            return 2
        }
        return 0
    }
    // Default row if unknown type.
    return 1
}

// buildAdjacency builds an undirected adjacency map from the slice of edges.
func buildAdjacency(edges []Edge) map[string][]string {
    adjacency := make(map[string][]string)
    for _, e := range edges {
        adjacency[e.Local.Device] = append(adjacency[e.Local.Device], e.Remote.Device)
        adjacency[e.Remote.Device] = append(adjacency[e.Remote.Device], e.Local.Device)
    }
    return adjacency
}

// assignRacksByConnectivity finds connected components in the adjacency graph.
// Each connected component gets a unique rack number ("1", "2", etc.).
func assignRacksByConnectivity(adjacency map[string][]string, devInfos []DeviceInfo) {
    // Create a helper map to find devices quickly
    deviceMap := make(map[string]*DeviceInfo)
    for i := range devInfos {
        deviceMap[devInfos[i].Device] = &devInfos[i]
    }

    visited := make(map[string]bool)
    rackCounter := 0

    // BFS or DFS for each device
    for _, d := range devInfos {
        // Skip if we've already assigned a rack
        if visited[d.Device] {
            continue
        }
        // BFS to find all reachable devices
        rackCounter++
        queue := []string{d.Device}
        visited[d.Device] = true

        for len(queue) > 0 {
            current := queue[0]
            queue = queue[1:]
            // Assign rack to the current device
            deviceMap[current].Rack = fmt.Sprintf("%d", rackCounter)

            // Traverse neighbors
            for _, neighbor := range adjacency[current] {
                if !visited[neighbor] {
                    visited[neighbor] = true
                    queue = append(queue, neighbor)
                }
            }
        }
    }
}

func main() {
    log.SetOutput(os.Stdout)

    // Parse command-line flags.
    // -v can be "flat" or "pan" (default = "pan").
    viewMode := flag.String("v", "pan", "view mode: 'pan' or 'flat'")
    flag.Parse()

    // -------------------------------------------------------
    // 1. Parse all JSON files matching 'data2/netgraph*.json' for edges.
    // -------------------------------------------------------
    edgeFiles, err := filepath.Glob("data2/netgraph*.json")
    if err != nil {
        log.Fatalf("Error globbing for netgraph JSON files: %v", err)
    }
    var edges []Edge
    for _, file := range edgeFiles {
        data, err := ioutil.ReadFile(file)
        if err != nil {
            log.Fatalf("Error reading file %s: %v", file, err)
        }
        var tmp []Edge
        if err := json.Unmarshal(data, &tmp); err != nil {
            log.Fatalf("Error unmarshaling file %s: %v", file, err)
        }
        edges = append(edges, tmp...)
    }
    log.Printf("Parsed %d edges from %d file(s).", len(edges), len(edgeFiles))

    // -------------------------------------------------------
    // 2. Read devices.json (assumed to be in the current directory)
    // -------------------------------------------------------
    devicesData, err := ioutil.ReadFile("data2/devices.json")
    if err != nil {
        log.Fatalf("Error reading devices.json: %v", err)
    }
    var devInfos []DeviceInfo
    if err := json.Unmarshal(devicesData, &devInfos); err != nil {
        log.Fatalf("Error unmarshaling devices.json: %v", err)
    }

    // -------------------------------------------------------
    // 2b. Assign racks automatically by connectivity if none provided.
    // (Or override the existing rack logic entirely if desired.)
    // -------------------------------------------------------
    adjacency := buildAdjacency(edges)
    assignRacksByConnectivity(adjacency, devInfos)

    // -------------------------------------------------------
    // 3. Group devices into rows using deviceRow().
    // Row 0: backend switches, Row 1: servers, Row 2: frontend switches.
    // -------------------------------------------------------
    rowMap := make(map[int][]DeviceInfo)
    for _, d := range devInfos {
        r := deviceRow(d)
        rowMap[r] = append(rowMap[r], d)
    }

    // -------------------------------------------------------
    // 3b. Sort each row by (rack, then device name).
    // Now that racks have been assigned, we can sort.
    // -------------------------------------------------------
    for r := 0; r < 3; r++ {
        sort.Slice(rowMap[r], func(i, j int) bool {
            di := rowMap[r][i]
            dj := rowMap[r][j]

            if di.Rack != dj.Rack {
                return di.Rack < dj.Rack
            }
            return di.Device < dj.Device
        })
    }

    // Define Y coordinates for the 3 rows.
    // Row 0 (backend switches): y=100, Row 1 (servers): y=300, Row 2 (frontend switches): y=500.
    rowY := []float64{100, 300, 500}

    // Define a wide coordinate space (totalWidth) to hold all devices.
    totalWidth := 3000.0

    // Compute positions for each device in its row.
    // We'll space them evenly across totalWidth.
    positions := make(map[string]PositionedDevice)
    for r := 0; r < 3; r++ {
        rowDevices := rowMap[r]
        n := len(rowDevices)
        if n == 0 {
            continue
        }
        step := totalWidth / float64(n+1)
        for i, dev := range rowDevices {
            x := step * float64(i+1)
            y := rowY[r]
            positions[dev.Device] = PositionedDevice{
                DeviceInfo: dev,
                X:          x,
                Y:          y,
            }
        }
    }

    // -------------------------------------------------------
    // 4. Generate "network_topology.svg"
    // -------------------------------------------------------
    outFile, err := os.Create("network_topology.svg")
    if err != nil {
        log.Fatalf("Error creating output file: %v", err)
    }
    defer outFile.Close()

    // Write XML prolog and DOCTYPE.
    fmt.Fprintln(outFile, `<?xml version="1.0" standalone="no"?>`)
    fmt.Fprintln(outFile, `<!DOCTYPE svg PUBLIC "-//W3C//DTD SVG 1.1//EN"`)
    fmt.Fprintln(outFile, `  "http://www.w3.org/Graphics/SVG/1.1/DTD/svg11.dtd">`)

    // Decide dimensions based on mode.
    var svgWidth, svgHeight int
    if *viewMode == "pan" {
        svgWidth = 1200
        svgHeight = 600
    } else {
        // Provide a wide, tall canvas to avoid cropping in flat mode.
        // Just assume 3200x700 for now.
        svgWidth = 3200
        svgHeight = 700
    }

    fmt.Fprintf(outFile, `<svg version="1.1" baseProfile="full"
    width="%d" height="%d"
    viewBox="0 0 %d %d"
    xmlns="http://www.w3.org/2000/svg">
`, svgWidth, svgHeight, svgWidth, svgHeight)

    // -------------------------------------------------------
    // 4a. Inline <style>
    // -------------------------------------------------------
    fmt.Fprintln(outFile, `<style type="text/css"><![CDATA[
/* Device styling: rectangles instead of circles */
.node.server rect {
  fill: #8bc34a;  /* green for servers */
  stroke: black;
  stroke-width: 1px;
}
.node.switch rect {
  fill: #2196f3;  /* blue for switches */
  stroke: black;
  stroke-width: 1px;
}
.node text {
  font-size: 12px;
  text-anchor: middle;
  pointer-events: none;
}
.edge {
  stroke: #999;
  stroke-width: 2px;
}
/* Highlighted node (device) */
.highlight rect {
  fill: orange !important;
}
/* Highlighted edge style */
.highlightEdge {
  stroke: orange !important;
  stroke-width: 3px;
}
/* Faded edge style */
.fadeEdge {
  opacity: 0.2;
}
/* Fixed control buttons at the top */
.control-button {
  cursor: pointer;
  font-family: sans-serif;
  font-size: 18px;
  fill: #333;
  user-select: none;
}
.control-button:hover {
  fill: red;
}
]]></style>`)

    // BFS-based link search script for highlighting path between two servers.
    linkSearchScript := `function buildGraph() {
    var adjacency = {};
    var edges = document.querySelectorAll('.edge');
    edges.forEach(function(e) {
        var local = e.getAttribute('data-local').toLowerCase();
        var remote = e.getAttribute('data-remote').toLowerCase();
        if (!adjacency[local]) adjacency[local] = [];
        if (!adjacency[remote]) adjacency[remote] = [];
        adjacency[local].push(remote);
        adjacency[remote].push(local);
    });
    return adjacency;
}

function findPath(adjacency, start, end) {
    // BFS
    var queue = [start];
    var visited = new Set([start]);
    var parent = {};

    while (queue.length > 0) {
        var current = queue.shift();
        if (current === end) {
            // reconstruct path
            var path = [end];
            while (parent[path[path.length - 1]] !== undefined) {
                path.push(parent[path[path.length - 1]]);
            }
            path.reverse();
            return path;
        }
        if (adjacency[current]) {
            adjacency[current].forEach(function(neighbor) {
                if (!visited.has(neighbor)) {
                    visited.add(neighbor);
                    parent[neighbor] = current;
                    queue.push(neighbor);
                }
            });
        }
    }
    return null;
}

function doLinkSearch() {
    var server1 = prompt('Enter first server name:');
    if (!server1) return;
    var server2 = prompt('Enter second server name:');
    if (!server2) return;

    server1 = server1.trim().toLowerCase();
    server2 = server2.trim().toLowerCase();

    var edges = document.querySelectorAll('.edge');
    edges.forEach(function(e) {
        e.classList.remove('highlightEdge');
        e.classList.remove('fadeEdge');
    });
    var nodes = document.querySelectorAll('.node');
    nodes.forEach(function(n) {
        n.classList.remove('highlight');
    });

    var adjacency = buildGraph();
    var path = findPath(adjacency, server1, server2);
    if (!path) {
        alert('No path found between ' + server1 + ' and ' + server2);
        return;
    }

    // highlight the edges on this path
    for (var i = 0; i < path.length - 1; i++) {
        var a = path[i];
        var b = path[i + 1];
        edges.forEach(function(e) {
            var local = e.getAttribute('data-local').toLowerCase();
            var remote = e.getAttribute('data-remote').toLowerCase();
            if ((local === a && remote === b) || (local === b && remote === a)) {
                e.classList.add('highlightEdge');
            }
        });
    }

    // highlight the nodes on this path
    nodes.forEach(function(n) {
        var dev = n.getAttribute('data-device').toLowerCase();
        if (path.indexOf(dev) !== -1) {
            n.classList.add('highlight');
        }
    });
}
`

    if *viewMode == "pan" {
        // Pan mode.
        fmt.Fprintf(outFile, `<script type="text/ecmascript"><![CDATA[
var offsetX = 0;
var step = 200;   // pan step size in pixels
var viewWidth = %d;
var totalWidth = %f;
var maxOffsetX = totalWidth - viewWidth;
if (maxOffsetX < 0) {
    maxOffsetX = 0;
}
function panLeft() {
    offsetX -= step;
    if (offsetX < 0) offsetX = 0;
    updatePan();
}
function panRight() {
    offsetX += step;
    if (offsetX > maxOffsetX) offsetX = maxOffsetX;
    updatePan();
}
function updatePan() {
    var grp = document.getElementById("panGroup");
    grp.setAttribute("transform", "translate(" + (-offsetX) + ", 0)");
}

function doSearch() {
    var term = prompt("Enter device name (or substring):");
    if (!term) return;
    term = term.toLowerCase();
    var nodes = document.querySelectorAll(".node");
    var edges = document.querySelectorAll(".edge");
    edges.forEach(function(e) {
       e.classList.remove("highlightEdge");
       e.classList.remove("fadeEdge");
    });
    var matchCount = 0;
    var matchedNode = null;
    nodes.forEach(function(n) {
        n.classList.remove("highlight");
        var dev = n.getAttribute("data-device").toLowerCase();
        if (dev.indexOf(term) !== -1) {
            n.classList.add("highlight");
            matchCount++;
            matchedNode = n;
        }
    });
    if (matchCount === 1 && matchedNode) {
         var xVal = parseFloat(matchedNode.getAttribute("data-x"));
         var newOffset = xVal - viewWidth / 2;
         if (newOffset < 0) newOffset = 0;
         if (newOffset > maxOffsetX) newOffset = maxOffsetX;
         offsetX = newOffset;
         updatePan();
         var deviceName = matchedNode.getAttribute("data-device").toLowerCase();
         edges.forEach(function(e) {
             var local = e.getAttribute("data-local").toLowerCase();
             var remote = e.getAttribute("data-remote").toLowerCase();
             if (local === deviceName || remote === deviceName) {
                 e.classList.add("highlightEdge");
             } else {
                 e.classList.add("fadeEdge");
             }
         });
    }
}
function resetSearch() {
    var nodes = document.querySelectorAll(".node");
    nodes.forEach(function(n) {
        n.classList.remove("highlight");
    });
    var edges = document.querySelectorAll(".edge");
    edges.forEach(function(e) {
        e.classList.remove("highlightEdge");
        e.classList.remove("fadeEdge");
    });
}

%[1]s
]]></script>`, linkSearchScript, svgWidth, totalWidth)

        // Control bar.
        fmt.Fprintln(outFile, `<g id="controls">
  <text x="20" y="30" class="control-button" onclick="doSearch()">Search</text>
  <text x="120" y="30" class="control-button" onclick="panLeft()">&lt;</text>
  <text x="160" y="30" class="control-button" onclick="panRight()">&gt;</text>
  <text x="220" y="30" class="control-button" onclick="resetSearch()">Reset</text>
  <text x="300" y="30" class="control-button" onclick="doLinkSearch()">Link</text>
</g>`)
        fmt.Fprintln(outFile, `<g id="panGroup">`)

    } else {
        // Flat mode.
        fmt.Println("Generating a flat SVG (no panning), but with search/reset/link...")
        fmt.Fprintf(outFile, `<script type="text/ecmascript"><![CDATA[
function doSearch() {
    var term = prompt("Enter device name (or substring):");
    if (!term) return;
    term = term.toLowerCase();
    var nodes = document.querySelectorAll(".node");
    var edges = document.querySelectorAll(".edge");
    edges.forEach(function(e) {
       e.classList.remove("highlightEdge");
       e.classList.remove("fadeEdge");
    });
    var matchCount = 0;
    var matchedNode = null;
    nodes.forEach(function(n) {
        n.classList.remove("highlight");
        var dev = n.getAttribute("data-device");
        if (dev && dev.toLowerCase().indexOf(term) !== -1) {
            n.classList.add("highlight");
            matchCount++;
            matchedNode = n;
        }
    });
    if (matchCount === 1 && matchedNode) {
         var deviceName = matchedNode.getAttribute("data-device").toLowerCase();
         edges.forEach(function(e) {
             var local = e.getAttribute("data-local").toLowerCase();
             var remote = e.getAttribute("data-remote").toLowerCase();
             if (local === deviceName || remote === deviceName) {
                 e.classList.add("highlightEdge");
             } else {
                 e.classList.add("fadeEdge");
             }
         });
    }
}
function resetSearch() {
    var nodes = document.querySelectorAll(".node");
    nodes.forEach(function(n) {
        n.classList.remove("highlight");
    });
    var edges = document.querySelectorAll(".edge");
    edges.forEach(function(e) {
        e.classList.remove("highlightEdge");
        e.classList.remove("fadeEdge");
    });
}

%[1]s
]]></script>`, linkSearchScript)

        // Minimal control bar.
        fmt.Fprintln(outFile, `<g id="controls">
  <text x="20" y="30" class="control-button" onclick="doSearch()">Search</text>
  <text x="120" y="30" class="control-button" onclick="resetSearch()">Reset</text>
  <text x="220" y="30" class="control-button" onclick="doLinkSearch()">Link</text>
</g>`)

        fmt.Fprintln(outFile, `<g id="flatGroup">`)
    }

    // -------------------------------------------------------
    // Draw edges.
    // -------------------------------------------------------
    for _, e := range edges {
        localPos, okLocal := positions[e.Local.Device]
        remotePos, okRemote := positions[e.Remote.Device]
        if !okLocal || !okRemote {
            continue
        }
        fmt.Fprintf(outFile,
            `<line class="edge" data-local="%s" data-remote="%s" x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f"/>`,
            e.Local.Device, e.Remote.Device,
            localPos.X, localPos.Y,
            remotePos.X, remotePos.Y)
    }

    // -------------------------------------------------------
    // Draw devices.
    // -------------------------------------------------------
    for _, pd := range positions {
        x := pd.X
        y := pd.Y
        device := pd.Device
        devType := pd.Type

        // Determine type class.
        typeClass := "switch"
        if devType == "server" {
            typeClass = "server"
        }

        // Create a group for each device.
        fmt.Fprintf(outFile,
            `<g class="node %s" data-device="%s" data-type="%s" data-x="%.1f" data-y="%.1f" transform="translate(%.1f,%.1f)">`,
            typeClass, device, devType, x, y, x, y)

        // Rectangle centered at (0,0)
        fmt.Fprintln(outFile, `<rect x="-20" y="-10" width="40" height="20"></rect>`)

        // Tooltip (show Rack if present)
        tooltip := fmt.Sprintf("Device: %s\nType: %s", device, devType)
        if pd.Rack != "" {
            tooltip += fmt.Sprintf("\nRack: %s", pd.Rack)
        }
        fmt.Fprintf(outFile, "<title>%s</title>", tooltip)

        // Label
        fmt.Fprintf(outFile, `<text dy="4">%s</text>`, device)
        fmt.Fprintln(outFile, `</g>`)
    }

    // Close group.
    if *viewMode == "pan" {
        fmt.Fprintln(outFile, `</g>`)
    } else {
        fmt.Fprintln(outFile, `</g>`)
    }

    // -------------------------------------------------------
    // 5. Close SVG.
    // -------------------------------------------------------
    fmt.Fprintln(outFile, `</svg>`)

    if *viewMode == "pan" {
        fmt.Println("Generated network_topology.svg with panning features.")
        fmt.Println("Open it in a browser to view. Use the top buttons to search, pan, reset, or link.")
    } else {
        fmt.Println("Generated a flat network_topology.svg (no panning). Search, reset, and link are available.")
    }
}
