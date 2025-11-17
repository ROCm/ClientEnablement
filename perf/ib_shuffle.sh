#!/bin/bash
#SBATCH --job-name=ib_bw_test
#SBATCH --output=/nfs/shared/ib_tests/%x-%j.out
#SBATCH --error=/nfs/shared/ib_tests/%x-%j.err
#SBATCH --time=01:00:00
#SBATCH --ntasks-per-node=1
#SBATCH --exclusive

# --- Configuration ---
RESULT_DIR="/mnt/shared/pras/ib_tests_nov10/${SLURM_JOB_ID}"
#NIC_LIST=("rocep28s0" "rocep62s0" "rocep79s0" "rocep96s0" "rocep158s0" "rocep190s0" "rocep206s0" "rocep222s0")
NIC_LIST=("ionic_0" "ionic_1" "ionic_2" "ionic_3" "ionic_4" "ionic_5" "ionic_6" "ionic_7")
link ionic_0/1 state ACTIVE physical_state LINK_UP netdev enp121s0 
link ionic_1/1 state ACTIVE physical_state LINK_UP netdev enp9s0 
link ionic_2/1 state ACTIVE physical_state LINK_UP netdev enp105s0 
link ionic_3/1 state ACTIVE physical_state LINK_UP netdev enp25s0 
link ionic_4/1 state ACTIVE physical_state LINK_UP netdev enp249s0 
link ionic_5/1 state ACTIVE physical_state LINK_UP netdev enp137s0 
link ionic_6/1 state ACTIVE physical_state LINK_UP netdev enp233s0 
link ionic_7/1 state ACTIVE physical_state LINK_UP netdev enp153s0 
TEST_SIZE=8000000

mkdir -p "$RESULT_DIR"

echo "====================================="
echo " SLURM JOB ID: ${SLURM_JOB_ID}"
echo " Result dir  : ${RESULT_DIR}"
echo "====================================="

# --- Get node list ---
if [[ -n "$SLURM_JOB_NODELIST" ]]; then
    NODES=$(scontrol show hostnames $SLURM_JOB_NODELIST)
else
    echo "ERROR: No node allocation detected. Submit with sbatch or specify --nodelist."
    exit 1
fi

NODE_ARRAY=($NODES)
NODE_COUNT=${#NODE_ARRAY[@]}
echo "Detected $NODE_COUNT nodes in allocation."

if (( NODE_COUNT < 2 )); then
    echo "Need at least 2 nodes."
    exit 1
fi

# --- Shuffle nodes ---
SHUFFLED_NODES=($(printf "%s\n" "${NODE_ARRAY[@]}" | shuf))

# --- Form pairs ---
PAIR_COUNT=$(( NODE_COUNT / 2 ))
echo "Creating $PAIR_COUNT node pairs..."
echo

for (( i=0; i<$PAIR_COUNT; i++ )); do
    SERVER=${SHUFFLED_NODES[$((2*i))]}
    CLIENT=${SHUFFLED_NODES[$((2*i+1))]}
    echo "Pair $((i+1)): $CLIENT → $SERVER"
    (
        LOG_PREFIX="${RESULT_DIR}/pair_${i}_${CLIENT}_to_${SERVER}"
        touch "${LOG_PREFIX}.log"

        for NIC in "${NIC_LIST[@]}"; do
            echo "[$(date)] Testing IB_WRITE_BW $CLIENT → $SERVER via $NIC" | tee -a "${LOG_PREFIX}.log"
            ssh $SERVER "pkill ib_write_bw; ib_write_bw -d ${NIC} -F -s ${TEST_SIZE} -x 1 -q 2 --report_gb > /dev/null 2>&1 &"
            sleep 2
            ssh $CLIENT "ib_write_bw -d ${NIC} -F -x 1 -q 2 -s ${TEST_SIZE} --report_gbits $SERVER" \
                >> "${LOG_PREFIX}.log" 2>&1
            sleep 2

            echo "[$(date)] Testing IB_READ_BW $CLIENT → $SERVER via $NIC" | tee -a "${LOG_PREFIX}.log"
            ssh $SERVER "pkill ib_read_bw; ib_read_bw -d ${NIC} -F -s ${TEST_SIZE} -x 1 -q 2 --report_gb > /dev/null 2>&1 &"
            sleep 2
            ssh $CLIENT "ib_read_bw -d ${NIC} -F -x 1 -q 2 -s ${TEST_SIZE} --report_gbits $SERVER" \
                >> "${LOG_PREFIX}.log" 2>&1
            sleep 2
        done

        echo "Completed tests for $CLIENT → $SERVER"
    ) &
done

wait
echo "All tests complete."
echo "Results saved under: ${RESULT_DIR}"
