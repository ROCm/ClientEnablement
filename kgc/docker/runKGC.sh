
docker run -it --user $(id -u):$(id -g) \
	--network=host \
	-w /home/user --device /dev/kfd --device /dev/dri \
      	--mount src="$(pwd)",target=/user,type=bind \
       	amd_ubuntu_rocm640_kgc /bin/bash
