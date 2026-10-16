#/bin/bash
export KS_PATH="$1"
export PROJECK_NAME="$2"
export KS_ALLOW_OS="$3"

if [ "$KS_PATH" == "" ]; then
        echo "please check projeck tar is it correct!"
        echo "$KS_PATH"
        exit 1
fi

if [ "$PROJECK_NAME" == "" ]; then
        echo "please input projeck name!"
        echo "$PROJECK_NAME"
        exit 1
fi

VER_PROJECK_NAME=`ls ${KS_PATH}/${PROJECK_NAME}`
if [ "$VER_PROJECK_NAME" == "" ]; then
	echo "ks project is non-existent"
	echo "$PROJECK_NAME"
	exit 1
fi

# find os version
OS_VERSION=`cat /etc/.kyinfo | grep ^milestone | awk -F ' ' '{ print $3 }'`
echo "OS version:${OS_VERSION}"

ALLOWED_OS_VERSION="$KS_ALLOW_OS"
if [[ ! "$OS_VERSION" == "$ALLOWED_OS_VERSION"* ]];then
    echo "OS version mismatched, Allowed OS Version:$ALLOWED_OS_VERSION"
    exit 1
fi

DEST_DIR="$VER_PROJECK_NAME"
if [ -d ${KS_PATH}/${PROJECK_NAME}/${DEST_DIR} ]; then
	chmod +x ${KS_PATH}/${PROJECK_NAME}/${DEST_DIR}/install-${PROJECK_NAME}.sh && ${KS_PATH}/${PROJECK_NAME}/${DEST_DIR}/install-${PROJECK_NAME}.sh ${KS_PATH}/${PROJECK_NAME}/${DEST_DIR}

	chmod +x ${KS_PATH}/${PROJECK_NAME}/${DEST_DIR}/uninstall-${PROJECK_NAME}.sh && mv -f ${KS_PATH}/${PROJECK_NAME}/${DEST_DIR}/uninstall-${PROJECK_NAME}.sh /usr/bin/${PROJECK_NAME}-uninstall

else
       	echo "Cannot find path:${KS_PATH}/${PROJECK_NAME}/${DEST_DIR}"
fi

