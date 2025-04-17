# ce
Customer Engineering

# tools

# netgraph

netgraph is a tool to discover network topology based on LLDP, ARP, CDP

generates a JSON file as output, when invoked with a filename as optional argument it writes the JSON contents to file

```
[
  {
    "local": {
      "device": "gpu-6",
      "interface": "ens2np0",
      "mac": "5c:25:73:3c:67:86"
    },
    "remote": {
      "device": "swi61",
      "interface": "ethernet-1/30",
      "mac": "58:30:6e:e3:1a:cb"
    }![nscale dot](https://github.com/user-attachments/assets/048cfa77-d1dc-41d9-8751-644097c69742)

  },
  {
    "local": {
      "device": "gpu-6",
      "interface": "ens4np0",
      "mac": "5c:25:73:37:28:32"
    },
    "remote": {
      "device": "swi63",
      "interface": "ethernet-1/30",
      "mac": "58:30:6e:ce:3f:5b"
    }
  },
... and so on

```

For collecting the data from nscale cluster, place the netgraph executable in /shared/apps directory, and invoke it like so:

```
time ( for x in $(echo 1 2 ); do ( ssh gpu-$x sudo /shared/apps/netgraph >&  /shareddata/prasanna/netgraph.gpu-$x & ) ; done )
```
Above, its an example to run it on 2 nodes (gpu-1 and gpu-2)

You can use sbatch script like this:

```
#!/bin/bash
#SBATCH -J netgraph_collector
#SBATCH -o /shareddata/prasanna/netgraph.$HOSTNAME.out
#SBATCH -t 2
#SBATCH -N 4
#SBATCH -p MI300_Ubuntu22
# Ensure that this script is launched from a read-write-able space
sudo /shared/apps/netgraph netgraph.gpu-$(HOSTNAME).json >&  /shareddata/prasanna/netgraph.gpu-$(HOSTNAME)
```


Once all the json files are created, you can transfer to post-process:

```
cp *.json json_snippets/
bin/gendot >& nscale.dot
dot -Tsvg -O nscale.dot
cp nscale.dot.svg ~/Downloads/
```

Its first iteration, the generated SVG file looks quite clunky, should be cleaned up in coming iterations.

# Using netgraph with gentopo

gentopo aims to remove graphviz / dot from the picture (as seen in gendot sub-tool)

Idea is to directly generate SVG file from json files as input.

Move all the netgraph*.json files to a directory named 'data/' relative to the path where gentopo is run from.
And create a special file called 'devices.json' to tag certain devices as switches, and certain devices as servers. And some of the switches as frontend switches. Here's how to do it:

**** create a list of servers using 
cat netgraph.* | egrep -A1 'local|remote' | grep device | awk '{print $NF}' | sort | uniq | tr -s "\"," " "
for example:
```
cat netgraph.* | egrep -A1 'local|remote' | grep device | awk '{print $NF}' | sort | uniq | tr -s "\"," " "
 chi2374.ord.vultr.cpe.ice.amd.com 
 chi2398.ord.vultr.cpe.ice.amd.com 
 chi2429.ord.vultr.cpe.ice.amd.com 
 chi2430.ord.vultr.cpe.ice.amd.com 
 chi2431.ord.vultr.cpe.ice.amd.com 
 chi2437.ord.vultr.cpe.ice.amd.com 
 chi2440.ord.vultr.cpe.ice.amd.com 
 chi2501.ord.vultr.cpe.ice.amd.com 
 chi2505.ord.vultr.cpe.ice.amd.com 
 chi2506.ord.vultr.cpe.ice.amd.com 
 ds1-m4-p01a12b12-A.chi3 
 ds1-m4-p01a12b12-B.chi3 
 ds1-m4-p01c02d02-A.chi3 
 ds1-m4-p01c02d02-B.chi3 
 sf1-j8-p01e01.chi3.as20473.net 
 sf1-j8-p01e05.chi3.as20473.net 
 sf2-j8-p01e01.chi3.as20473.net 
 sf2-j8-p01e05.chi3.as20473.net 
 sf3-j8-p01e01.chi3.as20473.net 
 sf3-j8-p01e05.chi3.as20473.net 
 sf4-j8-p01e01.chi3.as20473.net 
 sf4-j8-p01e05.chi3.as20473.net 
 sf5-j8-p01e01.chi3.as20473.net 
 sf5-j8-p01e05.chi3.as20473.net 
 sf6-j8-p01e01.chi3.as20473.net 
 sf6-j8-p01e05.chi3.as20473.net 
 sf7-j8-p01e01.chi3.as20473.net 
 sf7-j8-p01e05.chi3.as20473.net 
 sf8-j8-p01e01.chi3.as20473.net 
 sf8-j8-p01e05.chi3.as20473.net
```
# redirect output to /tmp/devices
cat netgraph.* | egrep -A1 'local|remote' | grep device | awk '{print $NF}' | sort | uniq | tr -s "\"," " " | sed -e 's/ //g' > /tmp/devices

**** create json snippets for devices
## create servers

cat /tmp/devices | grep '^chi' | awk '{printf "\t{\n\t\t\"device\": \"%s\",\n\t\t\"type\": \"server\"\n\t},\n", $1}'
```
cat /tmp/devices | grep '^chi' | awk '{printf "\t{\n\t\t\"device\": \"%s\",\n\t\t\"type\": \"server\"\n\t},\n", $1}'
	{
		"device": "chi2374.ord.vultr.cpe.ice.amd.com",
		"type": "server"
	},
	{
		"device": "chi2398.ord.vultr.cpe.ice.amd.com",
		"type": "server"
	},
	...
```
## create frontend switches
```
cat /tmp/devices | grep '^ds1' | awk '{printf "\t{\n\t\t\"device\": \"%s\",\n\t\t\"type\": \"switch\",\n\t\t\"subtype\": \"frontend\"\n\t},\n", $1}' > /tmp/fswitches
(base) prmuruge@scsprmuruge01:~/git/feb/ce/netgraph/data$ cat !$
cat /tmp/fswitches
	{
		"device": "ds1-m4-p01a12b12-A.chi3",
		"type": "switch",
		"subtype": "frontend"
	},
	{
		"device": "ds1-m4-p01a12b12-B.chi3",
		"type": "switch",
		"subtype": "frontend"
	},
```
## create backend switches
```
(base) prmuruge@scsprmuruge01:~/git/feb/ce/netgraph/data$ cat /tmp/devices | grep '^sf' | awk '{printf "\t{\n\t\t\"device\": \"%s\",\n\t\t\"type\": \"switch\"\n\t},\n", $1}' > /tmp/bswitches
(base) prmuruge@scsprmuruge01:~/git/feb/ce/netgraph/data$ cat !$
cat /tmp/bswitches
	{
		"device": "sf1-j8-p01e01.chi3.as20473.net",
		"type": "switch"
	},
	{
		"device": "sf1-j8-p01e05.chi3.as20473.net",
		"type": "switch"
	},
	{
		"device": "sf2-j8-p01e01.chi3.as20473.net",
		"type": "switch"
	},
	{
		"device": "sf2-j8-p01e05.chi3.as20473.net",
		"type": "switch"

```

## merge all the json snippets
once you have the servers, bswitches, and fswitches, you can concatenate all of them and enclose it in a '[' at top and ']' bottom to form a proper json file (remove the last comma), and you're good to go.

## generating svg from gentopo
Here's how to run gentopo. Assumption is 'data' directory exists relative to current directory and it has all the netgraph*.json files and devices.json file within it. If you see '0 edges' thats an indication of failure to parse any files!

```
(base) prmuruge@scsprmuruge01:~/git/feb/ce/netgraph$ bin/gentopo
2025/02/27 16:24:14 Parsed 222 edges from 10 file(s).
Generated network_topology.svg. Open it in a browser to view.
Use the top buttons to search, pan, and reset the view.
```