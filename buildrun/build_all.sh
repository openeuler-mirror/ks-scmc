#!/bin/bash

set -e

g_basedir=$(cd $(dirname $0); pwd -P)
cd $g_basedir

# Compatibility:
#  "PG-NS-3.3-6C-2211-249003"
#  "PG-3.4-4A-2203-031747"

g_software_name=ks-scmc
declare -a support_archs=(x86_64 aarch64)

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
ARCH_TYPE=\$(cat /etc/.kyinfo | sed 's/ //g' | grep ^arch= | awk -F= '{ print \$2 }')
OS_VERSION="\$(cat /etc/.kyinfo | sed 's/ //g' | grep ^milestone | awk -F= '{print \$2}')"

if [ -d "\$1" ]; then
    CURR_PATH=\$1
else
    CURR_PATH=\$(pwd)
fi

if [[ "\${OS_VERSION}" && "\${ARCH_TYPE}" ]]; then
    echo "Current os is \${OS_VERSION}, arch is \${ARCH_TYPE}"
    KS_ALLOW_OS=(\$(cd \${CURR_PATH}; find . -maxdepth 1 -mindepth 1 -type d -not -empty -printf '%f\n'))
    echo "Support os: \${KS_ALLOW_OS[@]}"
    for KS_OS_NAME in \${KS_ALLOW_OS[@]}
    do
        if [[ "\$OS_VERSION" == "\$KS_OS_NAME"* ]];then
            RPM_PATH=\${CURR_PATH}/\${KS_OS_NAME}/\${ARCH_TYPE}
            break
        fi
    done
else
    echo "Cannot find os or arch"
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
    yumoption="--nogpgcheck --disablerepo=*"
    if [[ "\${OS_VERSION}" == "3.4-4"* ]]; then
        yumoption="\$yumoption --allowerasing"
    fi
    yum localinstall \$deppgs -y \$yumoption
    if [[ \$? -ne 0 ]];then
        echo "unknown error, please connact administrator, yumoption: \$yumoption"
        exit 1
    else
        echo "yum localinstall depend packages succeed"
    fi 
    if [[ "\${OS_VERSION}" == "3.3-6"* ]]; then
        rpm -qa | grep kernel-kek-4.18.0-147.ky3.kb23.pg && rpm -e kernel-kek-4.18.0-147.ky3.kb23.pg
    fi
fi

rpm -Uvh \$kspkgs --nodeps --force
echo "rpm -Uvh \$kspkgs --nodeps --force succeed"
echo -ne "\$info"

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
    ls $g_basedir/ks-run/$proj | while read ksver; do

        rm -f $g_basedir/ks-run/$proj/$ksver/*.sh
        process_install_shell $proj $ksver
        process_uninstall_shell $proj $ksver
        if [ -e $g_basedir/ks-run/$proj/$ksver ];then
            cmd="sh ./generate-run.sh $proj $ksver"
            echo $cmd
            $cmd
        fi
    done
done
