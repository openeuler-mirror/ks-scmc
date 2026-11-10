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

struct CommonRespone {
    1: i32 status,
    2: string message
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

// 网卡信息
struct NetworkInterface {
    1: required string eth_name,    // 网卡名称
    2: required string ip,          // 网卡ip地址
    3: required string mac,         // 网卡mac地址
    4: required string gateway,     // 网卡网关
    5: required string mask         // 网卡子网掩码
}

struct NetworkInterfaceWithAgentNode {
    1: required string eth_name,    // 网卡名称
    2: required string ip,          // 网卡ip地址
    3: required string mac,         // 网卡mac地址
    4: required string gateway,     // 网卡网关
    5: required string netmask         // 网卡子网掩码
}

struct ContainerInfo {
    1: required string name,        // 容器名称
    2: required i32 stat, // 容器状态
    3: required string uuid,        // 容器ID
    4: required double cpu,        // cpu阈值，支持小数
    5: required double cpu_usage,   // cpu使用率，百分数(0-100)
    6: required i32 mem_usage,      // 内存使用量(MB)
    7: required i32 mem_total,      // 内存总量(MB)
    8: required double disk_usage,  // 磁盘使用量(GB)
    9: required i32 disk_total,     // 磁盘总量(GB)
    10: required double disk_rbw,   // 磁盘读速度(MB/s)
    11: required double disk_wbw,   // 磁盘写速度(MB/s)
    12: required double net_rbw,    // 网络下载速度(MB/s)
    13: required double net_wbw,    // 网络上传速度(MB/s)
    14: required string image,      // 使用的镜像名称
    15: required string version,    // 使用的镜像版本
    16: required string ha,           // ha使能状态
    17: required i32 level,         // 模板等级
    18: required list<NetworkInterface> nets // 容器多网卡信息
}

struct GetAllContainersResponse {
    1: required i32 status,        // 容器名称
    2: required string message, // 容器状态
    3: required list<ContainerInfo> containers,        // 容器ID
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

struct GetImagesListResponse {
    1: list<string> imageList,
    2: i32 ret_code
}

struct Version {
    // 1: string linxc,
    // 2: string linxc_ovs,
    // 3: string linxc_agent,
    // 4: string integrated
}

struct NodeVersion {
    1: string name,
    2: string uuid,
    3: Version versions
}

struct GetVersionsV3Response {
    1: i32 status,
    2: string message,
    3: list<NodeVersion> nodes
}

enum ReadType {
    ALL = 0,
    READED = 1,
    UNREAD = 2
}

enum EntityType {
    ALL = 0,
    NODE_WARNING = 1,
    CONTAINER_WARN = 2
}

struct GetAlarmCond {
	1: string node_name,		//获取指定节点上的告警信息，默认包括节点告警与该节点所有容器的告警
	2: i8 read_type,			//0或其他:获取所有告警（已读+未读）；1：仅获取已读告警；2：仅获取未读告警
	3: i8 entity_type,			//0或其他:获取所有告警；1:仅获取节点告警；2:仅获取容器告警
	4: i32 alarm_type,			//获取指定类型的告警,如不指定，则传-1
	5: string container_name,	//获取指定容器的告警信息，如指定了名称，则默认仅获取该容器告警
	6: i64 start_time,			//获取时间区间内的告警，指定起始时间，传递参数为unix时间戳，精确到秒，如:1602578163
	7: i64 end_time,			//获取时间区间内的告警，指定截止时间，传递参数为unix时间戳，精确到秒，如:1602578163
	8: i32 nums,				//单页获取告警的条数，如该参数小于等于0,则默认为10
	9: i32 page,				//指定获取的页码，如该参数小于等于0，则默认为1，默认返回第一页，如指定数字超过总页码数，则返回最后一页告警
	10: bool asc				//false:告警信息降序排列（根据创建时间）；true:告警信息升序排列（根据创建时间）
}

struct CondAlarms {
    1: i32 id,
    2: i32 status,
    3: string message,
    4: i32 alarm_type,
    5: string c_time,
    6: string u_time
}

struct CondGetAlarmListResponse {
    1: i32 status,
    2: string message,
    3: i32 total,
    4: i32 nums,
    5: i32 pages,
    6: i32 page,
    7: list<CondAlarms> h_alarms,
    8: list<CondAlarms> c_alarms
}

struct AgentNode {
    1: string name,
    2: string uuid,
    3: string ip,
    4: list<NetworkInterfaceWithAgentNode> nets,
    5: string desc,
    6: i32 stat,
    7: double cpu_usage
    8: double cpus,
    9: double mem_usage,
    10: double mem_total,
    11: double disk_usage,
    12: double disk_total,
    13: double drbd_usage,
    14: double drbd_total,
    15: double net_rbw,
    16: double net_wbw,
    17: double disk_rbw,
    18: double disk_wbw,
    19: double drbd_rbw,
    20: double drbd_wbw,
    21: i32 container_nums,
    22: i32 online_container_nums,
    23: i32 max_containers,
    24: i32 alarm_nums
}

struct GetAgentNodesResponse {
    1: i32 status,
    2: string message,
    3: list<AgentNode> nodes
}

// ================ 服务接口 ================
service KsScmcClient {
    string LoginV3(1: string user, 2: string passwd, 3: string node_ip),
    string GetVersionsV3(1:string session_id),
    string GetAgentNodesV3(1:string session_id),
    string InstallContainer(1: string node_name, 2: string image, 3: string ver, 4: string name, 5: string desc),
    // 容器生命周期管理
    // i32 installContainer(1: string node_nameorip, 2: string image_name, 3: string image_ver, 4: string container_name),
    string StartContainer(1: string node_name, 2: string name),
    string StartContainerV3(1:string session_id, 2:string cid),
    // i32 startContainer(1: string node_nameorip, 2: string container_name),
    string StopContainer(1: string node_name, 2: string name),
    string StopContainerV3(1:string session_id, 2:string cid),
    // i32 stopContainer(1: string node_nameorip, 2: string container_name),
    string RestartContainer(1:string node_name, 2:string name),
    string UninstallContainer(1: string node_name, 2: string name),
    // i32 uninstallContainer(1: string node_nameorip, 2: string container_name),

    // 状态查询
    // todo: 此接口对应的 thrift层需要确认
    i32 getContainerStatus(1: string node_nameorip, 2: string container_name),
    // todo: 此接口对应的 thrift层需要确认
    GetContainerMetricsResponse getContainerMetrics(1: string node_nameorip, 2: string container_name),
    string GetContainerImageVers(1: string node_name, 2: string name),
    GetImageVer4ContainerResponse getImageVer4Container(1: string node_nameorip, 2: string container_name),
    // todo: 此接口对应的 thrift层需要确认
    i32 getContainer(1: string node_name, 2: string container_name),

    // 更新操作
    string UpdateContainer(1: string node_name, 2: string container_name, 3: string new_ver)
    // i32 updateContainer(1: string node_name, 2: string container_name, 3: string new_ver),

    // 批量查询
    string GetAlarmList(1: string node_name),
    // GetAlarmListResponse getAlarmList(1: string node_name),
    // todo: 此接口对应的 thrift层需要确认
    GetAllNodeListResponse getAllNodeList(),
    string GetAllContainers(1: string node_name), // 存疑，待确认
    string GetAllContainersV2(1:string nid),
    string GetAllContainersV3(1:string session_id, 2:string nid),
    GetAllContainerListResponse getAllContainerList(1: string node_name),
    string CondGetAlarmListV3(1:string session_id, 2:GetAlarmCond cond),
    string SetAlarmReadV3(1:string session_id, 2:string node_name, 3:i32 id),

    // 网络配置
    string GetContainerNetwork(1: string node_nameorip, 2: string container_name),
    // GetContainerNetworkResponse getContainerNetwork(1: string node_nameorip, 2: string container_name),
    // todo: 此接口对应的 thrift层需要确认
    GetContainerNetRulesResponse getContainerNetRules(1: string node_name, 2: string container_name),
    // todo: 此接口对应的 thrift层需要确认
    i32 addContainerIpPort(1: string node_nameorip, 2: string container_name, 3: string ip_addr, 4: i32 port, 5: string net_proto),
    // todo: 此接口对应的 thrift层需要确认
    i32 addContainerReqIpPort(1: string node_name, 2: string container_name, 3: string ip_addr, 4: i32 port, 5: string net_proto),

    // 环境配置
    string AddContainerEnv(1: string node_nameorip, 2: string container_name, 3: string env_name, 4: string env_value),
    // i32 addContainerEnv(1: string node_nameorip, 2: string container_name, 3: string env_name, 4: string env_value),
    // todo: 此接口对应的 thrift层需要确认
    i32 addContainerLocalVolume(1: string node_nameorip, 2: string container_name, 3: string in_dir, 4: string out_dir, 5: AccessAuth access_auth),
    string SetContainerDisplayV2(1: string node_name, 2: string container_name, 3: bool enabled, 4: string display),
    // i32 setContainerDisplayV2(1: string node_name, 2: string container_name, 3: bool enabled, 4: string display),

    // 资源限制
    string SetContainerQuota(1: string node_name, 2: string container_name, 3: ContainerLimit containerLimit)
    // i32 setContainerQuota(1: string node_name, 2: string container_name, 3: ContainerLimit containerLimit)

        /**
     * 设置容器是否开启特权
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @param isPrivileged 是否开启特权(true为开启)
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    i32 setContainerPrivileged(1: string node_ip, 2: string container_name, 3: bool isPrivileged),

    /**
     * 设置容器内核参数
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @param sysctl_name 内核参数名(如"kernel.sem")
     * @param sysctl_value 内核参数值(如"512 32000 100 256")
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    i32 setContainerSysctls(1: string node_ip, 2: string container_name,
                           3: string sysctl_name, 4: string sysctl_value),

    /**
     * 设置容器/etc/hosts中的host信息
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @param container_extrahosts host信息数组(如["docker:127.0.0.1","docker:127.0.0.1"])
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    i32 setContainerExtraHosts(1: string node_ip, 2: string container_name,
                             3: list<string> container_extrahosts),

    /**
     * 获取节点上的镜像列表
     * @param node_ip 节点ip
     * @param image_list 镜像信息列表(输出参数)
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    GetImagesListResponse getImagesList(1: string node_ip),

    /**
     * 导入镜像到节点
     * @param node_ip 节点ip
     * @param tarPath 镜像tar文件路径
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    i32 loadImage(1: string node_ip, 2: string tarPath),

    /**
     * 暂停容器
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    i32 pauseContainer(1: string node_ip, 2: string container_name),

    /**
     * 恢复暂停的容器
     * @param node_ip 节点ip
     * @param container_name 容器名称
     * @return 0成功，1失败
     */
    // todo: 此接口对应的 thrift层需要确认
    i32 unpauseContainer(1: string node_ip, 2: string container_name)
}

// ================ 异常定义 ================
exception OperationFailed {
    1: i32 error_code,
    2: string reason
}
