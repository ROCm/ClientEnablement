
# Testing 

### Testing using the rocm-validation-suite

The ROCm-validation-suite (RVS) should be installed as part of the ROCm installation process. It uses the rvs program located in the ROCm installation directory (e.g. /opt/rocm/bin), and configs in  /opt/rocm/share/rocm-validation-suite/.

RVS is also available from https://github.com/ROCm/ROCmValidationSuite

RVS should already be installed on the system (e.g. /opt/rocm/bin/rvs). It can also be installed from source on github. Building from source requires these basic steps.

```
   git clone https://github.com/ROCm/ROCmValidationSuite.git
   sudo apt-get -y update && sudo apt-get install -y libpci3 libpci-dev doxygen unzip cmake git libyaml-cpp-dev
   cd ROCmValidationSuite/

   Note: Edit CMakeLists.txt and remove any unnecessary architectures from HCC_CXX_FLAGS

   cmake -B ./build -DROCM_PATH=/opt/rocm  -DCMAKE_INSTALL_PREFIX=/opt/rocm -DCPACK_PACKAGING_INSTALL_PREFIX=/opt/rocm
   make -C ./build/
   cd build
   make package
   sudo dpkg -i rocm-validation-suite*.deb
```

Verify that rvs is operational by running a test

```
  cd build/bin  # or  cd /opt/rocm/bin 
   ./rvs -c conf/gst_single.conf
```

Prepare the tests from rvs for execution via ctest

Build the tests using the CMakeList.txt file in sources/Test/rvs.

The default to run the MI300x tests and you can update the CMakeLists.txt
file for you specific architecture. Additional tests may be added as desired.

``` 
  cd sources/Tests/rvs 
  cmake -S. -B tests  -D BUILDNAME=rvs

  cd tests
  ctest -N         # list the tests
  ctest -L Basic   # run just the Basic tests
```

### Testing using HIP Examples

The HIP Examples repo has been deprecated and replaced by the ROCm Examples repo. The HIP Examples code is still valuable as a simple, easy set of "smoke" tests for the basic functionality of the system. It does not have an associated CMake environment and uses basic makefiles for compiles. A CTest wrapper could be added to the code but is not provided in this repo.

```
  git clone https://github.com/ROCm/HIP-Examples.git
  cd HIP-Examples
  ./test_all.sh
```

### Testing using ROCM Examples

The ROCm Examples repo contains over 100 tests that cover a range of applications and APIs. The examples can be built and tested as follows

```
  git clone https://github.com/ROCm/rocm-examples.git
  cd rocm-examples
  cmake -S . -B build -D BUILDNAME=rocm-examples  # (on ROCm) 
  cmake --build build
  cmake --install build --prefix install
```

```
  cd build
  ctest
```

The build directory can be moved to a more permanent location if desired.
In the example above, you could install the executables to /usr/local/bin using 

```
  cmake --install build --prefix /usr/local 
```

A customize CMakeLists.txt file is available in sources/Tests/rocm-examples that uses the code from /usr/local/bin

``` 
  cd sources/Tests/rocm-examples
  cmake -S . -B tests
  cd tests
  ctest
```

###  Testing using RCCL 

The RCL tests are based on https://github.com/ROCm/rccl-tests.  See that location for additional information on building your own RCCL tests.

A copy the RCCL tests are in the sources/Tests/rccl-test subdirectory.

Build the tests using the install.sh script 

```
   cd rccl-tests
   ./install.sh
```

Testing

An example of using RCCL tests is available in the sources/Tests/rccl subdirectory.

```
   cd sources/Tests/rccl
   cmake -S. -B tests -D BUILDNAME=rccl
```

This creates a tests subdirectory that contains a file that can be used with CTest.

```
   cd tests
   ctest
```

Reports of pass/fails will be recorded. 

### CDash

The output from CTest (optionally) may be saved to a CDash dash board. See kitware.com for how to install a CDash dash board.

To enable the CTest output to be stored on CDash, create a file named CTestConfig.cmake that looks like:

```
set(CTEST_PROJECT_NAME "MIC")
set(CTEST_NIGHTLY_START_TIME "01:00:00 UTC")

set(CTEST_DROP_METHOD "http")
set(CTEST_DROP_SITE "10.200.116.31")
set(CTEST_DROP_LOCATION "/cdash/submit.php?project=MIC")
set(CTEST_DROP_SITE_CDASH TRUE)
```
Customize the IP Address for CTEST_DROP_SITE for your own dash board and change the project (MIC in this example) to your own project name. Alternatively, create your CTestConfig.cmake file from within your project's settings on the CDash server, under the "Miscellaneous" tab

The CTestConfig.cmake should be placed in the top-level directory of your CMake project.
