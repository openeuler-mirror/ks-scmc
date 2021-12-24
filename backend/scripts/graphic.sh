######################### env.sh
BASE_DIR="/var/containers"
CONTAINER_NAME="display"

SOCK_DIR=${BASE_DIR}/${CONTAINER_NAME}/socket
AUTH_FILE=${BASE_DIR}/${CONTAINER_NAME}/Xauthority

SOCK_MOUNT="/tmp/.X11-unix"
AUTH_MOUNT="/tmp/.Xauthority"

######################### run_container.sh 
source ./env.sh

set -x

mkdir -p ${SOCK_DIR}
touch ${AUTH_FILE}

docker run -d \
  -e DISPLAY=:0 \
  -e XAUTHORITY=${AUTH_MOUNT} \
  -v ${SOCK_DIR}:${SOCK_MOUNT} \
  -v ${AUTH_FILE}:${AUTH_MOUNT} \
  --device /dev/vdb:/dev/vdb \
  --name ${CONTAINER_NAME} \
  --hostname ${CONTAINER_NAME} \
  jess/gparted

######################### open_display.sh 
source ./env.sh

set -x

# Get the DISPLAY slot
DISPLAY_NUMBER=$(echo $DISPLAY | cut -d. -f1 | cut -d: -f2)

# X11 auth permission
AUTH_COOKIE=$(xauth list | grep "^$(hostname)/unix:${DISPLAY_NUMBER} " | awk '{print $3}')
xauth -f ${AUTH_FILE} add ${CONTAINER_NAME}/unix:0 MIT-MAGIC-COOKIE-1 ${AUTH_COOKIE}

# Proxy with the :0 DISPLAY
socat UNIX-LISTEN:${SOCK_DIR}/X0,fork TCP4:localhost:60${DISPLAY_NUMBER} &

# (re)start container
docker start ${CONTAINER_NAME}
