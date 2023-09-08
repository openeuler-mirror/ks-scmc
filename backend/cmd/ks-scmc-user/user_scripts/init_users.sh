# 创建三权(系统管理员 审计管理员 安全管理员)对应的角色和用户

# 根据需要修改服务器地址
SERVER_ADDR="127.0.0.1:10050"

# 1. 创建管理员角色
./ks-scmc-user -s ${SERVER_ADDR} -c create_role "$(cat create_sysadm_r.json)"
./ks-scmc-user -s ${SERVER_ADDR} -c create_role "$(cat create_secadm_r.json)"
./ks-scmc-user -s ${SERVER_ADDR} -c create_role "$(cat create_audadm_r.json)"

# 2. 获取角色列表
./ks-scmc-user -s ${SERVER_ADDR} -c list_role "{}"

# 3. 创建管理员用户(注意role_id字段 可能需要修改)
./ks-scmc-user -s ${SERVER_ADDR} -c create_user "$(cat create_sysadm.json)"
./ks-scmc-user -s ${SERVER_ADDR} -c create_user "$(cat create_secadm.json)"
./ks-scmc-user -s ${SERVER_ADDR} -c create_user "$(cat create_audadm.json)"

# 4. 获取用户列表
./ks-scmc-user -s ${SERVER_ADDR} -c list_user "{}"