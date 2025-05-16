# AMD GPU MI-series Check Test Suite Instructions

## Introduction

This document provides basic instructions for running a suite of verification tests on an AMD GPU (MI300 or higher). These tests are designed to ensure the basic functionality and performance of the GPU.

## Prerequisites
The software is available using a Docker container and requires a Docker-compatible executions environment (docker, podman, singularity). Some software may need to be provided from AMD if the container needs to be rebuilt on site.   

The Operating System must be a supported version of Linux (tm).

## Setup

### ROCm (tm) Environment
Install the AMD ROCm software using version 6.3 or higher. The Docker container may run at a different ROCm level as long as AMD GPU compatibility exists between the releases.  A quick start guide may be found at   https://rocm.docs.amd.com/projects/install-on-linux/en/latest/install/quick-start.html

### Building the Docker Container (optional) 
In most cases, the Docker container has already been built and can be downloaded to the system as a tar file. 

If the container needs to be built, you will need:
1) The Dockerfile  (named Dockerfile at the time of writing)
2) The AMD provided AGFHC tar files (placed in docker/sources)
3) Access to github.com (for UCX, OpenMPI, rccl-tests, rocm-examples, repos)

In the docker/sources,  download the tests from github:

```
git clone git@github.com:Rocm/rocm-examples.git
git clone --recurse-submodules git@github.com:ROCm/rccl.git
git clone --recurse-submodules -b release/rocm-rel-6.4  git@github.com:ROCm/hip-tests.git
```

The command to build the container ( as amd_ubuntu_rocm640_kgc ) is:

```docker build -t amd_ubuntu_rocm640_kgc --build-arg USER_ID=$(id -u ${USER}) --build-arg GROUP_ID=$(id -g ${USER}) -f Dockerfile . ```

The image must then be saved to a file. For example: 

``` docker image save -o amd_ubuntu_rocm640_kgc.image  amd_ubuntu_rocm640_kgc```

## Running Verification Tests

### Command Usage

The verification tests have been wrapped using the Kit-Ware CTest (tm) program. This allows for consistent "pass/fail" information to be displayed as output. The installation could use CDash (tm) to  manage the test history by adding some small configuration files (not provided).

An example of running a test is:

```ctest --test-dir /home/user/Tests/rccl/tests``` 

Note that "--verbose" may be added to the ctest command to display more information about the running tests. 

The actual command line for this test is found in /home/user/Tests/rccl/CMakeLists.txt.  During the build time of the docker file, the CTest version of the tests was created using  ```cmake -S. -B tests``` within the /home/user/Tests/rccl  directory.

### Example
This example assumes that you are running the test in an interactive mode, start the container with the "-it" option:

```docker run -it --user $(id -u):$(id -g) -w /home/user --device /dev/kfd --device /dev/dri   --mount src="$(pwd)",target=/user,type=bind amd_ubuntu_rocm640_kgc /bin/bash```


RCCL:

```ctest --test-dir /home/user/Tests/rccl/tests``` 

AGFHC:

**MI300x specific tests:**

```ctest --test-dir /home/user/Tests/agfhc/tests -L MI300X_single_pass```

```ctest --test-dir /home/user/Tests/agfhc/tests -L MI300X_all_level1```

```ctest --test-dir /home/user/Tests/agfhc/tests -L MI300X_all_burnin_4h```

RVS (ROCm Validation Suite):

```ctest --test-dir /home/user/Tests/rvs/tests -L Basic```

```ctest --test-dir /home/user/Tests/rvs/tests -L Standard```

```ctest --test-dir /home/user/Tests/rvs/tests -L Check```

```ctest --test-dir /home/user/Tests/rvs/tests -L Stress```

ROCm-Examples:

```ctest --test-dir /home/user/Tests/rocm-examples/tests```

## Running Tests in Container

### Container
The tests may also be run in a batch mode by invoking the container with the "-it" parameter. For example:

``` docker run --user $(id -u):$(id -g) -w /home/user --device /dev/kfd --device /dev/dri  --mount src="$(pwd)",target=/user,type=bind  amd_ubuntu_rocm640_kgc ctest --test-dir  /home/user/Tests/rccl/tests  --verbose ```

### SLURM
The container may also be run as part of a SLURM allocation. For example, a sbatch file could wrap around statements such as:

```
## Job Steps
srun echo "Start process"
srun hostname
date
docker load -i /home/images/kgc/amd_ubuntu_rocm640_kgc.image
date
docker images | head 
# Note run with userid that built the container (for example, uid = 1009)
date
docker run  \
        --user 1009:1009 \
        -w /home/user \
        --device /dev/kfd --device /dev/dri \
        --mount src="$(pwd)",target=/user,type=bind \
        amd_ubuntu_rocm640_kgc ctest --test-dir /home/user/Tests/agfhc/tests -L MI300X_single_pass  --verbose  --output-on-failure

date
srun echo "End process"
```
The sbatch file could then be run on a regular basis against machines in the cluster.


## Conclusion

Following these instructions will help you set up and run verification tests on your AMD GPU efficiently. Ensure all steps are followed carefully to avoid any issues during the testing process.
