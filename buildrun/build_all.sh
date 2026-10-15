#!/bin/bash

set -e

g_basedir=$(cd $(dirname $0); pwd -P)
cd $g_basedir

function process_install_shell()
{
    local projname=$1
    local osversion=$2

    installsh=$g_basedir/ks-run/$projname/$osversion/install-${projname}.sh
    echo "process install shell: $installsh"
    cat > $installsh <<EOF
#!/bin/bash

# check the os version
VERSION_KY=\$(cat  /etc/.kyinfo  | grep "milestone = $osversion")
if [ "\${VERSION_KY}" == "" ]; then
    echo "WARNING: This only can install on kylinsec $osversion"
    exit 1
fi


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

deppgs=\$(ls | grep -v '^ks-ha')
cat /etc/.kyinfo | grep 3.3-6D > /dev/null
if [[ \$? -eq 0 ]];then
    deppgs=\$(ls | grep -v '^ks-ha' | grep -v '^python3')
fi

kspkgs=\$(ls | grep -E "^ks-ha")
if [[ ! \$deppgs == "" ]];then
    sudo yum localinstall \$deppgs -y --nogpgcheck --disablerepo=*
    if [[ \$? -ne 0 ]];then
        echo "please make sure the repo is valid or remove all invalid repo"
        exit 1
    fi 
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
kspkgs="\$(rpm -qa | grep -E '^ks-ha')"

echo "sudo rpm -e \$kspkgs --nodeps"
sudo rpm -e \$kspkgs --nodeps

echo "sudo rm -rf \$shelldir/\$shellname"
sudo rm -rf \$shelldir/\$shellname

EOF

}

support_archs=(x86_64 aarch64)
#support_archs=(x86_64)

ls $g_basedir/ks-run | grep ^ks-ha | while read proj; do
    ls $g_basedir/ks-run/$proj | while read osver; do

        rm -f $g_basedir/ks-run/$proj/$osver/*.sh
        process_install_shell $proj $osver
        process_uninstall_shell $proj $osver

        for arch in ${support_archs[*]};do
            if [ -e $g_basedir/ks-run/$proj/$osver/$arch ];then
                cmd="sh ./generate-run.sh $proj $osver $arch"
                echo $cmd
                $cmd
            fi
        done
    done
done
