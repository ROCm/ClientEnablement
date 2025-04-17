#!/usr/bin/env bash

################################################################################
# Bash script for HPC/AI OS checks + vendor-based Redfish BIOS checks
# Adds optional --bmc_ip for auto-generating the Redfish URL for Dell/HPE.
# Added optional -v argument to override vendor detection.
################################################################################

# 0) Check required tools before anything else (curl, jq)
REQUIRED_TOOLS=(curl jq)
for tool in "${REQUIRED_TOOLS[@]}"; do
  if ! command -v "$tool" &> /dev/null; then
    echo "Error: '$tool' is not installed."
    echo "       Please install it and re-run this script."
    echo "       For Ubuntu/Debian, you can do:  sudo apt-get update && sudo apt-get install -y $tool"
    exit 1
  fi
done

# Global counters
PASS_COUNT=0
FAIL_COUNT=0

# We'll store final results in an array of strings so we can print them at the end
# Each element will be "STATUS|NAME|CURRENT|RECOMMENDED"
RESULTS=()

################################################################################
# Helper: Print usage
################################################################################
usage() {
  echo "Usage: $0 [--bmc <redfish_endpoint>] [--bmc_ip <BMC IP>] [-u <user:pass>] [-v <vendor>]"
  echo ""
  echo "Either specify a full Redfish URL with --bmc, or provide a BMC IP via --bmc_ip."
  echo "If you provide --bmc_ip and the vendor is recognized (Dell, HPE, Supermicro), the script will"
  echo "automatically build the Redfish BIOS URL."
  echo ""
  echo "Optional: Use -v to specify the vendor manually (e.g., -v Supermicro) to skip vendor detection."
  echo ""
  echo "Examples:"
  echo "  $0 --bmc https://10.1.2.3/redfish/v1/Systems/system/Bios -u root:password"
  echo "  $0 --bmc_ip 10.1.2.3 -u admin:password -v Supermicro"
  echo ""
  echo "Note: If you provide both --bmc and --bmc_ip, the script will prefer --bmc and ignore --bmc_ip."
  echo ""
  exit 1
}

################################################################################
# Helper: read_single_line_file
#   Attempts to read one line from a file, else returns non-zero exit code.
################################################################################
read_single_line_file() {
  local path="$1"
  if [[ ! -f "$path" ]]; then
    return 1
  fi
  # Read just the first line
  IFS= read -r line < "$path"
  echo "$line"
}

################################################################################
# record_result
#   Helper to record PASS/FAIL results and update counters
################################################################################
record_result() {
  local pass="$1"
  local name="$2"
  local current="$3"
  local recommended="$4"

  if [[ "$pass" == "true" ]]; then
    ((PASS_COUNT++))
    RESULTS+=( "PASS|$name|$current|$recommended" )
  else
    ((FAIL_COUNT++))
    RESULTS+=( "FAIL|$name|$current|$recommended" )
  fi
}

################################################################################
# do_os_check
#   Handles reading a file, comparing to recommended, and printing result.
################################################################################
do_os_check() {
  local name="$1"
  local desc="$2"
  local recommended="$3"
  local file_path="$4"
  local check_func="$5"

  echo "=== $name ==="
  echo "$desc"

  if [[ ! -f "$file_path" ]]; then
    echo "  File not found: $file_path"
    echo ""
    record_result "false" "$name" "NotFound" "$recommended"
    return
  fi

  local content
  content="$(read_single_line_file "$file_path" 2>/dev/null)"
  if [[ $? -ne 0 ]]; then
    echo "  Error reading: $file_path"
    echo ""
    record_result "false" "$name" "Error" "$recommended"
    return
  fi

  # Call the specified check function
  # The function should echo: "<current_value>|<pass_or_fail_boolean>"
  local check_output
  check_output="$($check_func "$content" "$recommended")"
  local current_val="${check_output%%|*}"
  local pass_val="${check_output#*|}"

  if [[ "$pass_val" == "true" ]]; then
    echo "  Current: $current_val, Recommended: $recommended => PASS"
    echo ""
    record_result "true" "$name" "$current_val" "$recommended"
  else
    echo "  Current: $current_val, Recommended: $recommended => FAIL"
    echo ""
    record_result "false" "$name" "$current_val" "$recommended"
  fi
}

################################################################################
# "check functions" for OS-level checks
################################################################################

# Basic "exact match" check
check_exact_match() {
  local content="$1"
  local recommended="$2"
  local current_trimmed
  current_trimmed="$(echo "$content" | xargs)"
  if [[ "$current_trimmed" == "$recommended" ]]; then
    echo "$current_trimmed|true"
  else
    echo "$current_trimmed|false"
  fi
}

################################################################################
# OS-level checks (CPU governor, boost, NUMA balancing, etc.)
################################################################################
os_checks() {
  # Example: CPU Governor & CPU Boost checks only for HPE
  if [[ "$SYSTEM_VENDOR" == "HPE" ]]; then
    do_os_check \
      "CPU Governor" \
      "Check if scaling governor is 'performance'." \
      "performance" \
      "/sys/devices/system/cpu/cpu0/cpufreq/scaling_governor" \
      check_exact_match

    do_os_check \
      "CPU Boost" \
      "Check if CPU boost is enabled (1)." \
      "1" \
      "/sys/devices/system/cpu/cpufreq/boost" \
      check_exact_match
  fi

  # Example pass/fail check for NUMA Balancing
  do_os_check \
    "NUMA Balancing" \
    "Check if auto NUMA balancing is off (0)." \
    "0" \
    "/proc/sys/kernel/numa_balancing" \
    check_exact_match
}

################################################################################
# "Informational only" function for OS-related paths
################################################################################
report_os_file() {
  local label="$1"
  local file_path="$2"
  local guidance="$3"

  echo "=== INFO: $label ==="
  if [[ ! -f "$file_path" ]]; then
    echo "  File not found: $file_path"
    echo "  $guidance"
    echo ""
    return
  fi

  local content
  content="$(read_single_line_file "$file_path" 2>/dev/null)"
  if [[ $? -ne 0 ]]; then
    echo "  Error reading: $file_path"
    echo "  $guidance"
    echo ""
    return
  fi

  # Just show the content, no pass/fail
  echo "  Current: $content"
  echo "  $guidance"
  echo ""
}

################################################################################
# HPE BIOS checks
################################################################################
declare -A BIOS_CHECKS_HPE=(
  ["Determinism Slider"]="CbsCmnDeterminismEnable|CbsCmnDeterminismEnablePower"
  ["cTDP (Thermal Design Power)"]="CbsCmnTDPLimitRs|400"
  ["NPS (NUMA Per Socket)"]="CbsDfCmnDramNps|CbsDfCmnDramNpsNPS4"
  ["TSME"]="CbsCmnMemTsmeEnableDdr|CbsCmnMemTsmeEnableDdrDisabled"
  ["DFCStates"]="CbsCmnGnbSmuDfCstatesRs|CbsCmnGnbSmuDfCstatesRsDisabled"
  ["CoreCStates"]="CbsCmnCpuGlobalCstateCtrl|CbsCmnCpuGlobalCstateCtrlDisabled"
)

################################################################################
# Dell BIOS checks for HPC/AI
################################################################################
declare -A BIOS_CHECKS_DELL=(
  ["CPU Power/Performance (ProcPwrPerf)"]="ProcPwrPerf|MaxPerf"
  ["Turbo Mode (ProcTurboMode)"]="ProcTurboMode|Enabled"
  ["CPU C-States (ProcCStates)"]="ProcCStates|Disabled"
  ["C1E (ProcC1E)"]="ProcC1E|Disabled"
  ["Energy Efficient Turbo"]="EnergyEfficientTurbo|Disabled"
  ["Energy Performance Bias"]="EnergyPerformanceBias|MaxPower"
  ["Memory Frequency (MemFrequency)"]="MemFrequency|MaxPerf"
  ["Memory Encryption"]="MemoryEncryption|Disabled"
  ["Uncore Frequency"]="UncoreFrequency|MaxUFS"
  ["System Profile (SysProfile)"]="SysProfile|PerfOptimized"
)

################################################################################
# Supermicro BIOS checks for HPC/AI
################################################################################
# Above4GDecoding:
#   MI300x cards use large BARs, so having 4‑GB+ decoding is required.

# CorePerformanceBoost:
#   The HPC tuning guide recommends enabling boost to allow cores to reach higher frequencies when thermal
#   and power headroom permit. In other words, set Core Performance Boost to ON rather than leaving it on
#   Auto. This helps maximize performance for demanding workloads by allowing the processor to exceed its
#   base frequency when needed.

# DeterminismControl and DeterminismEnable:
#   The tuning guide suggests that for HPC workloads you generally want to avoid the overhead that
#   comes with enforcing strict performance determinism. In practice, that means you should keep
#   determinism control in manual mode and disable performance determinism. In other words, leave
#   DeterminismControl set to Manual and DeterminismEnable set to “Disable Performance Determinism.”
#   This effectively aligns with the guide’s recommendation—using a “Power” determinism setting
#   (which essentially means not incurring the extra overhead of enforcing determinism) to maximize
#   throughput and reduce latency for your HPC tasks.

# ASPMSupport:
#   Disabling ASPM can reduce PCIe link latency.

# PowerProfileSelection:
#   Forces the system to run at maximum performance rather than power‑saving states.

# PackagePowerLimit and PackagePowerLimitControl:
#   These settings ensure that the CPUs (and indirectly, PCIe resources) aren’t artificially throttled.
#   HPC tuning guide suggests setting PackagePowerLimit=400 and PackagePowerLimitControl=Manual.

# DFCstates:
#   Data fabric c-states (for Infinity Link).

# xGMIForceLinkWidthControl, xGMILinkMaxSpeed, xGMILinkWidthControl, xGMIMaxLinkWidthControl:
#   These settings directly affect the inter-GPU fabric. With MI300x GPUs that use AMD’s high‑speed
#   interconnect, maximizing the link width is critical for scaling performance.

# NUMANodesPerSocket:
#   “NPS4” can improve local memory and I/O locality for AI/HPC workloads.

# PCIeTenBitTagSupport:
#   PCIeTenBitTagSupport is a BIOS feature that enables extended (10-bit) tagging for PCI Express
#   transactions. By expanding the tag field, the system can handle more outstanding PCIe transactions
#   concurrently, which can improve throughput and reduce bottlenecks—a key advantage in high‑performance
#   workloads.

declare -A BIOS_CHECKS_SUPERMICRO=(
  ["Above4GDecoding"]="Above4GDecoding|Enabled"
  ["CorePerformanceBoost"]="CorePerformanceBoost|Auto"
  ["DeterminismControl"]="DeterminismControl|Manual"
  ["DeterminismEnable"]="DeterminismEnable|Power"
  ["ASPMSupport"]="ASPMSupport|Disabled"
  ["PowerProfileSelection"]="PowerProfileSelection|High Performance Mode"
  ["PackagePowerLimit"]="PackagePowerLimit|400"
  ["PackagePowerLimitControl"]="PackagePowerLimitControl|Manual"
  ["DFCstates"]="DFCstates|Disabled"
  ["xGMIForceLinkWidthControl"]="xGMIForceLinkWidthControl|Force"
  ["xGMILinkMaxSpeed"]="xGMILinkMaxSpeed|Auto"
  ["xGMILinkWidthControl"]="xGMILinkWidthControl|Manual"
  ["xGMIMaxLinkWidthControl"]="xGMIMaxLinkWidthControl|Manual"
  ["NUMANodesPerSocket"]="NUMANodesPerSocket|NPS4"
  ["PCIeTenBitTagSupport"]="PCIeTenBitTagSupport|Enabled"
)

################################################################################
# fetch_and_check_bios
################################################################################
fetch_and_check_bios() {
  local bmc_endpoint="$1"
  local bmc_user="$2"
  local bmc_pass="$3"
  local vendor="$4"

  echo ""
  echo "=== Redfish BIOS Checks for vendor: $vendor ==="
  echo "Fetching from: $bmc_endpoint as user=$bmc_user"

  # Curl command (insecure for demo). If no creds, skip -u; otherwise include them.
  local auth_args=()
  if [[ -n "$bmc_user" && -n "$bmc_pass" ]]; then
    auth_args=(-u "${bmc_user}:${bmc_pass}")
  fi

  local bios_json
  bios_json="$(curl -s -k "${auth_args[@]}" "$bmc_endpoint" 2>/dev/null)"
  if [[ -z "$bios_json" ]]; then
    echo "Error: Could not fetch BIOS JSON from $bmc_endpoint"
    # Mark all vendor checks as fail
    case "$vendor" in
      "HPE")
        for bios_name in "${!BIOS_CHECKS_HPE[@]}"; do
          local recommended="${BIOS_CHECKS_HPE[$bios_name]#*|}"
          record_result "false" "$bios_name" "RedfishError" "$recommended"
        done
        ;;
      "Dell")
        for bios_name in "${!BIOS_CHECKS_DELL[@]}"; do
          local recommended="${BIOS_CHECKS_DELL[$bios_name]#*|}"
          record_result "false" "$bios_name" "RedfishError" "$recommended"
        done
        ;;
      "Supermicro")
        for bios_name in "${!BIOS_CHECKS_SUPERMICRO[@]}"; do
          local recommended="${BIOS_CHECKS_SUPERMICRO[$bios_name]#*|}"
          record_result "false" "$bios_name" "RedfishError" "$recommended"
        done
        ;;
      *)
        ;;
    esac
    return
  fi

  local attributes_ok
  attributes_ok="$(echo "$bios_json" | jq -r '.Attributes | type' 2>/dev/null)"
  if [[ "$attributes_ok" != "object" ]]; then
    echo "Error: JSON does not have an 'Attributes' object"
    case "$vendor" in
      "HPE")
        for bios_name in "${!BIOS_CHECKS_HPE[@]}"; do
          local recommended="${BIOS_CHECKS_HPE[$bios_name]#*|}"
          record_result "false" "$bios_name" "RedfishError" "$recommended"
        done
        ;;
      "Dell")
        for bios_name in "${!BIOS_CHECKS_DELL[@]}"; do
          local recommended="${BIOS_CHECKS_DELL[$bios_name]#*|}"
          record_result "false" "$bios_name" "RedfishError" "$recommended"
        done
        ;;
      "Supermicro")
        for bios_name in "${!BIOS_CHECKS_SUPERMICRO[@]}"; do
          local recommended="${BIOS_CHECKS_SUPERMICRO[$bios_name]#*|}"
          record_result "false" "$bios_name" "RedfishError" "$recommended"
        done
        ;;
      *)
        ;;
    esac
    return
  fi

  # If we got here, we have a valid .Attributes object
  case "$vendor" in
    "HPE")
      for bios_name in "${!BIOS_CHECKS_HPE[@]}"; do
        local key="${BIOS_CHECKS_HPE[$bios_name]%%|*}"
        local recommended="${BIOS_CHECKS_HPE[$bios_name]#*|}"
        check_bios_attribute "$bios_json" "$bios_name" "$key" "$recommended"
      done
      ;;
    "Dell")
      # 1) Do pass/fail checks
      for bios_name in "${!BIOS_CHECKS_DELL[@]}"; do
        local key="${BIOS_CHECKS_DELL[$bios_name]%%|*}"
        local recommended="${BIOS_CHECKS_DELL[$bios_name]#*|}"
        check_bios_attribute "$bios_json" "$bios_name" "$key" "$recommended"
      done

      # 2) Additional "report-only" checks
      echo ""
      echo "=== Additional Dell Performance Parameters (informational) ==="
      report_bios_attribute \
        "$bios_json" \
        "NodeInterleave" \
        "NodeInterleave" \
        "[Guidance] For AI/HPC, either Disabled or Enabled depending on your NUMA strategy."

      report_bios_attribute \
        "$bios_json" \
        "Hyper-Threading (LogicalProc)" \
        "LogicalProc" \
        "[Guidance] Enable/disable based on your workload tests."

      report_bios_attribute \
        "$bios_json" \
        "HW Prefetcher (ProcHwPrefetcher)" \
        "ProcHwPrefetcher" \
        "[Guidance] Often best left Enabled for HPC/AI."

      report_bios_attribute \
        "$bios_json" \
        "DCU Streamer Prefetcher (DcuStreamerPrefetcher)" \
        "DcuStreamerPrefetcher" \
        "[Guidance] Often best left Enabled for HPC/AI."

      report_bios_attribute \
        "$bios_json" \
        "DCU IP Prefetcher (DcuIpPrefetcher)" \
        "DcuIpPrefetcher" \
        "[Guidance] Often best left Enabled for HPC/AI."
      ;;
    "Supermicro")
      # Perform pass/fail checks for Supermicro
      for bios_name in "${!BIOS_CHECKS_SUPERMICRO[@]}"; do
        local key="${BIOS_CHECKS_SUPERMICRO[$bios_name]%%|*}"
        local recommended="${BIOS_CHECKS_SUPERMICRO[$bios_name]#*|}"
        check_bios_attribute "$bios_json" "$bios_name" "$key" "$recommended"
      done
      ;;
    *)
      echo "No specific BIOS checks for vendor: $vendor"
      ;;
  esac
}

################################################################################
# check_bios_attribute
#   Pass/fail logic for vendor HPC/AI BIOS attributes.
################################################################################
check_bios_attribute() {
  local full_json="$1"
  local bios_name="$2"
  local key="$3"
  local recommended="$4"

  local val
  val="$(echo "$full_json" | jq -r ".Attributes.\"${key}\"")"
  if [[ "$val" == "null" ]]; then
    record_result "false" "$bios_name" "NotFound" "$recommended"
    return
  fi

  if [[ "${val,,}" == "${recommended,,}" ]]; then
    record_result "true" "$bios_name" "$val" "$recommended"
  else
    record_result "false" "$bios_name" "$val" "$recommended"
  fi
}

################################################################################
# report_bios_attribute
#   "Report-only": prints current value with guidance, no pass/fail.
################################################################################
report_bios_attribute() {
  local full_json="$1"
  local label="$2"
  local key="$3"
  local guidance_msg="$4"

  local val
  val="$(echo "$full_json" | jq -r ".Attributes.\"${key}\"")"
  if [[ "$val" == "null" ]]; then
    echo "[INFO] $label => Not Found"
  else
    echo "[INFO] $label => Current: $val"
    echo "       $guidance_msg"
  fi
  echo ""
}

################################################################################
# Main flow
################################################################################

# 1) Parse arguments
BMC_FLAG=""
BMC_IP_FLAG=""
CREDS_FLAG=""
VENDOR_OVERRIDE=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --bmc)
      BMC_FLAG="$2"
      shift 2
      ;;
    --bmc_ip)
      BMC_IP_FLAG="$2"
      shift 2
      ;;
    -u)
      CREDS_FLAG="$2"
      shift 2
      ;;
    -v)
      VENDOR_OVERRIDE="$2"
      shift 2
      ;;
    -h|--help)
      usage
      ;;
    *)
      echo "Unknown parameter: $1"
      usage
      ;;
  esac
done

# 2) If user:pass is provided, parse them
BMC_USER=""
BMC_PASS=""
if [[ -n "$CREDS_FLAG" ]]; then
  IFS=':' read -r user pass <<< "$CREDS_FLAG"
  BMC_USER="$user"
  BMC_PASS="$pass"
fi

# 3) Detect system vendor
if [[ -n "$VENDOR_OVERRIDE" ]]; then
  SYSTEM_VENDOR="$VENDOR_OVERRIDE"
  echo "Using vendor override: $SYSTEM_VENDOR"
else
  SYSTEM_VENDOR_FILE="/sys/devices/virtual/dmi/id/sys_vendor"
  VENDOR_STR="Unknown"
  if [[ -f "$SYSTEM_VENDOR_FILE" ]]; then
    VENDOR_STR="$(< "$SYSTEM_VENDOR_FILE")"
  fi

  SYSTEM_VENDOR="Unknown"
  case "$VENDOR_STR" in
    *"Dell Inc."*)
      SYSTEM_VENDOR="Dell"
      ;;
    *"Hewlett Packard Enterprise"*)
      SYSTEM_VENDOR="HPE"
      ;;
    *"Supermicro"*)
      SYSTEM_VENDOR="Supermicro"
      ;;
    *"Lenovo"*)
      SYSTEM_VENDOR="Lenovo"
      ;;
    *)
      SYSTEM_VENDOR="Unknown"
      ;;
  esac
echo "Detected system vendor: $SYSTEM_VENDOR"
fi

# 4) Construct or confirm BMC URL
#    Priority: If --bmc is specified, use it; else if --bmc_ip is given, build the URL by vendor.
#    If vendor is unknown for --bmc_ip, we show an error unless you want a default path.
if [[ -z "$BMC_FLAG" ]]; then
  if [[ -n "$BMC_IP_FLAG" ]]; then
    # Only build the Redfish BIOS URL if we know the vendor
    if [[ "$SYSTEM_VENDOR" == "HPE" ]]; then
      BMC_FLAG="https://${BMC_IP_FLAG}/redfish/v1/Systems/system/Bios"
    elif [[ "$SYSTEM_VENDOR" == "Dell" ]]; then
      BMC_FLAG="https://${BMC_IP_FLAG}/redfish/v1/Systems/System.Embedded.1/Bios"
    elif [[ "$SYSTEM_VENDOR" == "Supermicro" ]]; then
      BMC_FLAG="https://${BMC_IP_FLAG}/redfish/v1/Systems/1/Bios"
    else
      echo "Error: --bmc_ip specified, but vendor $SYSTEM_VENDOR is not recognized."
      echo "       We can't auto-construct the BIOS URL. Please use --bmc with a full path instead."
      exit 1
    fi
  fi
fi

# 5) Perform OS-level checks
os_checks

# 6) Show THP & ASLR as informational (no pass/fail)
report_os_file \
  "Transparent HugePages" \
  "/sys/kernel/mm/transparent_hugepage/enabled" \
  "[Guidance] For HPC/AI, 'never' is often recommended, though it's workload-dependent."

report_os_file \
  "ASLR (randomize_va_space)" \
  "/proc/sys/kernel/randomize_va_space" \
  "[Guidance] HPC/AI often sets this to 0 for consistent performance, but that has security implications."

# 7) If BMC is now set (either by --bmc or by auto-building it from --bmc_ip), do vendor-based BIOS checks
if [[ -n "$BMC_FLAG" ]]; then
  fetch_and_check_bios "$BMC_FLAG" "$BMC_USER" "$BMC_PASS" "$SYSTEM_VENDOR"
fi

# 8) Print final summary
echo ""
echo "=== FINAL SUMMARY ==="
echo "PASS: $PASS_COUNT | FAIL: $FAIL_COUNT"
echo ""

for result in "${RESULTS[@]}"; do
  # result is "STATUS|NAME|CURRENT|RECOMMENDED"
  IFS='|' read -r status name current recommended <<< "$result"
  echo "[$status] $name (current: $current, recommended: $recommended)"
done

echo "================================"

exit 0
