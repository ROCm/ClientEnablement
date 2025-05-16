# set minimum cmake version
cmake_minimum_required(VERSION 3.27)

# project name
project(test-rccl    LANGUAGES NONE)

# define tests
enable_testing()
include(CTest)

execute_process(
  COMMAND         hostname -f
  OUTPUT_VARIABLE SITENAME
)

add_test(NAME all2all  COMMAND   /opt/ompi/bin/mpirun
-np 8
-H localhost:8
-x NCCL_IB_GID_INDEX=3
-x NCCL_DEBUG=version
-x LD_LIBRARY_PATH=:${LD_LIBRARY_PATH}
-x NCCL_ALGO=Ring
-x NCCL_PROTO=Simple
--mca pml ucx --mca btl ^openib /opt/rccl-tests/bin/alltoall_perf -b 8 -e 16G -f 2 -g 1 -c 1 -n 100 )

set_tests_properties(all2all  PROPERTIES LABELS "All2All") 
set_tests_properties(all2all  PROPERTIES TIMEOUT 0)
