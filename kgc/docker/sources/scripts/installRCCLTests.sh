###########################################################
# Compile rccl-tests 
###########################################################
rm -rf /opt/rccl-tests/

if [ ! -e /home/user/rccl-tests ]; then
  echo "ERROR: Did not find /home/user/rccl-tests directory"
  exit 1
fi

echo "Build RCCL Testing into /opt/rccl-tests/bin"

export ROCM_PATH=/opt/rocm
cd /home/user/rccl-tests
./install.sh --gpu_targets=gfx90a,gfx942 --mpi  --mpi_home=/opt/ompi
mkdir -p /opt/rccl-tests/bin
scp -pr build/*  /opt/rccl-tests/bin/.
cd ..
