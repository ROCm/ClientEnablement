# build a docker image with the test scripts


# copy in the source code for the tests
#
# rocm-examples
#
pushd sources
if [[ ! -e ./rocm-examples ]]; then
    git clone git@github.com:Rocm/rocm-examples.git
fi;
#
#
# RCCL tests
if [[ ! -e ./rccl ]]; then
  git clone --recurse-submodules git@github.com:ROCm/rccl.git

  # master branch of rccl-tests is old but works 
  # git clone git@github.com:ROCm/rccl-tests.git -b master-cpp

  # develop branch of rccl-tests  works up to 41b383a 
  git clone  git@github.com:ROCm/rccl-tests.git -b develop
  cd rccl-tests
  git reset --hard 41b383a
  cd ..

fi;
#
#
# hip tests
#
if [[ ! -e ./hip-tests ]]; then
    git clone --recurse-submodules -b release/rocm-rel-6.4 git@github.com:ROCm/hip-tests.git
fi;
#

popd 

docker build --no-cache  --build-arg USER_ID=$(id -u) --build-arg GROUP_ID=$(id -g) \
	  -t  amd_ubuntu_rocm640_kgc -f Dockerfile .
