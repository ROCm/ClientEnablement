# Run hip-tests test via Docker

echo "Start HIP Tests"
echo "HostName  =" `hostname`

image=amd_ubuntu_rocm640_kgc
echo "Using " $image

date
export timestamp=$(date +%Y%m%d_%H%M%S);

if [[ ! -e OUTPUT ]]; then
   mkdir  OUTPUT
   chmod -R a+rw OUTPUT
fi

# Note add -D Experimental if you are using CDash
docker run  \
        --user $(id -u):$(id -g) \
        -w /home/user \
        --hostname=$(hostname) \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        $image sh -c  \
                                      "cd  /home/user/hip-tests; mkdir build; rm build/*; \
                                       cd  /home/user/hip-tests/catch; \
                                       scp /home/user/Tests/CTestConfig.cmake  . ; \
                                       cd  /home/user/hip-tests/build/; \
				       echo TS is  "${timestamp}" ; \
                                       cmake ../catch -DHIP_PLATFORM=amd -DBUILDNAME=hip-tests -DSITE="`hostname`"; \
                                       make build_tests -j 16 ; \
                                       ctest -R Perf  \
				         --verbose  -O /user/OUTPUT/hip-test."${timestamp}".output; sync; "

date

