#!/bin/bash
export KS_PROJECK_NAME=$1
export KS_PROJECK_VERSION=$2
export KS_ALLOW_ARCH=("x86_64")

CURR_PATH=$(cd $(dirname $0); pwd -P)

if [ "$KS_PROJECK_NAME" == "" ]; then
        echo "please input ks projeck name!"
        echo "./generate-run.sh [KS_PROJECK_NAME] [KS_PROJECK_VERSION]"
        exit 1
fi

if [ "$KS_PROJECK_VERSION" == "" ]; then
        echo "please input ks projeck version!"
        echo "./generate-run.sh [KS_PROJECK_NAME] [KS_PROJECK_VERSION]"
        exit 1
fi

SRC_PATH=$CURR_PATH/ks-run/$KS_PROJECK_NAME/$KS_PROJECK_VERSION
ls -l $SRC_PATH > /dev/null 2>&1 || { echo "$SRC_PATH not exist"; exit 1; }
export KS_ALLOW_OS=($(cd ${SRC_PATH}; find . -maxdepth 1 -mindepth 1 -type d -not -empty -printf '%f\n'))
if [ ${#KS_ALLOW_OS[@]} -eq 0 ]; then
        echo "$SRC_PATH is empty!"
        exit 1
fi

echo "allow os: ${KS_ALLOW_OS[@]}"
for KS_OS_NAME in ${KS_ALLOW_OS[@]}
do
    for KS_ARCH_NAME in ${KS_ALLOW_ARCH[@]}
    do
        RPM_PATH=$SRC_PATH/${KS_OS_NAME}/${KS_ARCH_NAME}
        echo ${RPM_PATH}
        if [ ! -e ${RPM_PATH} ];then
            echo "${RPM_PATH} donot exist"
            exit 1
        fi
    done
done

RUN_PATH=$CURR_PATH/$KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")
TMP_PATH=$CURR_PATH/ks-run-tmp/$KS_PROJECK_NAME/$KS_PROJECK_VERSION
mkdir -p $TMP_PATH
cp -r $SRC_PATH/* $TMP_PATH
cp -r $CURR_PATH/ks-run/install.sh $CURR_PATH/ks-run-tmp/

readmefile=$RUN_PATH.readme
cat > $readmefile <<EOF

麒麟信安安全容器魔方安装部署包

1、支持平台
硬件架构：${KS_ALLOW_ARCH[@]}
操作系统：${KS_ALLOW_OS[@]}

2、使用说明
执行 sh $KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d").run 完成安装
执行 ${KS_PROJECK_NAME}-uninstall.sh 完成卸载

3、软件用到的依赖包，已经全部整理到 run 包内

EOF

for KS_OS_NAME in ${KS_ALLOW_OS[@]}
do
    for KS_ARCH_NAME in ${KS_ALLOW_ARCH[@]}
    do
        echo "${KS_OS_NAME}.${KS_ARCH_NAME} 依赖包：" >> $readmefile
        find $TMP_PATH/${KS_OS_NAME}/${KS_ARCH_NAME} -name *.rpm | while read line;do
            echo $(basename $line) >> $readmefile
        done
        echo "" >> $readmefile
    done
done

sed -i "s/KS_PROJECK_NAME/$KS_PROJECK_NAME/g" $CURR_PATH/ks-run.sh
sed -i "s/KS_ALLOW_OS/\"${KS_ALLOW_OS[*]}\"/g" $CURR_PATH/ks-run.sh
tar cvf $CURR_PATH/ks-run-tmp.tar.gz -C $CURR_PATH ks-run-tmp
cat $CURR_PATH/ks-run.sh $CURR_PATH/ks-run-tmp.tar.gz > $RUN_PATH.run
chmod +x $RUN_PATH.run
md5sum $RUN_PATH.run > $RUN_PATH.md5.txt
sed -i "s/$KS_PROJECK_NAME/KS_PROJECK_NAME/g" $CURR_PATH/ks-run.sh
sed -i "s/\"${KS_ALLOW_OS[*]}\"/KS_ALLOW_OS/g" $CURR_PATH/ks-run.sh
rm -rf $CURR_PATH/ks-run-tmp.tar.gz $CURR_PATH/ks-run-tmp

