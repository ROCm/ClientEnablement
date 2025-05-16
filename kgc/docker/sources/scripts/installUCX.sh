#!/usr/bin/bash

#######################################################
# Install pre-requisites
#######################################################
apt install flex libtool  libstdc++-12-dev -y

#######################################################
# Compile UCX 
#######################################################
rm -rf /opt/ucx ucx
git clone --recursive -b v1.18.x  https://github.com/openucx/ucx.git
cd ucx || exit
./autogen.sh
mkdir build
cd build || exit
../contrib/configure-release --prefix=/opt/ucx --disable-logging --disable-debug --disable-assertions --disable-params-check --with-rocm=/opt/rocm --with-rc --with-ud --with-dc --with-dm --with-ib-hw-tm
make -j install
echo "/opt/ucx/lib" | tee /etc/ld.so.conf.d/ucx.conf
ldconfig
