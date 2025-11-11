#!/bin/bash
set -x 
export WORKDIR=~/nfs_models/rccl
export OMPI_INSTALL_DIR=${WORKDIR}/ompi/install
export PATH=${OMPI_INSTALL_DIR}/bin:${PATH}
export RCCL_INSTALL_DIR=/opt/rocm/lib
export LD_LIBRARY_PATH=${RCCL_INSTALL_DIR}:${OMPI_INSTALL_DIR}/lib:${LD_LIBRARY_PATH}

LOGDIR=logs.0923.006

NNODES=2
GPUS=$((NNODES*8))
mkdir -p ${LOGDIR}


for TIMES in $(seq 50); do
    FILE=hosts.0923.fmt.perf.001
    shuf ${FILE}  > hosts.shuf
    PAIRS=$(($(wc -l ${FILE} | awk '{print $1}')/2))
    mkdir -p hostfile
    cd hostfile
    bash ../split.sh ../hosts.shuf
    cd ..

    pids=()
    for x in $(seq 1 ${PAIRS}); do 
	HOSTFILE=./hostfile/$(printf "tt%02d" $x);
	OUT=${LOGDIR}/$(printf "%siter%d.out" $(cat ${HOSTFILE} | tr -s "\n" "_") $TIMES)
	echo "RUNNING TIMES $TIMES $HOSTFILE " >> ${OUT}
	cat ${HOSTFILE} >> ${OUT}
	time ${OMPI_INSTALL_DIR}/bin/mpirun --allow-run-as-root -np ${GPUS} -npernode 8 --hostfile ./${HOSTFILE} --mca pml ucx  -x PATH=${PATH} -x LD_LIBRARY_PATH=${LD_LIBRARY_PATH} -x NCCL_DEBUG=WARN -x UCX_NET_DEVICES=bnxt_re0:1,bnxt_re1:1,bnxt_re2:1,bnxt_re3:1,bnxt_re4:1,bnxt_re5:1,bnxt_re7:1,bnxt_re8:1 -x NCCL_IB_HCA=bnxt_re0:1,bnxt_re1:1,bnxt_re2:1,bnxt_re3:1,bnxt_re4:1,bnxt_re5:1,bnxt_re7:1,bnxt_re8:1 -x NCCL_IB_GID_INDEX=3 -x NCCL_NET_GDR_READ=1  -x NCCL_SHM_DISABLE=1 -x NCCL_IB_PCI_RELAXED_ORDERING=1 -x HSA_FORCE_FINE_GRAIN_PCIE=1 -x NCCL_IGNORE_CPU_AFFINITY=1 -x  NCCL_MIN_NCHANNELS=64 -x NCCL_MAX_NCHANNELS=64 -x NCCL_PXN_DISABLE=0 -x HSA_NO_SCRATCH_RECLAIM=1 -x UCX_UD_TIMEOUT=30 -x NCCL_SOCKET_IFNAME=enp49s0f0np0 -x NCCL_TIMEOUT=30 -x UCX_IB_GID_INDEX=3 ${WORKDIR}/rccl-tests/build/alltoall_perf -b 1 -e 8G -f 2 -g 1 2>& - >> ${OUT} && echo $OUT >> PASSED.${LOGDIR} || echo $OUT >> FAILED.${LOGDIR} & 
	pids+=($!)
	echo "pids now ${pids[@]}"
  done


    timeout=40
    start=$(date +%s)

    while :; do
	any_alive=false

	for pid in "${pids[@]}"; do
	    if kill -0 "$pid" 2>/dev/null; then
		any_alive=true
		break
	    fi
	done

	now=$(date +%s)
	elapsed=$((now - start))

	if ! $any_alive || (( elapsed >= timeout )); then
	    break
	fi

	sleep 1
    done

    # After timeout or all finished, kill survivors
    for pid in "${pids[@]}"; do
	if kill -0 "$pid" 2>/dev/null; then
	    echo "killing ${pid} (still running after $timeout seconds)"
	    kill -9 "$pid"
	    ps -eaf | grep PRTE | awk '{print $2}' | xargs kill -9
	fi
    done
done
