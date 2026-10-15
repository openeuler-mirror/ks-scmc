#!/bin/bash
export KS_PROJECK_NAME=$1
export KS_PROJECK_VERSION=$2
export KS_ARCH_NAME=$3
export KS_ALLOW_OS=$4

CURR_PATH=$(cd $(dirname $0); pwd -P)

if [ "$KS_PROJECK_NAME" == "" ]; then
        echo "please input ks projeck name!"
        echo "./generate-run.sh [KS_PROJECK_NAME] [KS_PROJECK_VERSION] [KS_ARCH_NAME]"
        exit 1
fi

if [ "$KS_PROJECK_VERSION" == "" ]; then
        echo "please input ks projeck version!"
        echo "./generate-run.sh [KS_PROJECK_NAME] [KS_PROJECK_VERSION] [KS_ARCH_NAME]"
        exit 1
fi

if [ "$KS_ARCH_NAME" == "" ]; then
        echo "please input ks projeck arch!"
        echo "./generate-run.sh [KS_PROJECK_NAME] [KS_PROJECK_VERSION] [KS_ARCH_NAME]"
        exit 1
fi

mkdir -p $CURR_PATH/ks-run-tmp/$KS_PROJECK_NAME/$KS_PROJECK_VERSION
if [ ! -e $CURR_PATH/ks-run/$KS_PROJECK_NAME/$KS_PROJECK_VERSION/${KS_ARCH_NAME} ];then
	echo "$CURR_PATH/ks-run/$KS_PROJECK_NAME/$KS_PROJECK_VERSION donot exist"
	exit 1
fi
cp -r $CURR_PATH/ks-run/$KS_PROJECK_NAME/$KS_PROJECK_VERSION/${KS_ARCH_NAME} $CURR_PATH/ks-run-tmp/$KS_PROJECK_NAME/$KS_PROJECK_VERSION
cp -r $CURR_PATH/ks-run/$KS_PROJECK_NAME/$KS_PROJECK_VERSION/install-${KS_PROJECK_NAME}.sh $CURR_PATH/ks-run-tmp/$KS_PROJECK_NAME/$KS_PROJECK_VERSION
cp -r $CURR_PATH/ks-run/$KS_PROJECK_NAME/$KS_PROJECK_VERSION/uninstall-${KS_PROJECK_NAME}.sh $CURR_PATH/ks-run-tmp/$KS_PROJECK_NAME/$KS_PROJECK_VERSION
cp -r $CURR_PATH/ks-run/install.sh $CURR_PATH/ks-run-tmp/

readmefile=$CURR_PATH/$KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")-${KS_ARCH_NAME}.readme
cat > $readmefile <<EOF

麒麟信安安全容器魔方安装部署包

1、支持平台
硬件架构：$KS_ARCH_NAME
操作系统：$KS_ALLOW_OS

2、使用说明
执行 sh $KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")-${KS_ARCH_NAME}.run 完成安装
执行 uninstall-${KS_PROJECK_NAME}.sh 完成卸载

3、软件用到的依赖包，已经全部整理到 run 包内 

EOF

find $CURR_PATH/ks-run-tmp -name *.rpm | while read line;do
	echo $(basename $line) >> $readmefile
done

sed -i "s/KS_PROJECK_NAME/$KS_PROJECK_NAME/g" $CURR_PATH/ks-run.sh
tar cvf $CURR_PATH/ks-run-tmp.tar.gz -C $CURR_PATH ks-run-tmp
cat $CURR_PATH/ks-run.sh $CURR_PATH/ks-run-tmp.tar.gz > $CURR_PATH/$KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")-${KS_ARCH_NAME}.run
chmod +x $CURR_PATH/$KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")-${KS_ARCH_NAME}.run
md5sum $CURR_PATH/$KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")-${KS_ARCH_NAME}.run > $CURR_PATH/$KS_PROJECK_NAME-$KS_PROJECK_VERSION-$(date "+%Y%m%d")-${KS_ARCH_NAME}.md5
sed -i "s/$KS_PROJECK_NAME/KS_PROJECK_NAME/g" $CURR_PATH/ks-run.sh
rm -rf $CURR_PATH/ks-run-tmp.tar.gz $CURR_PATH/ks-run-tmp

