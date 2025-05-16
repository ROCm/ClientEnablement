# Run rocm-examples test via Docker

echo "Start ROCm-Examples Tests"
echo "HostName  =" `hostname`

image=amd_ubuntu_rocm640_kgc
echo "Using " $image

date
export timestamp=$(date +%Y%m%d_%H%M%S);
if [[ ! -e OUTPUT ]]; then
  mkdir OUTPUT
  chmod -R a+rw OUTPUT
fi


# Note add -D Experimental if you are using CDash
docker run  \
        --user $(id -u):$(id -g) \
        -w /home/user \
        --hostname=$(hostname) \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        $image sh -c "cd /home/user/Tests/rocm-examples/; \
                                       rm tests/*; \
                                       cmake -DBUILDNAME=ROCm-examples -S . -B tests -DSITE="`hostname`"; \
                                       cd tests; \
                                       ctest --verbose -O /user/OUTPUT/rocm-example."${timestamp}".output; sync; "

date
      
