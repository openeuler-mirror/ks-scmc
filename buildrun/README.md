
1 config repo

# Kylin-Base.repo
#

[base]
name=KylinSec OS-$releasever - Base
gpgcheck=0
baseurl=http://192.168.120.17/kojifiles/repos/KY3.3-6-PG-build/latest/x86_64/
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-kylin-release

[docker]
name=KylinSec OS-$releasever - docker
gpgcheck=0
baseurl=http://192.168.120.17/kojifiles/repos/KY3.3-9-docker-ce-20-build/latest/x86_64/
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-kylin-release

[epel]
name=KylinSec OS-$releasever - Epel
gpgcheck=0
baseurl=http://192.168.120.17/kojifiles/repos/KY3.3-EPEL-build/latest/x86_64/
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-kylin-release

[license]
name=KylinSec OS-$releasever - License
gpgcheck=0
baseurl=http://192.168.120.17/kojifiles/repos/KY3.3-6-PG-ks-license-2.6-build/latest/x86_64/
gpgkey=file:///etc/pki/rpm-gpg/RPM-GPG-KEY-kylin-release

2 change /etc/yum.conf make keepcache=1

3 get depend packages
yum install docker-ce coreutils bash socat systemd lsyncd rsync xinetd keepalived ks-license-client ks-license-gui-client  

4 build run 
