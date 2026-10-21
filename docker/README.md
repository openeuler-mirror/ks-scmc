docker 启动

docker run  -d  --network=host   -v /sys/fs/cgroup:/sys/fs/cgroup  -e PS1='\u@\h \w\$ '  --name scmc  --privileged  ks-scmc:1.1.1-5

docker exec  -it  scmc bash 后，进入deploy目录 ，修改hosts配置，执行：

ansible-playbook deploy.yml --skip-tags "install"
