# 创建三权(系统管理员 审计管理员 安全管理员)对应的角色和用户

# 根据需要修改服务器地址
SERVER_ADDR="127.0.0.1:10050"
BIN="ks-scmc-user"

set -e

cd $(dirname $0)

echo "创建角色 sysadm_r ..."
${BIN} -s ${SERVER_ADDR} -c create_role "$(cat create_sysadm_r.json)"

echo "创建角色 secadm_r ..."
${BIN} -s ${SERVER_ADDR} -c create_role "$(cat create_secadm_r.json)"

echo "创建角色 audadm_r ..."
${BIN} -s ${SERVER_ADDR} -c create_role "$(cat create_audadm_r.json)"

echo "创建管理员用户 ..."
${BIN} -s ${SERVER_ADDR} -c create_admin "{}"