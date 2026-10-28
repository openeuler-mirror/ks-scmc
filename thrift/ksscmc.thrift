namespace cpp ksscmc
namespace go ksscmc

// ================ 基础数据结构 ================
enum ContainerStatus {
    ONLINE = 0,
    OFFLINE = 1,
    HA_EXECUTING = 2,
    UNKNOWN = 3
}

enum NodeStatusType {
    NODE_ONLINE = 0,
    NODE_OFFLINE = 1
}

enum AlarmStatus {
    READ = 0,
    UNREAD = 1
}

enum AccessAuth {
    READ_ONLY = 0,
    READ_WRITE = 1
}

struct ContainerMetric {
    1: string container_name,
    2: string container_id,
    3: string node_name,
    4: ContainerStatus container_status,
    5: double cpu_usage,       // 百分数(0~100)
    6: i32 mem_usage,          // MB
    7: double disk_usage,      // GB
    8: double net_upload,      // MB/s
    9: double net_download     // MB/s
}

struct NodeStatus {
    1: string node_name,
    2: string node_id,
    3: string ip_addr,
    4: NodeStatusType status,
    5: double cpu_usage,       // 百分数(0-100)
    6: double cpu_total,       // CPU总核心数
    7: double mem_usage,       // MB
    8: double mem_total,       // MB
    9: double disk_usage,      // GB
    10: double disk_total,     // GB
    11: i32 container_nums,
    12: i32 container_max_nums
}

struct ContainerImage {
    1: string image_name,
    2: string inuse_version
}

struct ContainerAlarm {
    1: i32 id,
    2: AlarmStatus status,
    3: string message,
    4: string alarm_time,
    5: string container_name
}

struct ContainerNetInfo {
    1: string ip,
    2: string mask,
    3: string gateway,
    4: i32 port
}

struct ContainerNetRule {
    1: string ip,
    2: i32 port,
    3: string protocol  // tcp/udp/icmp
}

struct ContainerLimit {
    1: double cpu,        // CPU核心数
    2: i32 cpu_prio,      // 1-5
    3: i32 mem_hard,      // MB
    4: i32 mem_soft,      // MB
    5: i32 disk_size,     // GB
    6: i32 disk_rio,      // 次/s
    7: i32 disk_wio,      // 次/s
    8: double disk_rbw,   // MB/s
    9: double disk_wbw,   // MB/s
    10: i32 disk_prio,    // 1-5
    11: double net_rbw,   // MB/s
    12: double net_wbw    // MB/s
}

struct GetAllNodeListResponse {
    1: list<NodeStatus> nodelist,
    2: i32 ret_code
}

struct GetImageVer4ContainerResponse {
    1: ContainerImage containerImage,
    2: i32 ret_code
}

struct GetAlarmListResponse {
    1: list<ContainerAlarm> alarmlist,
    2: i32 ret_code
}

struct GetAllContainerListResponse {
    1: list<ContainerMetric> containerlist,
    2: i32 ret_code
}

struct GetContainerNetworkResponse {
    1: ContainerNetInfo containerNetInfo,
    2: i32 ret_code
}

struct GetContainerNetRulesResponse {
    1: list<ContainerNetRule> containerNetRules,
    2: i32 ret_code
}

struct GetContainerMetricsResponse {
    1: ContainerMetric containerMetric,
    2: i32 ret_code
}

// ================ 服务接口 ================
service KsScmcClient {
    // 容器生命周期管理
    i32 installContainer(1: string node_nameorip, 2: string image_name, 3: string image_ver, 4: string container_name),
    i32 startContainer(1: string node_nameorip, 2: string container_name),
    i32 stopContainer(1: string node_nameorip, 2: string container_name),
    i32 uninstallContainer(1: string node_nameorip, 2: string container_name),

    // 状态查询
    i32 getContainerStatus(1: string node_nameorip, 2: string container_name),
    GetContainerMetricsResponse getContainerMetrics(1: string node_nameorip, 2: string container_name),
    GetImageVer4ContainerResponse getImageVer4Container(1: string node_nameorip, 2: string container_name),
    i32 getContainer(1: string node_name, 2: string container_name),

    // 更新操作
    i32 updateContainer(1: string node_name, 2: string container_name, 3: string new_ver),

    // 批量查询
    GetAlarmListResponse getAlarmList(1: string node_name),
    GetAllNodeListResponse getAllNodeList(),
    GetAllContainerListResponse getAllContainerList(),

    // 网络配置
    GetContainerNetworkResponse getContainerNetwork(1: string node_nameorip, 2: string container_name),
    GetContainerNetRulesResponse getContainerNetRules(1: string node_name, 2: string container_name),
    i32 addContainerIpPort(1: string node_nameorip, 2: string container_name, 3: string ip_addr, 4: i32 port, 5: string net_proto),
    i32 addContainerReqIpPort(1: string node_name, 2: string container_name, 3: string ip_addr, 4: i32 port, 5: string net_proto),

    // 环境配置
    i32 addContainerEnv(1: string node_nameorip, 2: string container_name, 3: string env_name, 4: string env_value),
    i32 addContainerLocalVolume(1: string node_nameorip, 2: string container_name, 3: string in_dir, 4: string out_dir, 5: AccessAuth access_auth),
    i32 setContainerDisplayV2(1: string node_name, 2: string container_name, 3: bool enabled, 4: string display),

    // 资源限制
    i32 setContainerQuota(1: string node_name, 2: string container_name, 3: ContainerLimit containerLimit)

        /**
     * 设置容器是否开启特权
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @param isPrivileged 是否开启特权(true为开启)
     * @return 0成功，1失败
     */
    i32 setContainerPrivileged(1: string node_ip, 2: string container_name, 3: bool isPrivileged),

    /**
     * 设置容器内核参数
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @param sysctl_name 内核参数名(如"kernel.sem")
     * @param sysctl_value 内核参数值(如"512 32000 100 256")
     * @return 0成功，1失败
     */
    i32 setContainerSysctls(1: string node_ip, 2: string container_name,
                           3: string sysctl_name, 4: string sysctl_value),

    /**
     * 设置容器/etc/hosts中的host信息
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @param container_extrahosts host信息数组(如["docker:127.0.0.1","docker:127.0.0.1"])
     * @return 0成功，1失败
     */
    i32 setContainerExtraHosts(1: string node_ip, 2: string container_name,
                             3: list<string> container_extrahosts),

    /**
     * 获取节点上的镜像列表
     * @param node_ip 节点ip
     * @param image_list 镜像信息列表(输出参数)
     * @return 0成功，1失败
     */
    i32 getImagesList(1: string node_ip, 2: list<string> image_list),

    /**
     * 导入镜像到节点
     * @param node_ip 节点ip
     * @param tarPath 镜像tar文件路径
     * @return 0成功，1失败
     */
    i32 loadImage(1: string node_ip, 2: string tarPath),

    /**
     * 暂停容器
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @return 0成功，1失败
     */
    i32 pauseContainer(1: string node_ip, 2: string container_name),

    /**
     * 恢复暂停的容器
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @return 0成功，1失败
     */
    i32 unpauseContainer(1: string node_ip, 2: string container_name)
}

// ================ 异常定义 ================
exception OperationFailed {
    1: i32 error_code,
    2: string reason
}