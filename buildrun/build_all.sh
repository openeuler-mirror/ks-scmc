#!/bin/bash

set -e

g_basedir=$(cd $(dirname $0); pwd -P)
cd $g_basedir

g_software_name=ks-scmc
support_archs=(x86_64 aarch64)
ALLOWED_OS_VERSION="3.3-6C"

function process_install_shell()
{
    local projname=$1
    local osversion=$2

    installsh=$g_basedir/ks-run/$projname/$osversion/install-${projname}.sh
    echo "process install shell: $installsh"
    cat > $installsh <<EOF
#!/bin/bash

# check the os version
# has checked in ks-run/install.sh

# check arch type
ARCH_TYPE=\$(cat /etc/.kyinfo | grep "arch =" | awk -F ' ' '{ print \$3 }')

if [ -d "\$1" ]; then
    CURR_PATH=\$1
else
    CURR_PATH=\$(pwd)
fi

if [ "\${ARCH_TYPE}" ]; then
    echo "Current arch is \${ARCH_TYPE}"
    RPM_PATH=\${CURR_PATH}/\${ARCH_TYPE}
else
    echo "Cannot find arch"
    exit 1
fi

if [ -d "\${RPM_PATH}" ]; then
    echo "The rpm path \${RPM_PATH}"
else
    echo "Cannot find path \${RPM_PATH}"
    exit 1
fi

# install pkg one by one, mariadb-server cannot be started
cd \$RPM_PATH

deppgs=\$(ls | grep -v '^$g_software_name')
kspkgs=\$(ls | grep -E "^$g_software_name")
if [[ ! \$deppgs == "" ]];then
    sudo yum localinstall \$deppgs -y --nogpgcheck --disablerepo=*
    if [[ \$? -ne 0 ]];then
        echo "unknown error, please connact administrator"
        exit 1
    fi 

    rpm -qa | grep kernel-kek-4.18.0-147.ky3.kb23.pg && rpm -e kernel-kek-4.18.0-147.ky3.kb23.pg
fi

sudo rpm -Uvh \$kspkgs --nodeps --force

cd -

EOF

}

function process_uninstall_shell()
{
    local projname=$1
    local osversion=$2

    uninstallsh=$g_basedir/ks-run/$projname/$osversion/uninstall-${projname}.sh
    echo "process uninstall shell: $uninstallsh"
    cat > $uninstallsh <<EOF
#!/bin/bash

shelldir=\$(cd "\$(dirname "\$0")";pwd)
shellname=\$(basename "\$0")
kspkgs="\$(rpm -qa | grep -E '^$g_software_name')"

echo "sudo rpm -e \$kspkgs --nodeps"
sudo rpm -e \$kspkgs --nodeps

systemctl disable cadvisor.service --now
systemctl disable opensnitch.service --now

echo "sudo rm -rf \$shelldir/\$shellname"
sudo rm -rf \$shelldir/\$shellname

EOF

}

ls $g_basedir/ks-run | grep ^$g_software_name | while read proj; do
    ls $g_basedir/ks-run/$proj | while read osver; do

        rm -f $g_basedir/ks-run/$proj/$osver/*.sh
        process_install_shell $proj $osver
        process_uninstall_shell $proj $osver

        for arch in ${support_archs[*]};do
            if [ -e $g_basedir/ks-run/$proj/$osver/$arch ];then
                cmd="sh ./generate-run.sh $proj $osver $arch $ALLOWED_OS_VERSION"
                echo $cmd
                $cmd
            fi
        done
    done
done
