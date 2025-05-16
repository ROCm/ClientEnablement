# Create base Ubuntu 22.04 (jammy) and ROCm 6.4 container for MI300
FROM ubuntu:22.04

ARG USER_ID
ARG GROUP_ID

# MI300A | MI300X | MI300X-HF | MI308X | MI308X-HF | MI325X
ARG GPU_MODEL=MI300X

# do not install AGFHC unless the package is provided externally
# AGFHC must be requested from AMD 

ARG INSTALL_AGFHC=1

# map a known userid (say user) to our host uid/gid
# set the user up as a non-password sudoer to make agfhc execution easier

RUN if [ ${USER_ID:-0} -ne 0 ] && [ ${GROUP_ID:-0} -ne 0 ]; then \
    if getent group user ; then groupdel user; fi &&\
    groupadd -g ${GROUP_ID} user &&\
    useradd -l -u ${USER_ID} -g user user &&\
    echo "user:pass!!!!" | chpasswd && \
    install -d -m 0755 -o user -g user /home/user \
;fi

RUN DEBIAN_FRONTEND=noninteractive apt-get update -y \
    && TZ=Etc/UTC ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && echo $TZ > /etc/timezone \
    && apt-get install -y ca-certificates build-essential gfortran automake autoconf git vim nano libtinfo-dev \
       initramfs-tools libtool libelf-dev numactl numad libnuma-dev numad curl wget pkg-config flex hwloc \
       libstdc++-12-dev libxml2-dev python3-dev python3-pip unzip sudo libyaml-cpp-dev  \
    && apt-get clean

RUN pip3 install PyYAML

RUN adduser user sudo && echo "user  ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers
#RUN echo "user ALL=(ALL:ALL) NOPASSWD: ALL" > /etc/sudoers.d/user 

ENV ROCM_PATH=/opt/rocm

ENV PATH=$ROCM_PATH/bin:$ROCM_PATH/profiler/bin:/opt/cmake/bin:/opt/rccl-tests/bin:/usr/local/bin:$PATH \
    LD_LIBRARY_PATH=$ROCM_PATH/lib:$ROCM_PATH/lib64:$ROCM_PATH/llvm/lib:$LD_LIBRARY_PATH \
    LIBRARY_PATH=$ROCM_PATH/lib:$ROCM_PATH/lib64:$LIBRARY_PATH \
    C_INCLUDE_PATH=$ROCM_PATH/include:$ROCM_PATH/include/rocm-core:$C_INCLUDE_PATH \
    CPLUS_INCLUDE_PATH=$ROCM_PATH/include:$ROCM_PATH/include/rocm-core:$CPLUS_INCLUDE_PATH \
    CPATH=$ROCM_PATH/include:$CPATH \
    INCLUDE=$ROCM_PATH/include:$INCLUDE \
    CMAKE_PREFIX_PATH=/opt/rocm/lib/cmake  \
    GPU_MODEL=${GPU_MODEL}

RUN  apt update -y \
     && wget https://repo.radeon.com/amdgpu-install/6.4/ubuntu/jammy/amdgpu-install_6.4.60400-1_all.deb \
     && apt install -y ./amdgpu-install_6.4.60400-1_all.deb

RUN  amdgpu-install --usecase=rocm,hiplibsdk --no-dkms -y \
    && mkdir -p /opt/cmake  \
    && wget --no-check-certificate --quiet -O - https://cmake.org/files/v3.28/cmake-3.28.1-linux-x86_64.tar.gz | tar --strip-components=1 -xz -C /opt/cmake 

# NOTE: Having multiple GPU_TARGETs can cause a bad target.lst file (makes hip-test fail) 

# inifinity band related
RUN apt-get install -y ibverbs-utils libibverbs-dev libibverbs1 librdmacm-dev libibumad-dev

WORKDIR /home/user/
COPY ./sources/ .

WORKDIR /home/user/Tests
WORKDIR /home/user/scripts

COPY ./sources/scripts/ .

# install UCX, OpenMPI

RUN ./installUCX.sh  && ./installMPI.sh

# install Google Test framework

RUN apt-get install -y libgtest-dev \
    && cd /usr/src/gtest \
    && cmake -S . -B build \
    && cd build/ \
    && make \
    && scp -p ./lib/* /usr/lib/.

# Install the AGFHC tool (if available)


WORKDIR /home/user/
# install agfhc  for 
# MI300X | MI300X-HF | MI308X | MI308X-HF | MI325X

#        /usr/bin/scp -p /home/user/agfhc-mi300x_1.18.3_jammy.tar.bz2 . && \
#        tar -xvf   agfhc-mi300x_1.18.3_jammy.tar.bz2 && \

RUN if [ "${GPU_MODEL}" = "MI300X" ]    ||  \
       [ "${GPU_MODEL}" = "MI300X-HF" ] ||  \
       [ "${GPU_MODEL}" = "MI308X" ]    ||  \
       [ "${GPU_MODEL}" = "MI308X-HF" ]     \
        ; then \
     if [ "${INSTALL_AGFHC}" = "1" ]; then \
        echo "Info: Install AGFHC " && \
        mkdir -p  /home/user/agfhc/mi300x && \
        cd /home/user/agfhc/mi300x && \
        /usr/bin/scp -p /home/user/agfhc-mi300x_1.20.2_ub2204.tar.bz2 . && \
        tar -xvf    agfhc-mi300x_1.20.2_ub2204.tar.bz2 && \
        ./install  \
     ;fi \
;fi

# or, install agfhc  for MI300A
RUN if [ "${GPU_MODEL}" = "MI300A" ] ; then \
     if [ "${INSTALL_AGFHC}" = "1" ]; then \
        echo "Info: Install AGFHC for MI300A " && \
        mkdir -p  /home/user/agfhc/mi300a && \
        cd /home/user/agfhc/mi300a && \
        /usr/bin/scp -p /home/user/agfhc-mi300a_1.20.2_ub2204.tar.bz2 . && \
        tar -xvf   agfhc-mi300a_1.20.2_ub2204.tar.bz2 && \
        ./install && exit 1  \
     ;fi \
;fi

# do not need the source/install files for agfhc so remove them
RUN rm -rf /home/user/agfhc/mi300x
RUN rm -rf /home/user/agfhc/mi300a
RUN rm -f /home/user/agfhc*.bz2

# Install the RCCL

# git clone git@github.com:ROCm/rccl.git
WORKDIR /home/user/rccl
COPY ./sources/rccl/  .

RUN cmake -S . -B build  -DAMDGPU_TARGETS="gfx90a;gfx942;gfx950"    \
    && cd build \
    && make -j 16 \
    && make install

# copy all of the reduction code to usr/local/bin

WORKDIR /home/user/scripts
# install RCCL Tests
RUN ./installRCCLTests.sh

WORKDIR /home/user
RUN  apt install -y amd-smi-lib

# Install the RVS
# see configs in  /opt/rocm/share/rocm-validation-suite/

# Install the rocm-examples

RUN apt-get -y update \
    &&  apt-get install -y libpci3 libpci-dev doxygen libyaml-cpp-dev \
        libglfw3-dev glslang-tools rocm-llvm-dev libboost-all-dev 

WORKDIR /home/user/rocm-examples

# git clone git@github.com:ROCm/rocm-examples.git
COPY ./sources/rocm-examples/ .


RUN cmake -S . -B build -DCMAKE_CXX_COMPILER=amdclang++ -DGPU_RUNTIME=HIP  -D GPU_RUNTIME=HIP  \
         -DOpenGL_FOUND=FALSE -DGLFW_FOUND=FALSE -DCMAKE_BUILD_TYPE=Release \
    && cd build \
    && make -j 16 \
    && make install

# copy all of the reduction code to usr/local/bin
RUN scp -p /home/user/rocm-examples/build/bin/Tutorials/reduction/reduct* /usr/local/bin/.

# Install hip-tests
# see https://github.com/ROCm/hip-tests

# git clone --recurse-submodules  git@github.com:ROCm/hip-tests.git

WORKDIR /home/user/hip-tests

COPY ./sources/hip-tests/  .

RUN export HIP_TESTS_DIR=$(pwd) \
    && rm -rf ./build \
    && mkdir -p build \
    && cd build \
    && cmake ../catch  -DHIP_PLATFORM=amd -DBUILDNAME=Hip-Tests  -DSITE=$(hostname)  -DOFFLOAD_ARCH_STR="--offload-arch=gfx90a --offload-arch=gfx942 "   \
    && echo "*** RUN MAKE **** " \
    && make -j$(nproc) build_tests \
    && make install 

# CTest

WORKDIR /home/user/Tests
COPY ./sources/Tests/  .

# prepare the tests from rocm-examples for execution via ctest

RUN  cd /home/user/Tests/rocm-examples \
     && if [ -f  ../CTestConfig.cmake ]; then \
           scp ../CTestConfig.cmake . \
        ;fi \
     && cmake -S. -B tests -D BUILDNAME=rocm-examples

# prepare the tests from agfhc for execution via ctest

RUN  cd /home/user/Tests/agfhc \
     && if [ -f  ../CTestConfig.cmake ]; then \
           scp ../CTestConfig.cmake . \
        ;fi \
     && cmake -S. -B tests -D BUILDNAME=agfhc

# prepare the tests from rvs for execution via ctest

RUN  cd /home/user/Tests/rvs \
     && if [ -f  ../CTestConfig.cmake ]; then \
           scp ../CTestConfig.cmake . \
        ;fi \
     && cmake -S. -B tests  -D BUILDNAME=rvs

# prepare the tests for rccl. if there is a ctest config file use it.


#RUN  cd /home/user/Tests/rccl  \
#     && if [ -f  ../CTestConfig.cmake ]; then \
#           scp ../CTestConfig.cmake . \
#        ;fi \
#     &&  cmake -S. -B tests -D BUILDNAME=rccl

RUN chown -R user /home/user/
RUN chmod -R a+rw /home/user/

USER user

CMD ["/bin/bash"]

