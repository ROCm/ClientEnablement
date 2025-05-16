# Run an AGFHC  test via Docker

echo "Start AGFHC Tests"
echo "HostName =" `hostname`

image=amd_ubuntu_rocm640_kgc

echo "Using " $image

date
timestamp=$(date +%Y%m%d_%H%M%S);
if [[ ! -e OUTPUT ]]; then
  mkdir OUTPUT
  chmod -R a+rw OUTPUT
fi

# Note add  -D Experimental if you are using CDash
docker run  \
        --user $(id -u):$(id -g) \
        -w /home/user \
        --hostname=$(hostname) \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        ${image} sh -c "cd /home/user/Tests/agfhc/; \
                                       rm tests/*; \
                                       cmake -DBUILDNAME=AGFHC-Tests -S . -B tests -DSITE=$(hostname) ; \
                                       cd tests; \
                                       ctest --test-dir /home/user/Tests/agfhc/tests\
                                           -L MI300X_all_level1 \
                                           --verbose -O  /user/OUTPUT/agfhc-test.${timestamp}.output; sync; "                                 
date
