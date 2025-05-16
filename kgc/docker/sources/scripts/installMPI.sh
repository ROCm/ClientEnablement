#############################################################
# Compile OpenMPI 
#############################################################
rm -rf /opt/ompi/ ompi/
git clone --recursive -b v5.0.7  https://github.com/open-mpi/ompi.git
cd ompi || exit
./autogen.pl
mkdir build
cd build || exit
../configure --prefix=/opt/ompi --with-ucx=/opt/ucx --enable-mca-no-build=btl-uct
make -j install
# Add Broadcom Thor NIC devID (0x1750) and Thor2 NIC devID (0x1760) to mca-btl-openib-device-params.ini file
# sed -i '/0x16f0,0x16f1/ s/$/,0x1750,0x1760/' /opt/ompi/share/openmpi/mca-btl-openib-device-params.ini
echo "/opt/ompi/lib" | tee /etc/ld.so.conf.d/ompi.conf
ldconfig
