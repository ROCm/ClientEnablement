# Run rvs test via Docker

echo "Start RVS Tests"
echo "HostName  = " `hostname`

image=amd_ubuntu_rocm640_kgc
echo "Using " $image


if [[ $# -ne 1 ]]; then
    echo 'Too many/few arguments, expecting exactly one parm' >&2
    echo 'Parm must be one of:  Basic, Stress, QT, Check' >&2
    exit 1
fi

case $1 in
    Basic|Stress|QT|Check)  # Ok
        ;;
    *)
        echo 'Parm must be one of:  Basic, Stress, QT, Check' >&2
        exit 1
esac


date
export timestamp=$(date +%Y%m%d_%H%M%S);
if [[ ! -e OUTPUT ]]; then
   mkdir  OUTPUT
fi
chmod -R a+rw OUTPUT

docker run  \
        --user $(id -u):$(id -g) \
        -w /home/user \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        $image rvs -g

docker run  \
        --user $(id -u):$(id -g) \
        -w /home/user \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        $image rvs --listTests

# Note add -D Experimental if you are using CDash
docker run  \
        --user $(id -u):$(id -g) \
        -w /home/user \
        --hostname=$(hostname) \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        $image sh -c "cd /home/user/Tests/rvs/; \
                                       rm tests/*; \
                                       cmake -DBUILDNAME=RVS-Tests -S . -B tests  -DSITE="`hostname`"; \
                                       cd tests; \
                                       ctest -L $1 --verbose /user/OUTPUT/rvs-test."${timestamp}".output; sync; "

date

