If using HTTPS:

git clone https://github.com/AMD-DC-GPU/ce.git

If using SSH

git@github.com:AMD-DC-GPU/ce.git

cd ce/perf
make build
scp bin/perftune user@server:~/path

then run it so:

```
frank@ds-1e706-b06-13:~$ ./perftune -bmc https://10.7.125.214/redfish/v1/Systems/system/Bios -u root:0penBmc
=== CPU Governor ===
Check if scaling governor is 'performance'.
  Current: schedutil, Recommended: performance => FAIL

=== CPU Boost ===
Check if CPU boost is enabled (1).
  Current: 1, Recommended: 1 => PASS

=== Transparent HugePages ===
Check if THP is 'never' or 'always' (HPC). Using 'never' for this example.
  Current: madvise, Recommended: never => FAIL

=== NUMA Balancing ===
Check if auto NUMA balancing is off (0).
  Current: 0, Recommended: 0 => PASS

=== ASLR (randomize_va_space) ===
Check if address space layout randomization is disabled (0).
  Current: 2, Recommended: 0 => FAIL


=== Redfish BIOS Checks ===
Fetching from: https://10.7.125.214/redfish/v1/Systems/system/Bios as user=root

=== FINAL SUMMARY ===
PASS: 7 | FAIL: 4

[FAIL] CPU Governor (current: schedutil, recommended: performance)
[PASS] CPU Boost (current: 1, recommended: 1)
[FAIL] Transparent HugePages (current: madvise, recommended: never)
[PASS] NUMA Balancing (current: 0, recommended: 0)
[FAIL] ASLR (randomize_va_space) (current: 2, recommended: 0)
[PASS] Determinism Slider (current: CbsCmnDeterminismEnablePower, recommended: CbsCmnDeterminismEnablePower)
[PASS] cTDP (Thermal Design Power) (current: 400, recommended: 400)
[PASS] NPS (NUMA Per Socket) (current: CbsDfCmnDramNpsNPS4, recommended: CbsDfCmnDramNpsNPS4)
[PASS] TSME (current: CbsCmnMemTsmeEnableDdrDisabled, recommended: CbsCmnMemTsmeEnableDdrDisabled)
[FAIL] DFCStates (current: CbsCmnGnbSmuDfCstatesRsAuto, recommended: CbsCmnGnbSmuDfCstatesRsDisabled)
[PASS] CoreCStates (current: CbsCmnCpuGlobalCstateCtrlDisabled, recommended: CbsCmnCpuGlobalCstateCtrlDisabled)
================================
```

Caveat:

Currently, it only Supports HPE hardware,
Need to support, Dell, SMCI, Lenovo in future revisions, based on Redfish BIOS paths

Need to support Intel CPUs:

All the parameters are based on EPYC: 
As per "High Performance Computing Tuning Guide for AMD EPYC Series Processors" Section 8.4 (HPL) and 8.5 (DGEMM), SMT is disabled while collecting benchmarks, I'd assume, we want SMT disabled for AI workloads?
 https://www.amd.com/content/dam/amd/en/documents/epyc-technical-docs/tuning-guides/58002_amd-epyc-9004-tg-hpc.pdf

# bash version
Moved away from golang implementation since its simpler to distribute using script

# DELL support
```
amd@dell300x-pla-t14-09:~$ !!
sudo bash ./amd-perftune.sh -u root:calvin --bmc https://10.194.71.173/redfish/v1/Systems/System.Embedded.1/Bios
Detected system vendor: Dell
=== NUMA Balancing ===
Check if auto NUMA balancing is off (0).
  Current: 0, Recommended: 0 => PASS

=== INFO: Transparent HugePages ===
  Current: always [madvise] never
  [Guidance] For HPC/AI, 'never' is often recommended to minimize TLB overhead, but it's workload-dependent.

=== INFO: ASLR (randomize_va_space) ===
  Current: 2
  [Guidance] HPC/AI often sets this to 0 for consistent performance, though there are security considerations.


=== Redfish BIOS Checks for vendor: Dell ===
Fetching from: https://10.194.71.173/redfish/v1/Systems/System.Embedded.1/Bios as user=root

=== Additional Dell Performance Parameters (informational) ===
[INFO] NodeInterleave => Current: Disabled
       [Guidance] For AI/HPC, either Disabled or Enabled depending on your NUMA strategy.

[INFO] Hyper-Threading (LogicalProc) => Current: Disabled
       [Guidance] Enable/disable based on your workload tests.

[INFO] HW Prefetcher (ProcHwPrefetcher) => Current: Enabled
       [Guidance] Often best left Enabled for HPC/AI.

[INFO] DCU Streamer Prefetcher (DcuStreamerPrefetcher) => Current: Enabled
       [Guidance] Often best left Enabled for HPC/AI.

[INFO] DCU IP Prefetcher (DcuIpPrefetcher) => Current: Enabled
       [Guidance] Often best left Enabled for HPC/AI.


=== FINAL SUMMARY ===
PASS: 11 | FAIL: 0

[PASS] NUMA Balancing (current: 0, recommended: 0)
[PASS] Turbo Mode (ProcTurboMode) (current: Enabled, recommended: Enabled)
[PASS] Memory Frequency (MemFrequency) (current: MaxPerf, recommended: MaxPerf)
[PASS] Memory Encryption (current: Disabled, recommended: Disabled)
[PASS] CPU Power/Performance (ProcPwrPerf) (current: MaxPerf, recommended: MaxPerf)
[PASS] C1E (ProcC1E) (current: Disabled, recommended: Disabled)
[PASS] Uncore Frequency (current: MaxUFS, recommended: MaxUFS)
[PASS] System Profile (SysProfile) (current: PerfOptimized, recommended: PerfOptimized)
[PASS] CPU C-States (ProcCStates) (current: Disabled, recommended: Disabled)
[PASS] Energy Performance Bias (current: MaxPower, recommended: MaxPower)
[PASS] Energy Efficient Turbo (current: Disabled, recommended: Disabled)
================================
amd@dell300x-pla-t14-09:~$
```

# Supermicro support

## Above4GDecoding:
   MI300x cards use large BARs, so having 4‑GB+ decoding is required.

## CorePerformanceBoost:
   The HPC tuning guide recommends enabling boost to allow cores to reach higher frequencies when thermal
   and power headroom permit. In other words, set Core Performance Boost to ON rather than leaving it on
   Auto. This helps maximize performance for demanding workloads by allowing the processor to exceed its
   base frequency when needed.

## DeterminismControl and DeterminismEnable:
   The tuning guide suggests that for HPC workloads you generally want to avoid the overhead that
   comes with enforcing strict performance determinism. In practice, that means you should keep
   determinism control in manual mode and disable performance determinism. In other words, leave
   DeterminismControl set to Manual and DeterminismEnable set to “Disable Performance Determinism.”
   This effectively aligns with the guide’s recommendation—using a “Power” determinism setting
   (which essentially means not incurring the extra overhead of enforcing determinism) to maximize
   throughput and reduce latency for your HPC tasks.

## ASPMSupport:
   Disabling ASPM can reduce PCIe link latency.

## PowerProfileSelection:
   Forces the system to run at maximum performance rather than power‑saving states.

## PackagePowerLimit and PackagePowerLimitControl:
   These settings ensure that the CPUs (and indirectly, PCIe resources) aren’t artificially throttled.
   HPC tuning guide suggests setting PackagePowerLimit=400 and PackagePowerLimitControl=Manual.

## DFCstates:
   Data fabric c-states (for Infinity Link).

## xGMIForceLinkWidthControl, xGMILinkMaxSpeed, xGMILinkWidthControl, xGMIMaxLinkWidthControl:
   These settings directly affect the inter-GPU fabric. With MI300x GPUs that use AMD’s high‑speed
   interconnect, maximizing the link width is critical for scaling performance.

## NUMANodesPerSocket:
   “NPS4” can improve local memory and I/O locality for AI/HPC workloads.

## PCIeTenBitTagSupport:
   PCIeTenBitTagSupport is a BIOS feature that enables extended (10-bit) tagging for PCI Express
   transactions. By expanding the tag field, the system can handle more outstanding PCIe transactions
   concurrently, which can improve throughput and reduce bottlenecks—a key advantage in high‑performance
   workloads.

```
prmuruge@smc-sc-di15-24:~$ sudo bash ./amd-perftune.sh -u ADMIN:PASSWORD --bmc_ip 10.216.113.130
Detected system vendor: Supermicro
=== NUMA Balancing ===
Check if auto NUMA balancing is off (0).
  Current: 1, Recommended: 0 => FAIL

=== INFO: Transparent HugePages ===
  Current: always [madvise] never
  [Guidance] For HPC/AI, 'never' is often recommended, though it's workload-dependent.

=== INFO: ASLR (randomize_va_space) ===
  Current: 2
  [Guidance] HPC/AI often sets this to 0 for consistent performance, but that has security implications.


=== Redfish BIOS Checks for vendor: Supermicro ===
Fetching from: https://10.216.113.130/redfish/v1/Systems/1/Bios as user=ADMIN

=== FINAL SUMMARY ===
PASS: 12 | FAIL: 4

[FAIL] NUMA Balancing (current: 1, recommended: 0)
[FAIL] PCIeTenBitTagSupport (current: Auto, recommended: Enabled)
[PASS] xGMILinkMaxSpeed (current: Auto, recommended: Auto)
[PASS] xGMILinkWidthControl (current: Manual, recommended: Manual)
[PASS] xGMIForceLinkWidthControl (current: Force, recommended: Force)
[PASS] PowerProfileSelection (current: High Performance Mode, recommended: High Performance Mode)
[PASS] Above4GDecoding (current: Enabled, recommended: Enabled)
[PASS] PackagePowerLimit (current: 400, recommended: 400)
[PASS] PackagePowerLimitControl (current: Manual, recommended: Manual)
[PASS] ASPMSupport (current: Disabled, recommended: Disabled)
[PASS] DeterminismEnable (current: Disable Performance Determinism, recommended: Disable Performance Determinism)
[PASS] xGMIMaxLinkWidthControl (current: Manual, recommended: Manual)
[FAIL] CorePerformanceBoost (current: Auto, recommended: Enabled)
[PASS] DeterminismControl (current: Manual, recommended: Manual)
[PASS] DFCstates (current: Disabled, recommended: Disabled)
[FAIL] NUMANodesPerSocket (current: NPS1, recommended: NPS4)
================================
prmuruge@smc-sc-di15-24:~$ 
```