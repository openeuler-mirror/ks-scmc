/*
* 设置容器阈值的结构
*/
struct ContainerQuota {
	1: double cpu,			//cpu数量，支持小数
	2: i8 cpu_prio,			//cpu优先级，1-5,5优先级最高
	3: i32 mem_hard,		//内存硬限制(MB)
	4: i32 mem_soft,		//内存软限制(MB)
	5: i32 disk_size,		//磁盘大小(GB)
	6: i32 disk_rio,		//磁盘读频率(次/s)
	7: i32 disk_wio,		//磁盘写频率(次/s)
	8: double disk_rbw,		//磁盘读速度(MB/s)
	9: double disk_wbw,		//磁盘写速度(MB/s)
	10: i8 disk_prio,		//磁盘优先级，1-5,5优先级最高
	11: double net_rbw,		//网络下载速度(MB/s)
	12: double net_wbw		//网络上传速度(MB/s)
}

/*
* 设置获取告警信息的结构
* 注：告警类型：
* 节点：0：节点离线；1：节点cpu超额；2：节点内存超额；3：节点系统磁盘超额；4：节点共享磁盘超额；5：节点上存在非法容器
* 容器：5001：容器被篡改；5002：容器对应的镜像丢失，不存在仓库中，无法下载重新安装；5003：容器cpu超额；5004：容器内存软限制超额；
* 5005:容器内存硬限制超额；5006：容器磁盘大小超额；5007：容器磁盘读速度超额；5008：容器磁盘写速度超额；5009：容器磁盘读IO超额；
* 5010：容器磁盘写IO超额；5011：容器网络下载速度超额；5012：容器网络上传速度超额
*/
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

/*
* 设置容器网卡的结构
*/
struct ContainerNetwork {
	1:string eth_name,		//容器网卡名称，eg:eth0,eth1
	2:string attach,		//容器网卡关联的平台网卡名称
	3:string ip,			//容器网卡Ip地址
	4:string mask,			//容器网卡掩码
	5:string gw,			//容器网卡网关地址
	6:string mac			//容器网卡mac地址
}

/*
* 容器添加磁盘的结构
*/
struct ContainerDisk {
	1:string name,			//磁盘名称
	2:string mount_point,	//容器内部挂载点,eg: /tmp
	3:i32 size				//磁盘大小,单位GB
}

/*
*容器HA配置结构
*/
struct ContainerHA{
	1:i64 ha_start_times,		//容器连续启动失败最大次数，默认0次（表示将无限制进行启动容器）；如果设置为0次，那么将在执行高可用过程中，只进行第一次异常退出告警；
	2:i64 container_timeout, 	//容器离线超时时间，默认为30秒；在线状态监控2秒一次，超时时间除以监控周期向上取整得到离线次数15次；每次启动之后，从在线第一次开始将进行在线计次，连续在线三次，则认为HA执行成功，退出HA；如果从启动之后没有达到连续三次在线，认为上次启动失败，则将继续连续监控到容器异常15次执行启动，依次类推；
	3:bool ha_alarm_type		//HA告警方式，默认在每次启动失败之后进行告警；可以设置为只在达到最大次数进行告警，那么就将忽略中间次数告警；另外，容器第一次异常退出会立即告警提示；
}

/*
*应用模板时的模板项目选择结构体
*/
struct TSelect{
	1:bool net,				//应用网卡
	2:bool ipc,				//应用ipc
	3:bool display,			//应用图形使能
	4:bool quota,			//应用配额
	5:bool net_control,		//应用网络控制
	6:bool net_process,		//应用网络进程控制
	7:bool env,				//应用环境变量
	8:bool share_dir,		//应用共享存储
	9:bool storage,			//应用存储磁盘
	10:bool process_whitelist,	//应用进程白名单
	11:bool protected_file,	//应用关键文件保护
	12:bool ha,				//应用高可用
	13:bool level,			//应用分级
}

/*
*获取审计日志的条件结构
*/
struct AuditLogCond{
    1:i32 nums,         //指定返回的条数，不指定该参数则默认返回10条
    2:i32 page,         //指定返回的当前页码，不指定该参数则默认返回第一页
    3:bool asc,         //指定排序方式 false:审计日志降序排列（根据日志生成时间）；true:审计日志升序排列（根据日志生成时间）
    4:i64 start_time,   //指定搜索起始时间
    5:i64 end_time,     //指定搜索终止时间
    6:string op_user,   //指定搜索的用户，不指定该参数则默认返回所有用户的审计日志
    7:i8 op_user_id,    //指定事件操作用户id
    8:i32 event_type,   //指定搜索的事件类型
    9:i32 event_desc,   //指定搜索的事件描述
    10:i8 op_type,      //指定搜索的操作类型
    11:i8 op_result,    //指定搜索的操作结果
    12:string nid,      //指定搜索指定节点下的审计日志
    13:string cid,      //指定搜索指定容器下的审计日志
    14:string tid,      //指定搜索指定模板下的审计日志
}

/*
*备份数据和还原数据时的选择结构体
*/
struct BackupSelect {
	1:bool net,				//备份网卡配置
	2:bool ipc,				//备份ipc配置
	3:bool display,			//备份图形使能配置
	4:bool quota,			//备份配额配置
	5:bool net_control,		//备份网络控制配置
	6:bool net_process,		//备份网络进程控制配置
	7:bool env,				//备份环境变量配置
	8:bool share_dir,		//备份共享存储配置
	9:bool storage,			//备份磁盘持久化配置
	10:bool process_whitelist,	//备份进程白名单配置
	11:bool protected_file,	//备份关键文件保护配置
	12:bool ha,				//备份高可用配置
	13:bool level,			//备份分级配置
	14:bool backup_img,		//备份镜像文件配置
	15:bool backup_disk,	//备份磁盘持久化数据
	16:bool backup_sharedir	//备份共享目录数据
}

/*------------------------------------------------接口------------------------------------------------------*/
/*
* 所有接口返回值均为json字符串，除获取信息的接口外，其余接口返回值均参考接口Login
* status：0,成功；1,失败
* 接口备注中：
* @Deprecated: 表示此方法已废弃、暂时可用，但以后此类或方法都不会再更新、后期可能带来兼容性问题或被删除，建议切换到新接口
*/
service Agent{
	/*(1)用户管理接口*/

	//登录
	/*
	* param1: 用户名
	* param2: 密码
	* param3: 登录节点的IP（使用vip进行访问）
	* desc: 该方法用于登录验证，验证通过后跳转页面，然后可使用其他接口
	* return: {"status":0,"message":" success"} 或者 {"status":1,"message":" failed"}
	* return: {"status":0,"message":"75d32bde-f69f-11eb-a29d-80ff028b7557"} 或者 {"status":1,"message":" failed"}
	*return: {"status":2, "message":"75d32bde-f69f-11eb-a29d-80ff028b7557"} (该返回值需要进行提醒：用户密码过期，请及时修改)
	*return: {"status":3, "message":"用户密码过期，请修改"} (该返回值需要进行提醒：用户密码过期，请修改 . 并再次调用登录接口进行验证.)
	*return: {"status":4, "message":"75d32bde-f69f-11eb-a29d-80ff028b7557"} (该返回值需要进行提醒： 用户密码验证成功. 并调用修改密码接口.)
	*return: {"status":5, "message":"二次验证失败"} (该返回值需要进行提醒： 用户密码验证失败)
	* 注:status：0,成功；1,失败
	*/
	string Login(1:string user, 2:string passwd, 3:string node_ip),

	//登录
	/*
	* param1: 用户名
	* param2: 密码
	* param3: 登录节点的IP（使用vip进行访问）
	* desc: 该方法用于登录验证，验证通过后跳转页面，然后可使用其他接口
	* return: {"status":0,"message":"75d32bde-f69f-11eb-a29d-80ff028b7557"} 或者 {"status":1,"message":" failed"}
	*return: {"status":2, "message":"75d32bde-f69f-11eb-a29d-80ff028b7557"} (该返回值需要进行提醒：用户密码过期，请及时修改)
	*return: {"status":3, "message":"用户密码过期，请修改"} (该返回值需要进行提醒：用户密码过期，请修改 . 并再次调用登录接口进行验证.)
	*return: {"status":4, "message":"75d32bde-f69f-11eb-a29d-80ff028b7557"} (该返回值需要进行提醒： 用户密码验证成功. 并调用修改密码接口.)
	*return: {"status":5, "message":"二次验证失败"} (该返回值需要进行提醒： 用户密码验证失败)
	* 注:status：0,成功；1,失败  //登录成功时返回用户session的id
	*/
	string LoginV3(1:string user, 2:string passwd, 3:string node_ip),

	//修改密码
	/*
	* param1: 用户名
	* param2: 密码
	* param3: 新密码
	*/
	string ChangePasswd(1:string user, 2:string passwd, 3:string new_passwd),

	//修改密码
	/*
	* param1: session的id
	* param2: 用户名
	* param3: 密码
	* param4: 新密码
	*/
	string ChangePasswdV3(1:string session_id, 2:string user, 3:string passwd, 4:string new_passwd),

	//用户验证
	/*
	* param1: session的id
	* param2: 用户名
	* param3: 密码
	* param4: 需要进行用户验证的当次功能操作id
	*/
	string ValidateUserRun(1:string session_id, 2:string user, 3:string passwd, 4:string action_id),

	//添加用户
	/*
	* param1: session的id
	* param2: 用户名
	* param3: 密码
	* param4: 用户描述信息
	*/
	string AddUser(1:string session_id, 2:string user, 3:string passwd, 4:string desc),

	//删除用户
	/*
	* param1: session的id
	* param2: 用户名
	*/
	string DeleteUser(1:string session_id, 2:string user),

	//编辑用户描述
	/*
	* param1: session的id
	* param2: 用户名
	* param3: 用户描述信息
	*/
	string EditUserDesc(1:string session_id, 2:string user, 3:string desc),

	//获取所有用户
	/*
	* param1: session的id
	* return:
	{
		"status":0,"message":"Get users success", "users":[
			{"id":1, "user","admin", "passwd":"xxxxxxxxxx", "role":"admin", "desc":"系统管理员", "skip_audit":false, "skip_pending":false}
		]
	}
	*/
	string GetUsers(1:string session_id),

	//解锁用户
	/*
	* param1: session的id
	* param2: 用户名
	*/
	string UnlockUser(1:string session_id, 2:string user),

	//设置自动删除审计日志允许的时间上限
	/*
	* param1: session的id
	* param2: 日志保存时间（单位:月）
	*/
	string SetDelAuditLogLimitTime(1:string session_id, 2:i32 limit_time),

	//获取审计日志
	/*
	* param1: session的id
	* param2: 审计日志搜索条件
	* return:
	{
		"status":0,
		"message":"获取审计日志成功",
		"total":2,										//日志总条数
		"nums":2,										//当前返回日志总条数
		"pages":1,										//总页码数
		"page":1,										//当前页码
		"logs":[{										//日志列表
            "id":1,										//日志id
            "op_user":"admin",							//事件操作用户
            "op_user_id":1,								//事件操作用户id
            "op_type":3,								//事件操作类型
            "op_result":1,								//事件操作结果
            "event_type":1,								//事件类型
            "event_desc":1,								//事件描述
            "event_time":"2020-08-26 09:22:08",			//事件执行时间
            "event_abs":"修改节点名称[node1 -> Node1]",	//摘要
            "entity_name":"节点[Node1]",					//对象实体名称
            "nid":"xxx-xxx-xxx-xxx",					//节点id
            "cid":"xxx-xxx-xxx-xxx",					//容器id
            "tid":"xxx-xxx-xxx-xxx",					//模板id
        }]
	}
	*/
	string GetAuditLog(1:string session_id, 2:AuditLogCond cond),

	/*(2)节点管理接口*/
	
	//添加节点
	/*
	* param1: 节点名称
	* param2: 节点ip地址
	* param3: 节点描述信息
	*/
	string AddAgentNode(1:string node_name, 2:string node_ip, 3:string node_desc),
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 节点ip地址
	* param4: 节点描述信息
	*/
	string AddAgentNodeV3(1:string session_id, 2:string node_name, 3:string node_ip, 4:string node_desc),

	//删除节点
	/*
	* param1: 节点名称
	*/
	string DelAgentNode(1:string node_name),
	//删除节点
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string DelAgentNodeV3(1:string session_id, 2:string node_name),

	//更新节点
	/*
	* param1: 节点名称
	* param2: 节点描述信息
	* desc: 目前只支持修改节点描述信息，节点名称用于确定被修改的节点
	*/
	string UpdateAgentNode(1:string node_name, 2:string node_desc),
	//更新节点
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 节点描述信息
	* desc: 目前只支持修改节点描述信息，节点名称用于确定被修改的节点
	*/
	string UpdateAgentNodeV3(1:string session_id, 2:string node_name, 3:string node_desc),

	//获取所有节点信息
	/*
	* desc：以json返回所有节点的信息，包含基础设置信息，节点在线状态，节点当前的使用率
	* return:
	{
		"status":0,
		"message":"Get nodes success",
		"nodes":[	//json数组
			{
				"name":"node1",										//节点名称
				"uuid":"92e8d428-7a6a-11eb-afde-0050568b82fa",		//节点uuid
				"ip":"172.30.24.101",								//节点Ip地址
				"nets":[	//节点多网卡信息
					{
						"eth_name":"eth0",					//网卡名称
						"ip":"172.30.25.101",				//网卡ip地址
						"mac":"00:50:56:c0:00:01",			//网卡mac地址
						"gateway":"172.30.24.254",			//网卡网关
						"netmask":"255.255.255.0"			//网卡子网掩码
					},
					{
						"eth_name":"eth1",					//网卡名称
						"ip":"172.30.25.101",				//网卡ip地址
						"mac":"00:50:56:c0:00:02",			//网卡mac地址
						"gateway":"172.30.25.254",			//网卡网关
						"netmask":"255.255.255.0"			//网卡子网掩码
					}
				],
				"desc":"描述信息",			//节点描述信息
				"stat":0,					//节点状态，0:在线，1:离线
				"cpu_usage":10.5,			//节点cpu使用率，百分数(0-100)，浮点数
				"cpus":24,					//节点cpus数量，cpu数量*核数*线程数，浮点数
				"mem_usage":500,			//节点内存使用量(MB)，浮点数
				"mem_total":2048,			//节点内存总量（MB)，浮点数
				"disk_usage":10,			//节点系统磁盘使用量(GB)，浮点数
				"disk_total":100,			//节点系统磁盘总量(GB)，浮点数
				"drbd_usage":100,			//节点共享磁盘使用量(GB)，浮点数
				"drbd_total":1000,			//节点共享磁盘总量(GB)，浮点数
				"net_rbw":10,				//节点网络下载速度(MB/s)，浮点数
				"net_wbw":10,				//节点网络上传速度(MB/s)，浮点数
				"disk_rbw":10,				//节点系统磁盘读速度(MB/s)，浮点数
				"disk_wbw":10,				//节点系统磁盘写速度(MB/s)，浮点数
				"drbd_rbw":10,				//节点共享磁盘读速度(MB/s)，浮点数
				"drbd_wbw":10,				//节点共享磁盘写速度(MB/s)，浮点数
				"container_nums":10,		//节点上容器的总数
				"online_container_nums":10,//节点上在线容器的总数
				"max_containers":10,		//该节点上支持的最大容器数量
				"alarm_nums":10				//节点上未读告警的数量
			}
		]
	}
	*/
	string GetAgentNodes(),
	//获取节点列表
	/*
	* param1: 用户session的id
	*/
	string GetAgentNodesV3(1:string session_id),

	//获取指定节点状态
	/*
	* param1: 节点名称
	* desc：以json返回指定节点的信息，包含基础设置信息，节点在线状态，节点当前的使用率
	* return: 
	{
		"status":0,
		"message":"Get node success",
		"nodes":{
			"name":"node1",									//节点名称
			"uuid":"92e8d428-7a6a-11eb-afde-0050568b82fa",	//节点uuid
			"nets":[	//节点多网卡信息
					{
						"eth_name":"eth0",				//网卡名称
						"ip":"172.30.25.101",			//网卡ip地址
						"mac":"00:50:56:c0:00:01",		//网卡mac地址
						"gateway":"172.30.24.254",		//网卡网关
						"netmask":"255.255.255.0"		//网卡子网掩码
					},
					{
						"eth_name":"eth1",				//网卡名称
						"ip":"172.30.25.101",			//网卡ip地址
						"mac":"00:50:56:c0:00:02",		//网卡mac地址
						"gateway":"172.30.25.254",		//网卡网关
						"netmask":"255.255.255.0"		//网卡子网掩码
					}
				],
			"ip":"172.30.24.101",		//节点Ip地址
			"desc":"描述信息",			//节点描述信息
			"stat":0,					//节点状态，0:在线，1:离线
			"cpu_usage":10.5,			//节点cpu使用率，百分数(0-100)，浮点数
			"cpus":24,					//节点cpus数量，cpu数量*核数*线程数，浮点数
			"mem_usage":500,			//节点内存使用量(MB)，浮点数
			"mem_total":2048,			//节点内存总量（MB)，浮点数
			"disk_usage":10,			//节点系统磁盘使用量(GB)，浮点数
			"disk_total":100,			//节点系统磁盘总量(GB)，浮点数
			"drbd_usage":100,			//节点共享磁盘使用量(GB)，浮点数
			"drbd_total":1000,			//节点共享磁盘总量(GB)，浮点数
			"net_rbw":10,				//节点网络下载速度(MB/s)，浮点数
			"net_wbw":10,				//节点网络上传速度(MB/s)，浮点数
			"disk_rbw":10,				//节点系统磁盘读速度(MB/s)，浮点数
			"disk_wbw":10,				//节点系统磁盘写速度(MB/s)，浮点数
			"drbd_rbw":10,				//节点共享磁盘读速度(MB/s)，浮点数
			"drbd_wbw":10,				//节点共享磁盘写速度(MB/s)，浮点数
			"container_nums":10,		//节点上容器的总数
			"online _container_nums":10,//节点上在线容器的总数
			"max_containers":10,		//该节点上支持的最大容器数量
			"alarm_nums":10				//节点上未读告警的数量
		}
	}
	*/
	string GetAgentNode(1:string node_name),
	//获取指定节点
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetAgentNodeV3(1:string session_id, 2:string node_name),

	//设置主机阈值
	/*
	* param1: 节点名称
	* param2: 节点cpu阈值设置，仅支持设置整数个cpu。如节点cpu核心数为24，则可以设置为12
	* param3: 节点内存阈值设置，仅支持设置整数，如1024，单位默认为MB
	* param4: 节点系统磁盘阈值设置，仅支持设置整数，如500，单位默认为GB
	* param5: 节点共享磁盘阈值设置，仅支持设置整数，如500，单位默认为GB
	* return: {"status":0,"message":" success"} 或者 {"status":1,"message":" failed"}
	*/
	string SetNodeQuota(1:string node_name, 2:i32 cpu_th, 3:i32 mem_th, 4:i32 disk_th, 5:i32 drbd_th),
	//设置主机阈值
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 节点cpu阈值设置，仅支持设置整数个cpu。如节点cpu核心数为24，则可以设置为12
	* param4: 节点内存阈值设置，仅支持设置整数，如1024，单位默认为MB
	* param5: 节点系统磁盘阈值设置，仅支持设置整数，如500，单位默认为GB
	* param6: 节点共享磁盘阈值设置，仅支持设置整数，如500，单位默认为GB
	*/
	string SetNodeQuotaV3(1:string session_id, 2:string node_name, 3:i32 cpu_th, 4:i32 mem_th, 5:i32 disk_th, 6:i32 drbd_th),

	//获取主机阈值
	/*
	* param1: 节点名称
	* desc: 获取指定节点设置的阈值信息
	* return:
	{
	"status":0,
	"message":"Get node quota success",
	"quota":
		{
		"cpu_th":12,		//节点cpu使用阈值，cpu个数，整数
		"mem_th":10240,		//节点内存使用阈值，10240MB
		"disk_th":1024,		//节点系统磁盘使用阈值,1024GB
		"drbd_th":2048,		//节点共享磁盘使用阈值,2048GB
		}
	}
	*/
	string GetNodeQuota(1:string node_name),
	//获取主机阈值
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* desc: 获取指定节点设置的阈值信息
	*/
	string GetNodeQuotaV3(1:string session_id, 2:string node_name),

	/*(3)节点安全接口*

	//添加节点关键文件保护
	/*
	* param1: 节点名称
	* param2: 关键文件，需要指定该文件在节点上的完整路径
	* param3: 删除权限，0：可删除，1：不可删除
	* param4: 修改权限，0：可修改，1：不可修改
	* desc: 添加关键文件保护，调用成功则即时生效
	*/
	string AddNodeProtectFile(1:string node_name, 2:string file, 3:i8 permit_del, 4:i8 permit_mod),
	//添加节点关键文件保护
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* param4: 删除权限，0：可删除，1：不可删除
	* param5: 修改权限，0：可修改，1：不可修改
	*/
	string AddNodeProtectFileV3(1:string session_id, 2:string node_name, 3:string file, 4:i8 permit_del, 5:i8 permit_mod),

	//删除节点关键文件保护
	/*
	* param1: 节点名称
	* param2: 关键文件，需要指定该文件在节点上的完整路径
	* desc: 删除关键文件保护，调用成功则即时生效
	*/
	string DelNodeProtectFile(1:string node_name, 2:string file),
	//删除节点关键文件保护
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	*/
	string DelNodeProtectFileV3(1:string session_id, 2:string node_name, 3:string file),

	//修改节点关键文件保护
	/*
	* param1: 节点名称
	* param2: 关键文件，需要指定该文件在节点上的完整路径
	* param3: 新关键文件，需要指定该文件在节点上的完整路径，如仅需修改权限位，则该路径指定与param2一致
	* param4: 删除权限，0：可删除，1：不可删除
	* param5: 修改权限，0：可修改，1：不可修改
	* desc: 修改关键文件保护，调用成功则即时生效
	*/
	string UpdateNodeProtectFile(1:string node_name, 2:string file, 3:string new_file, 4:i8 permit_del, 5:i8 permit_mod),
	//修改节点关键文件保护
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* param4: 新关键文件，需要指定该文件在节点上的完整路径，如仅需修改权限位，则该路径指定与param2一致
	* param5: 删除权限，0：可删除，1：不可删除
	* param6: 修改权限，0：可修改，1：不可修改
	*/
	string UpdateNodeProtectFileV3(1:string session_id, 2:string node_name, 3:string file, 4:string new_file, 5:i8 permit_del, 6:i8 permit_mod),

	//获取节点关键文件列表
	/*
	* param1: 节点名称
	* desc: 返回节点上设置的所有的关键文件及其权限
	* return:
	{
		"status":0,
		"message":"Get node protect files success",
		"files":[	//json数组
			{
				"path":"/home/file1",	//关键文件路径
				"permit_del":0,			//关键文件删除权限，0：可删除，1：不可删除
				"permit_mod":1			//关键文件修改权限，0：可修改，1：不可修改
			}
		]
	}
	*/
	string GetNodeProtectFiles(1:string node_name),
	//获取节点关键文件列表
	/*
	* param1: 用户session的id
	* param1: 节点名称
	*/
	string GetNodeProtectFilesV3(1:string session_id, 2:string node_name),

	/*(4)节点监控接口*/

	//获取节点统计信息
	/*
	* param1: 节点名称
	* param2: 开始时间，传递参数为unix时间戳，精确到秒，如:1602578163
	* param3: 结束时间，传递参数为unix时间戳，精确到秒
	* param4: 间隔时间，获取到的监控信息条数应为：nums = （end_time - start_time）/ interval
	* return:
	{
		"status":0,
		"message":"Get node statistics success",
		"cpu":[		//json数组
			0.5,			//cpu使用率，百分数(0-100),浮点数
			0.6
		],
		"mem":[
			500,			//节点内存使用量(MB)，浮点数
			600
		],
		"disk_rbw":[
			200,			//节点系统磁盘读速度(MB/s)，浮点数
			300
		],
		"disk_wbw":[
			200,			//节点系统磁盘写速度(MB/s)，浮点数
			300
		],
		"disk_rio":[
			10,				//节点系统磁盘读频率(次/s)，整数
			20
		],
		"disk_wio":[
			10,				//节点系统磁盘写频率(次/s)，整数
			20
		],
		"drbd_rbw":[
			200,			//节点共享磁盘读速度(MB/s)，浮点数
			300
		],
		"drbd_wbw":[
			200,			//节点共享磁盘写速度(MB/s)，浮点数
			300
		],
		"drbd_rio":[
			10,				//节点共享磁盘读频率(次/s)，整数
			20
		],
		"drbd_wio":[
			10,				//节点共享磁盘写频率(次/s)，整数
			20
		],
		"net_rbw":[
			200,			//节点网络下载速度(MB/s)，浮点数
			300
		],
		"net_wbw":[
			200,			//节点网络上传速度(MB/s)，浮点数
			300
		]
	}
	*/
	string GetNodeStatistics(1:string node_name, 2:i64 start_time, 3:i64 end_time, 4:i32 interval),
	//获取节点统计信息
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 开始时间，传递参数为unix时间戳，精确到秒，如:1602578163
	* param4: 结束时间，传递参数为unix时间戳，精确到秒
	* param5: 间隔时间，获取到的监控信息条数应为：nums = （end_time - start_time）/ interval
	*/
	string GetNodeStatisticsV3(1:string session_id, 2:string node_name, 3:i64 start_time, 4:i64 end_time, 5:i32 interval),
	
	//获取告警信息列表
	/*
	* param1: 节点名称
	* desc: 返回指定节点的所有告警信息，包括当前节点的告警信息，以及当前节点之上的容器的告警信息，
	* 默认获取最新50条未读告警信息。如未读告警数量小于50，则获取最新未读告警信息和最新已读告警共50条（保证所有未读告警获取到）。
	* return:
	{
		"status":0,
		"message":"Get alarms success",
		"h_alarms":[	//json数组，节点告警信息
			{
				"id":1,								//告警信息id
				"status":1,							//告警信息状态，0:已读，1:未读
				"message":"Alarm1",					//告警信息内容
				"c_time":"2020-08-26 09:09:09"		//告警时间
			},
			{
				"id":2,
				"status":0,
				"message":"Alarm2",
				"c_time":"2020-08-26 09:09:09"
			}
		],
		"c_alarms":[	//json数组，容器告警信息
			{
				"id":1,								//告警信息id
				"status":1,							//告警信息状态，0:已读，1:未读
				"message":"Alarm1",					//告警信息内容
				"c_time":"2020-08-26 09:09:09",		//告警时间
				"c_name":"APP1"						//该告警信息所属的容器名称
			},
			{
				"id":2,
				"status":0,
				"message":"Alarm2",
				"c_time":"2020-08-26 09:09:09",
				"c_name":"APP2"
			}
		]
	}
	*/
	string GetAlarmList(1:string node_name),
	//获取告警信息列表
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetAlarmListV3(1:string session_id, 2:string node_name),
	
	//根据指定条件获取告警信息列表
	/*
	* param1: 条件参数，参考设置获取告警信息的结构：struct GetAlarmCond
	* desc: 根据指定条件获取告警信息
	* return:
	{
		"status":0,
		"message":"Get alarms success",
		"total":4,									//总告警数
		"nums":4,									//当前返回总告警数
		"pages":4,									//总页码数
		"page":1,									//当前页码
		"h_alarms":[	//json数组，节点告警信息
			{
				"id":1,								//告警信息id
				"status":1,							//告警信息状态，0:已读，1:未读
				"message":"Alarm1",					//告警信息内容
				"alarm_type":0,						//告警类型,0为节点离线告警
				"c_time":"2020-08-26 09:09:09"		//告警创建时间
				"u_time":"2020-08-26 09:22:08"		//最近一次告警更新时间
			},
			{
				"id":2,
				"status":0,
				"message":"Alarm2",
				"alarm_type":1,						//告警类型,1为节点cpu超额告警
				"c_time":"2020-08-26 09:09:09"
				"u_time":"2020-08-26 09:22:08"
			}
		],
		"c_alarms":[	//json数组，容器告警信息
			{
				"id":1,								//告警信息id
				"status":1,							//告警信息状态，0:已读，1:未读
				"message":"Alarm1",					//告警信息内容
				"alarm_type":5001,					//告警类型,5001容器被篡改告警
				"c_time":"2020-08-26 09:09:09",		//告警创建时间
				"u_time":"2020-08-26 09:22:08"		//最近一次告警更新时间
				"c_name":"APP1"						//该告警信息所属的容器名称
			},
			{
				"id":2,
				"status":0,
				"message":"Alarm2",
				"alarm_type":5003,			//告警类型,5003为容器cpu超额告警
				"c_time":"2020-08-26 09:09:09",
				"u_time":"2020-08-26 09:22:08"
				"c_name":"APP2"
			}
		]
	}
	*/
	string CondGetAlarmList(1:GetAlarmCond cond),
	//根据指定条件获取告警信息列表
	/*
	* param1: 用户session的id
	* param2: 条件参数，参考设置获取告警信息的结构：struct GetAlarmCond
	*/
	string CondGetAlarmListV3(1:string session_id, 2:GetAlarmCond cond),
	
	//设置告警信息已读
	/*
	* param1: 节点名称
	* param2: 告警信息id
	* desc: 设置指定节点的指定id的告警信息为已读状态
	*/
	string SetAlarmRead(1:string node_name, 2:i32 id),
	//设置告警信息已读
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 告警信息id
	*/
	string SetAlarmReadV3(1:string session_id, 2:string node_name, 3:i32 id),

	//批量设置告警信息已读
	/*
	* param1: 节点名称
	* param2: 告警信息id列表
	* desc: 设置指定节点的指定id的告警信息为已读状态
	*/
	string SetAlarmListRead(1:string node_name, 2:list<i32> id_list),
	//批量设置告警信息已读
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 告警信息id列表
	*/
	string SetAlarmListReadV3(1:string session_id, 2:string node_name, 3:list<i32> id_list),	

	/*(5)节点网络安全接口*/
	
	//添加主机网络规则白名单
	/*
	* param1: 节点名称
	* param2: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param3: 指定端口，如8080
	* param4: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string AddNodeNetWL(1:string node_name, 2:string net, 3:i32 port, 4:string protocol),
	//添加主机网络规则白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param4: 指定端口，如8080
	* param5: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string AddNodeNetWLV3(1:string session_id, 2:string node_name, 3:string net, 4:i32 port, 5:string protocol),

	//删除主机网络规则白名单
	/*
	* param1: 节点名称
	* param2: 网络规则对应的id
	* desc: 删除指定节点的指定id的网络规则
	*/
	string DelNodeNetWL(1:string node_name, 2:i64 id),
	//删除主机网络规则白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 网络规则对应的id
	*/
	string DelNodeNetWLV3(1:string session_id, 2:string node_name, 3:i64 id),

	//修改主机网络规则白名单
	/*
	* param1: 节点名称
	* param2: 原网络规则对应的id
	* param3: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param4: 指定端口，如8080
	* param5: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string UpdateNodeNetWL(1:string node_name, 2:i64 id, 3:string net, 4:i32 port, 5:string protocol),
	//修改主机网络规则白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 原网络规则对应的id
	* param4: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param5: 指定端口，如8080
	* param6: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string UpdateNodeNetWLV3(1:string session_id, 2:string node_name, 3:i64 id, 4:string net, 5:i32 port, 6:string protocol),

	//获取主机网络规则白名单
	/*
	* param1: 节点名称
	* return:
	{
		"status":0,
		"message":"success",
		"rules":[	//json数组，设置的网络规则
			{
				"id":1,					//规则对应的id值
				"net":"172.30.24.101",	//规则设置的网络ip信息
				"port":8080,			//规则设置的端口信息
				"protocol":"tcp"		//规则设置的网络协议信息
			},
			{
				"id":2,
				"net":"172.30.24.0/24",
				"port":0,
				"protocol":"icmp"
			}
		]
	}
	*/
	string GetNodeNetWL(1:string node_name),
	//获取主机网络规则白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetNodeNetWLV3(1:string session_id, 2:string node_name),

	//添加主机网络进程白名单
	/*
	* param1: 节点名称
	* param2: 节点上的进程，需填写完整路径
	*/
	string AddNodeProcessNetWL(1:string node_name, 2:string process_path),
	//添加主机网络进程白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 节点上的进程，需填写完整路径
	*/
	string AddNodeProcessNetWLV3(1:string session_id, 2:string node_name, 3:string process_path),

	//删除主机网络进程白名单
	/*
	* param1: 节点名称
	* param2: 节点上的进程，需填写完整路径
	*/
	string DelNodeProcessNetWL(1:string node_name, 2:string process_path),
	//删除主机网络进程白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 节点上的进程，需填写完整路径
	*/
	string DelNodeProcessNetWLV3(1:string session_id, 2:string node_name, 3:string process_path),

	//修改主机网络进程白名单
	/*
	* param1: 节点名称
	* param2: 节点上的进程，需填写完整路径
	* param3: 节点上的新的进程，需填写完整路径
	*/
	string UpdateNodeProcessNetWL(1:string node_name, 2:string process_path, 3:string new_process_path),
	//修改主机网络进程白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 节点上的进程，需填写完整路径
	* param4: 节点上的新的进程，需填写完整路径
	*/
	string UpdateNodeProcessNetWLV3(1:string session_id, 2:string node_name, 3:string process_path, 4:string new_process_path),

	//获取主机网络进程白名单
	/*
	* param1: 节点名称
	* return: {
		"status":0,
		"message":"success",
		"processes":[				//json数组，进程白名单列表
			"/usr/bin/test1",		//进程路径
			"/usr/bin/test2"
		]
	}
	*/
	string GetNodeProcessNetWL(1:string node_name),
	//获取主机网络进程白名单
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetNodeProcessNetWLV3(1:string session_id, 2:string node_name),

	//设置主机网络进程白名单开关状态
	/*
	* param1: 节点名称
	* param2: 开关状态，true:开启，false:关闭
	*/
	string SetNodeProcessNetWLEnable(1:string node_name, 2:bool enabled),
	//设置主机网络进程白名单开关状态
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 开关状态，true:开启，false:关闭
	*/
	string SetNodeProcessNetWLEnableV3(1:string session_id, 2:string node_name, 3:bool enabled),

	//获取主机网络进程白名单开关状态
	/*
	* param1: 节点名称
	* return: {
		"status":0,
		"message":"success",
		"enabled":true			//进程白名单开关状态，true:开启，false:关闭
	}
	*/
	string GetNodeProcessNetWLEnable(1:string node_name),
	//获取主机网络进程白名单开关状态
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetNodeProcessNetWLEnableV3(1:string session_id, 2:string node_name),

	/*(6)获取网络控制器接口*/

	/*
	* param1: 节点名称
	* desc: 返回ovs网络控制器上所有的网络接口，包含对外网卡信息，平台网卡信息以及所有容器的网卡信息
	* return:
	{
		"status":0,
		"message":"Get net controller infos success",
		"real_eth":[	//json数组，真实网卡信息
			{
				"eth_name":"eth0-real",			//网卡名称，固定eth-real
				"mac":"00:50:56:c0:00:01",		//网卡mac地址
				"status":"正常"					//网卡状态，正常、异常
			},
			{
				"eth_name":"eth1-real",
				"mac":"00:50:56:c0:00:02",
				"status":"正常"
			}
		],
		"platform_eth":[	//json数组，平台网卡信息
			{
				"eth_name":"eth0",		        //网卡名称，固定eth-platform
				"ip":"172.30.25.201",			//网卡ip
				"netmask":"255.255.255.0",		//网卡掩码
				"gateway":"172.30.25.254",		//网卡网关
				"mac":"00:50:56:c0:00:01",		//网卡mac地址
				"status":"正常"					//网卡状态，正常、异常
			}
		],
		"container_eth":[	//json数组，容器网卡信息
			{
				"container_name":"chrome",	//容器名称
				"eths":[
					{
						"eth_name":"eth0",			//容器内网卡名称
						"attach":"eth1"				//容器网卡关联的平台网卡名称
						"ip":"172.30.25.201",		//容器网卡ip
						"netmask":"255.255.255.0",	//容器网卡掩码
						"gateway":"172.30.25.254",	//容器网卡网关
						"mac":"00:50:56:c0:00:01",	//容器网卡mac地址
						"status":"正常"				//网卡状态，正常、异常
					}，
					{
						"eth_name":"eth1",			//容器内网卡名称
						"attach":"eth0"				//容器网卡关联的平台网卡名称
						"ip":"172.30.26.201",		//容器网卡ip
						"netmask":"255.255.255.0",	//容器网卡掩码
						"gateway":"172.30.26.254",	//容器网卡网关
						"mac":"00:50:56:c0:00:02",	//容器网卡mac地址
						"status":"正常"				//网卡状态，正常、异常
					}
				]
			},
    	]
	}
	*/
	string GetNetControllerInfos(1:string node_name),
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetNetControllerInfosV3(1:string session_id, 2:string node_name),


	/********************************************************************************************/

	/*(7)容器管理接口*/

	//获取节点上的所有通过客户端安装的容器
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* return:
	{
		"status":0,
		"message":"success",
		"containers":[	//json数组,所有通过平台安装的容器
			{
				"name":"tomcat",		//容器名称
				"stat":0,				//容器状态，0：在线，1：离线, 2：HA执行中，3：未知
				"uuid":"19c210df-855f-11eb-aba2-0050568b3151",//容器ID
				"cpu":1.5,				//cpu阈值，支持小数
				"cpu_usage":55.5,		//cpu使用率，百分数(0-100)，浮点数
				"mem_usage":1024,		//内存使用量(MB)，整数
				"mem_total":2048,		//内存总量（MB)，整数
				"disk_usage":1024,		//磁盘使用量(GB)，浮点数
				"disk_total":20480,		//磁盘总量(GB)，整数
				"disk_rbw":500,			//磁盘读速度(MB/s)，浮点数
				"disk_wbw":500,			//磁盘写速度(MB/s)，浮点数
				"net_rbw":300,			//网络下载速度(MB/s)，浮点数
				"net_wbw":300,			//网络上传速度(MB/s)，浮点数
				"image":"tomcat",		//使用的镜像名称
				"version":"v1.0.0"		//使用的镜像版本
				"ha":"true",			//ha使能状态
				"level":5,				//模板等级
				"nets":[	//容器多网卡信息
					{
						"eth_name":"eth0",    		//网卡名称
						"ip":"172.30.25.101",		//网卡ip地址
						"mac":"00:50:56:c0:00:01",	//网卡mac地址
						"gateway":"172.30.24.254",	//网卡网关
						"mask":"255.255.255.0"		//网卡子网掩码
					},
					{
						"eth_name":"eth1",			//网卡名称
						"ip":"172.30.25.101",		//网卡ip地址
						"mac":"00:50:56:c0:00:02",	//网卡mac地址
						"gateway":"172.30.25.254",	//网卡网关
						"mask":"255.255.255.0"		//网卡子网掩码
					}
				]
			}
		]
	}
	*/
	string GetAllContainers(1:string node_name),
	/*
	* param1: 节点uuid
	*/
	string GetAllContainersV2(1:string nid),
	/*
	* param1: 用户session的id
	* param2: 节点uuid
	*/
	string GetAllContainersV3(1:string session_id, 2:string nid),

	//安装容器
	/*
	* param1: 节点名称
	* param2: 需要安装的镜像的名称
	* param3: 需要安装的镜像的版本
	* param4: 安装后容器的名称
	* param5: 容器的描述信息
	*/
	string InstallContainer(1:string node_name, 2:string image, 3:string ver, 4:string name, 5:string desc),
	//安装容器
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 需要安装的镜像的名称
	* param4: 需要安装的镜像的版本
	* param5: 安装后容器的名称
	* param6: 容器的描述信息
	*/
	string InstallContainerV3(1:string session_id, 2:string node_name, 3:string image, 4:string ver, 5:string name, 6:string desc),

	//以任务方式安装容器,立即返回任务id
	/*
	* param1: 节点名称
	* param2: 需要安装的镜像的名称
	* param3: 需要安装的镜像的版本
	* param4: 安装后容器的名称
	* param5: 容器的描述信息
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskInstallContainer(1:string node_name, 2:string image, 3:string ver, 4:string name, 5:string desc),
	//以任务方式安装容器,立即返回任务id
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 需要安装的镜像的名称
	* param4: 需要安装的镜像的版本
	* param5: 安装后容器的名称
	* param6: 容器的描述信息
	*/
	string TaskInstallContainerV3(1:string session_id, 2:string node_name, 3:string image, 4:string ver, 5:string name, 6:string desc),

	//启动容器
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	*/
	string StartContainer(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID
	*/
	string StartContainerV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string StartContainerV3(1:string session_id, 2:string cid),

	//任务方式启动容器
	/*
	* param1: 容器UUID
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskStartContainer(1:string cid),
	//任务方式启动容器
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string TaskStartContainerV3(1:string session_id, 2:string cid),

	//停止容器
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	*/
	string StopContainer(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID
	*/
	string StopContainerV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string StopContainerV3(1:string session_id, 2:string cid),

	//重启容器
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	*/
	string RestartContainer(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID
	*/
	string RestartContainerV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string RestartContainerV3(1:string session_id, 2:string cid),

	//任务方式重启容器
	/*
	* param1: 容器UUID
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskRestartContainer(1:string cid),
	//任务方式重启容器
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string TaskRestartContainerV3(1:string session_id, 2:string cid),

	//启动容器内应用
	/*
	* @Deprecated: since 1.0.2
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string StartContainerAppV3(1:string session_id, 2:string cid),

	string StartContainerApp(1:string cid),

	//停止容器内应用
	/*
	* @Deprecated: since 1.0.2
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string StopContainerAppV3(1:string session_id, 2:string cid),

	string StopContainerApp(1:string cid),

	//重启容器内应用
	/*
	* @Deprecated: since 1.0.2
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string RestartContainerAppV3(1:string session_id, 2:string cid),

	string RestartContainerApp(1:string cid),

	//更新容器
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 需要更新的镜像版本
	*/
	string UpdateContainer(1:string node_name, 2:string name, 3:string new_ver),
	/*
	* param1: 容器UUID
	* param2: 需要更新的镜像版本
	*/
	string UpdateContainerV2(1:string cid, 2:string new_ver),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	* param3: 需要更新的镜像版本
	*/
	string UpdateContainerV3(1:string session_id, 2:string cid, 3:string new_ver),

	//任务方式更新容器,立即返回任务id
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 需要更新的镜像版本
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskUpdateContainer(1:string node_name, 2:string name, 3:string new_ver),
	/*
	* param1: 容器UUID
	* param2: 需要更新的镜像版本
	* return: 同上
	*/
	string TaskUpdateContainerV2(1:string cid, 2:string new_ver),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	* param3: 需要更新的镜像版本
	* return: {"status":90,"message":"4c653bd4-e85a-11eb-9ce1-80ff028bfce4"}
	*/
	string TaskUpdateContainerWithAuth(1:string session_id, 2:string cid, 3:string new_ver),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	* param3: 需要更新的镜像版本
	*/
	string TaskUpdateContainerV3(1:string session_id, 2:string cid, 3:string new_ver),

	//获取容器使用的镜像的所有版本
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* desc：返回当前容器使用的这个镜像，所有可用的镜像版本，用于在升级时可选择版本信息
	* return:
	{
		"status":0,
		"message":"success",
		"image":"tomcat",		//容器当前使用的镜像名称
		"c_ver":"v1.0.0",		//容器当前使用的镜像版本
		"vers":[				//json数组，当前镜像其他版本列表
			"v1.0.1",			//镜像在仓库中存在的其他版本
			"v1.0.2"
		]
	}
	*/
	string GetContainerImageVers(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID
	* desc：返回当前容器使用的这个镜像，所有可用的镜像版本，用于在升级时可选择版本信息
	* return: 同上
	*/
	string GetContainerImageVersV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string GetContainerImageVersV3(1:string session_id, 2:string cid),

	//卸载容器
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	*/
	string UninstallContainer(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID
	*/
	string UninstallContainerV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string UninstallContainerV3(1:string session_id, 2:string cid),

	//任务方式卸载容器
	/*
	* param1: 容器UUID
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskUninstallContainer(1:string cid),
	//卸载容器需进行用户验证
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	* return: {"status":90,"message":"4c653bd4-e85a-11eb-9ce1-80ff028bfce4"}
	*/
	string TaskUninstallContainerWithAuth(1:string session_id, 2:string cid),
	//任务方式卸载容器
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string TaskUninstallContainerV3(1:string session_id, 2:string cid),

	//设置容器配额
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器限额设置
	* desc: 限额设置的具体参数，参见 struct ContainerQuota
	*/
	string SetContainerQuota(1:string node_name, 2:string name, 3:ContainerQuota cquota),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器限额设置
	* desc: 限额设置的具体参数，参见 struct ContainerQuota
	*/
	string SetContainerQuotaV2(1:string cid, 2:ContainerQuota cquota),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器限额设置
	*/
	string SetContainerQuotaV3(1:string session_id, 2:string cid, 3:ContainerQuota cquota),

	//获取容器配额
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"Get container mytomcat quota success",
		"quota":{	//容器阈值
			"cpu":0.5,			//cpu数量，支持小数
			"cpu_prio":3,		//cpu优先级，1-5,5优先级最高
			"mem_hard":2048,	//内存硬限制(MB)
			"mem_soft":1024,	//内存软限制(MB)
			"disk_size":2048,	//磁盘大小(GB)
			"disk_rio":20,		//磁盘读频率(次/s)
			"disk_wio":20,		//磁盘写频率(次/s)
			"disk_rbw":20,		//磁盘读速度(MB/s)
			"disk_wbw":20,		//磁盘写速度(MB/s)
			"disk_prio":4,		//磁盘优先级，1-5,5优先级最高
			"net_rbw":20,		//网络下载速度(MB/s)
			"net_wbw":20		//网络上传速度(MB/s)
		}
	}
	*/
	string GetContainerQuota(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerQuotaV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerQuotaV3(1:string session_id, 2:string cid),

	/*(8)容器监控接口*/

	//获取容器统计信息
	/*
	* param1: 节点名称
	* param2: 容器名称
	* param3: 开始时间，传递参数为unix时间戳，精确到秒
	* param4: 结束时间，传递参数为unix时间戳，精确到秒
	* param5: 间隔时间，获取到的监控信息条数应为：nums = （end_time - start_time）/ interval
	* return:
	{
		"status":0,
		"message":"Get container statistics success",
		"cpu":[			//json数组，cpu使用率
			0.5,		//cpu使用率，百分数(0-100),浮点数
			0.6
		],
		"mem":[				
			500,		//内存使用量(MB)，浮点数
			600
		],
		"disk_rbw":[		
			20.5,		//磁盘读速度(MB/s)，浮点数
			30.5
		],
		"disk_wbw":[
			20.5,		//磁盘写速度(MB/s)，浮点数
			30.5
		],
		"disk_rio":[
			10,			//磁盘读频率(次/s)，整数
			20
		],
		"disk_wio":[
			10,			//磁盘写频率(次/s)，整数
			20
		],
		"net_rbw":[
			20.5,		//网络下载速度(MB/s)，浮点数
			30.5
		]
		"net_wbw":[		//网络上传速度(MB/s)，浮点数
			20.5,
			30.5
		]
	}
	*/
	string GetContainerStatistics(1:string node_name, 2:string name, 3:i64 start_time, 4:i64 end_time, 5:i32 interval),
	//获取容器统计信息
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 容器名称
	* param4: 开始时间，传递参数为unix时间戳，精确到秒
	* param5: 结束时间，传递参数为unix时间戳，精确到秒
	* param6: 间隔时间，获取到的监控信息条数应为：nums = （end_time - start_time）/ interval
	*/
	string GetContainerStatisticsV3(1:string session_id, 2:string node_name, 3:string name, 4:i64 start_time, 5:i64 end_time, 6:i32 interval),
	
	/*(9)容器存储接口*/

	//添加容器-主机共享目录
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	* param5: 共享目录权限(0:只读，1:读写)
	*/
	string AddNode2ContainerShareDir(1:string node_name, 2:string name, 3:string c_dir, 4:string n_dir, 5:i8 permit),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 映射到容器内部的挂载目录
	* param3: 主机上需要映射的目录
	* param4: 共享目录权限(0:只读，1:读写)
	*/
	string AddNode2ContainerShareDirV2(1:string cid, 2:string c_dir, 3:string n_dir, 4:i8 permit),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	* param5: 共享目录权限(0:只读，1:读写)
	* return: {"status":90,"message":"4c653bd4-e85a-11eb-9ce1-80ff028bfce4"}
	*/
	string AddNode2ContainerShareDirWithAuth(1:string session_id, 2:string cid, 3:string c_dir, 4:string n_dir, 5:i8 permit),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	* param5: 共享目录权限(0:只读，1:读写)
	*/
	string AddNode2ContainerShareDirV3(1:string session_id, 2:string cid, 3:string c_dir, 4:string n_dir, 5:i8 permit),

	//删除容器-主机共享目录
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	*/
	string DelNode2ContainerShareDir(1:string node_name, 2:string name, 3:string c_dir, 4:string n_dir),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 映射到容器内部的挂载目录
	* param3: 主机上需要映射的目录
	*/
	string DelNode2ContainerShareDirV2(1:string cid, 2:string c_dir, 3:string n_dir),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	*/
	string DelNode2ContainerShareDirV3(1:string session_id, 2:string cid, 3:string c_dir, 4:string n_dir),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	*/
	string DelNode2ContainerShareDirWithAuth(1:string session_id, 2:string cid, 3:string c_dir, 4:string n_dir),

	//修改容器-主机共享目录
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	* param5: 共享目录权限(0:只读，1:读写)
	* param6: 新的映射到容器内部的挂载目录
	* param7: 新的主机上需要映射的目录
	* param8: 新的共享目录权限(0:只读，1:读写)
	* desc: 新修改值可只修改其中一项，其他如不修改需传递原始值；修改共享目录，容器重启后生效
	*/
	string UpdateNode2ContainerShareDir(1:string node_name, 2:string name, 3:string c_dir, 4:string n_dir, 5:i8 permit, 6:string new_c_dir, 7:string new_n_dir, 8:i8 new_permit),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 映射到容器内部的挂载目录
	* param3: 主机上需要映射的目录
	* param4: 共享目录权限(0:只读，1:读写)
	* param5: 新的映射到容器内部的挂载目录
	* param6: 新的主机上需要映射的目录
	* param7: 新的共享目录权限(0:只读，1:读写)
	* desc: 新修改值可只修改其中一项，其他如不修改需传递原始值；修改共享目录，容器重启后生效
	*/
	string UpdateNode2ContainerShareDirV2(1:string cid, 2:string c_dir, 3:string n_dir, 4:i8 permit, 5:string new_c_dir, 6:string new_n_dir, 7:i8 new_permit),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	* param5: 共享目录权限(0:只读，1:读写)
	* param6: 新的映射到容器内部的挂载目录
	* param7: 新的主机上需要映射的目录
	* param8: 新的共享目录权限(0:只读，1:读写)
	*/
	string UpdateNode2ContainerShareDirV3(1:string session_id, 2:string cid, 3:string c_dir, 4:string n_dir, 5:i8 permit, 6:string new_c_dir, 7:string new_n_dir, 8:i8 new_permit),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 映射到容器内部的挂载目录
	* param4: 主机上需要映射的目录
	* param5: 共享目录权限(0:只读，1:读写)
	* param6: 新的映射到容器内部的挂载目录
	* param7: 新的主机上需要映射的目录
	* param8: 新的共享目录权限(0:只读，1:读写)
	*/
	string UpdateNode2ContainerShareDirWithAuth(1:string session_id, 2:string cid, 3:string c_dir, 4:string n_dir, 5:i8 permit, 6:string new_c_dir, 7:string new_n_dir, 8:i8 new_permit),

	//获取容器-主机共享目录
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"dirs":[	//json数组，所有共享目录
			{
				"c_dir":"/home/data",	//容器内目录
				"n_dir":"/opt/test",	//节点上目录
				"permit":0				//共享目录权限值，0:只读，1:读写
			}
		]
	}
	*/
	string GetNode2ContainerShareDir(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetNode2ContainerShareDirV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetNode2ContainerShareDirV3(1:string session_id, 2:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetNode2ContainerShareDirWithAuth(1:string session_id, 2:string cid),

	/*(10)容器安全接口*/
	
	//添加容器关键文件保护
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* param4: 删除权限，0：可删除，1：不可删除
	* param5: 修改权限，0：可修改，1：不可修改
	* desc: 添加关键文件保护，调用成功则即时生效
	*/
	string AddContainerProtectFile(1:string node_name, 2:string name, 3:string file, 4:i8 permit_del, 5:i8 permit_mod),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 关键文件，需要指定该文件在节点上的完整路径
	* param3: 删除权限，0：可删除，1：不可删除
	* param4: 修改权限，0：可修改，1：不可修改
	* desc: 添加关键文件保护，调用成功则即时生效
	*/
	string AddContainerProtectFileV2(1:string cid, 2:string file, 3:i8 permit_del, 4:i8 permit_mod),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* param4: 删除权限，0：可删除，1：不可删除
	* param5: 修改权限，0：可修改，1：不可修改
	*/
	string AddContainerProtectFileV3(1:string session_id, 2:string cid, 3:string file, 4:i8 permit_del, 5:i8 permit_mod),

	//删除容器关键文件保护
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* desc: 删除关键文件保护，调用成功则即时生效
	*/
	string DelContainerProtectFile(1:string node_name, 2:string name, 3:string file),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 关键文件，需要指定该文件在节点上的完整路径
	* desc: 删除关键文件保护，调用成功则即时生效
	*/
	string DelContainerProtectFileV2(1:string cid, 2:string file),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	*/
	string DelContainerProtectFileV3(1:string session_id, 2:string cid, 3:string file),

	//修改容器关键文件保护
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* param4: 新关键文件，需要指定该文件在节点上的完整路径，如仅需修改权限位，则该路径指定与param2一致
	* param5: 删除权限，0：可删除，1：不可删除
	* param6: 修改权限，0：可修改，1：不可修改
	* desc: 修改关键文件保护，调用成功则即时生效
	*/
	string UpdateContainerProtectFile(1:string node_name, 2:string name, 3:string file, 4:string new_file, 5:i8 permit_del, 6:i8 permit_mod),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 关键文件，需要指定该文件在节点上的完整路径
	* param3: 新关键文件，需要指定该文件在节点上的完整路径，如仅需修改权限位，则该路径指定与param2一致
	* param4: 删除权限，0：可删除，1：不可删除
	* param5: 修改权限，0：可修改，1：不可修改
	* desc: 修改关键文件保护，调用成功则即时生效
	*/
	string UpdateContainerProtectFileV2(1:string cid, 2:string file, 3:string new_file, 4:i8 permit_del, 5:i8 permit_mod),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 关键文件，需要指定该文件在节点上的完整路径
	* param4: 新关键文件，需要指定该文件在节点上的完整路径，如仅需修改权限位，则该路径指定与param2一致
	* param5: 删除权限，0：可删除，1：不可删除
	* param6: 修改权限，0：可修改，1：不可修改
	*/
	string UpdateContainerProtectFileV3(1:string session_id, 2:string cid, 3:string file, 4:string new_file, 5:i8 permit_del, 6:i8 permit_mod),

	//获取容器关键文件列表
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"Get node protect files success",
		"files":[	//json数组，关键文件列表
			{
				"path":"/home/file1",	//文件路径
				"permit_del":0,			//删除权限，0：可删除，1：不可删除
				"permit_mod":1			//修改权限，0：可删除，1：不可删除
			}
		]
	}
	*/
	string GetContainerProtectFiles(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerProtectFilesV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerProtectFilesV3(1:string session_id, 2:string cid),

	//添加白名单进程
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string AddProcessWhiteList(1:string node_name, 2:string name, 3:string process_path),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string AddProcessWhiteListV2(1:string cid, 2:string process_path),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string AddProcessWhiteListV3(1:string session_id, 2:string cid, 3:string process_path),

	//删除白名单进程
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string DelProcessWhiteList(1:string node_name, 2:string name, 3:string process_path),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string DelProcessWhiteListV2(1:string cid, 2:string process_path),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string DelProcessWhiteListV3(1:string session_id, 2:string cid, 3:string process_path),

	//修改白名单进程
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	* param4: 容器上的新进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string UpdateProcessWhiteList(1:string node_name, 2:string name, 3:string process_path, 4:string new_process_path),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	* param3: 容器上的新进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string UpdateProcessWhiteListV2(1:string cid, 2:string process_path, 3:string new_process_path),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	* param4: 容器上的新进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string UpdateProcessWhiteListV3(1:string session_id, 2:string cid, 3:string process_path, 4:string new_process_path),

	//获取白名单进程
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"values":[					//json数组，网络进程白名单列表
			"/usr/bin/test1",		//进程路径
			"/usr/bin/test2"
		]
	}
	*/
	string GetProcessWhiteList(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetProcessWhiteListV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetProcessWhiteListV3(1:string session_id, 2:string cid),

	//设置容器进程白名单状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 安全模块状态
	*/
	string SetSecurityModuleStat(1:string node_name, 2:string name, 3:i8 status),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 安全模块状态
	*/
	string SetSecurityModuleStatV2(1:string cid, 2:i8 status),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 安全模块状态
	*/
	string SetSecurityModuleStatV3(1:string session_id, 2:string cid, 3:i8 status),

	//获取容器进程白名单状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return: {
		"status":0,
		"message":"success",
		"security_module_status":[1,0,0] // 数组的三个元素分别设置安全模块工作模式、进程白名单限制状态、关键文件保护状态（0：关闭/调试模式， 1：开启/强制模式）
	}
	*/
	string GetSecurityModuleStat(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetSecurityModuleStatV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetSecurityModuleStatV3(1:string session_id, 2:string cid),

	/*(11)容器网络设置接口*/

	//设置容器网络参数
	/*
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器ip地址
	* param4: 容器子网掩码
	* param5: 容器网关
	*/
	string SetContainerNetwork(1:string node_name, 2:string name, 3:string ip, 4:string mask, 5:string gw),
	//设置容器网络参数
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 容器名称
	* param4: 容器ip地址
	* param5: 容器子网掩码
	* param6: 容器网关
	*/
	string SetContainerNetworkV3(1:string session_id, 2:string node_name, 3:string name, 4:string ip, 5:string mask, 6:string gw),
	
	//获取容器网络参数
	/*
	* param1: 节点名称
	* param2: 容器名称
	* return: {
		"status":0,
		"message":"success",
		"ip":"192.168.0.1",		//容器ip
		"mask":"255.255.255.0",		//容器掩码
		"gw":"192.168.0.254"		//容器网关
	}
	*/
	string GetContainerNetwork(1:string node_name, 2:string name),
	//获取容器网络参数
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: 容器名称
	*/
	string GetContainerNetworkV3(1:string session_id, 2:string node_name, 3:string name),

	//添加容器网卡
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string AddContainerNetwork(1:string node_name, 2:string name, 3:list<ContainerNetwork> net),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string AddContainerNetworkV2(1:string cid, 2:list<ContainerNetwork> net),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string AddContainerNetworkV3(1:string session_id, 2:string cid, 3:list<ContainerNetwork> net),
	
	//删除容器网卡
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 指定要删除的容器网卡名称
	*/
	string DelContainerNetwork(1:string node_name, 2:string name, 3:list<string> eth_name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 指定要删除的容器网卡名称
	*/
	string DelContainerNetworkV2(1:string cid, 2:list<string> eth_name),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 指定要删除的容器网卡名称
	*/
	string DelContainerNetworkV3(1:string session_id, 2:string cid, 3:list<string> eth_name),
	
	//更新容器网卡
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string UpdateContainerNetwork(1:string node_name, 2:string name, 3:list<ContainerNetwork> net),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string UpdateContainerNetworkV2(1:string cid, 2:list<ContainerNetwork> net),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string UpdateContainerNetworkV3(1:string session_id, 2:string cid, 3:list<ContainerNetwork> net),

	//获取容器网卡
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return: {
		"status":0,
		"message":"success",
		"network":[
			{
				"eth_name":"eth0",			//网卡名称
				"attach":"eth1",			//网卡关联的平台网卡名称
				"ip":"192.168.0.1",			//ip
				"mask":"255.255.255.0",		//掩码
				"gw":"192.168.0.254",		//网关
				"mac":"80:ff:01:90:76:a2"	//mac地址
			},
			{
				"eth_name":"eth1",
				"attach":"eth0",
				"ip":"192.168.1.1",
				"mask":"255.255.255.0",
				"gw":"192.168.1.254",
				"mac":"80:ff:01:90:76:a3"
			}
		]
	}
	*/
	string GetContainerNetworkList(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerNetworkListV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerNetworkListV3(1:string session_id, 2:string cid),


	/*(12)容器网络安全接口*/
	//设置容器对外网络访问许可
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 对外网络访问许可：true:启用对外访问限制，此时不允许容器访问外部网络；false:关闭对外访问限制，此时允许容器访问外部网络 
	*/
	string SetContainerExtNetLimit(1:string node_name, 2:string name, 3:bool permit),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 对外网络访问许可：true:启用对外访问限制，此时不允许容器访问外部网络；false:关闭对外访问限制，此时允许容器访问外部网络 
	*/
	string SetContainerExtNetLimitV2(1:string cid, 2:bool permit),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 对外网络访问许可：true:启用对外访问限制，此时不允许容器访问外部网络；false:关闭对外访问限制，此时允许容器访问外部网络 
	*/
	string SetContainerExtNetLimitV3(1:string session_id, 2:string cid, 3:bool permit),

	//获取容器对外网络访问许可
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return: {
		"status":0,
		"message":"success",
		"extnet_limit":true		//true:启用对外访问限制，此时不允许容器访问外部网络；false:关闭对外访问限制，此时允许容器访问外部网络 
	}
	*/
	string GetContainerExtNetLimit(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerExtNetLimitV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerExtNetLimitV3(1:string session_id, 2:string cid),

	//添加容器网络规则白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param4: 指定端口，如8080
	* param5: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string AddContainerNetWL(1:string node_name, 2:string name, 3:string net, 4:i32 port, 5:string protocol),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param3: 指定端口，如8080
	* param4: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string AddContainerNetWLV2(1:string cid, 2:string net, 3:i32 port, 4:string protocol),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param4: 指定端口，如8080
	* param5: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string AddContainerNetWLV3(1:string session_id, 2:string cid, 3:string net, 4:i32 port, 5:string protocol),

	//删除容器网络规则白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 原网络规则对应的id
	*/
	string DelContainerNetWL(1:string node_name, 2:string name, 3:i64 id),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 原网络规则对应的id
	*/
	string DelContainerNetWLV2(1:string cid, 2:i64 id),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 原网络规则对应的id
	*/
	string DelContainerNetWLV3(1:string session_id, 2:string cid, 3:i64 id),

	//修改容器网络规则白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 原网络规则对应的id
	* param4: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param5: 指定端口，如8080
	* param6: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string UpdateContainerNetWL(1:string node_name, 2:string name, 3:i64 id, 4:string net, 5:i32 port, 6:string protocol),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 原网络规则对应的id
	* param3: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param4: 指定端口，如8080
	* param5: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string UpdateContainerNetWLV2(1:string cid, 2:i64 id, 3:string net, 4:i32 port, 5:string protocol),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 原网络规则对应的id
	* param4: 设置网络规则的ip地址或网段，示例：172.30.25.100 或者 172.30.25.0/24
	* param5: 指定端口，如8080
	* param6: 指定协议，如tcp，指定的协议可为 tcp \ udp \ icmp
	*/
	string UpdateContainerNetWLV3(1:string session_id, 2:string cid, 3:i64 id, 4:string net, 5:i32 port, 6:string protocol),

	//获取容器网络规则白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"rules":[	//json数组，容器网络规则
			{
				"id":1,						//规则id
				"net":"172.30.24.101",		//规则ip
				"port":8080,				//规则端口
				"protocol":"tcp"			//规则协议
			},
			{
				"id":2,
				"net":"172.30.24.0/24",
				"port":0,
				"protocol":"icmp"
			}
		]
	}
	*/
	string GetContainerNetWL(1:string node_name, 2:string name),
	//获取容器网络规则白名单
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerNetWLV2(1:string cid),
	//获取容器网络规则白名单
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerNetWLV3(1:string session_id, 2:string cid),

	//添加容器网络进程白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string AddContainerProcessNetWL(1:string node_name, 2:string name, 3:string process_path),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string AddContainerProcessNetWLV2(1:string cid, 2:string process_path),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string AddContainerProcessNetWLV3(1:string session_id, 2:string cid, 3:string process_path),

	//删除容器网络进程白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string DelContainerProcessNetWL(1:string node_name, 2:string name, 3:string process_path),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string DelContainerProcessNetWLV2(1:string cid, 2:string process_path),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string DelContainerProcessNetWLV3(1:string session_id, 2:string cid, 3:string process_path),

	//修改容器网络进程白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	* param4: 容器上的新进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string UpdateContainerProcessNetWL(1:string node_name, 2:string name, 3:string process_path, 4:string new_process_path),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	* param3: 容器上的新进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string UpdateContainerProcessNetWLV2(1:string cid, 2:string process_path, 3:string new_process_path),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器上的进程，需填写完整路径(该文件在容器内部的完整路径)
	* param4: 容器上的新进程，需填写完整路径(该文件在容器内部的完整路径)
	*/
	string UpdateContainerProcessNetWLV3(1:string session_id, 2:string cid, 3:string process_path, 4:string new_process_path),

	//获取容器网络进程白名单
	/*
	* @Deprecated: since 1.0.2
	* param1: 容器UUID 或者 模板UUID
	* return: 
	{
		"status":0,
		"message":"success",
		"values":[				//json数组，白名单中的进程
			"/usr/bin/test1",		//进程路径
			"/usr/bin/test2"
		]
	}
	*/
	string GetContainerProcessNetWL(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerProcessNetWLV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerProcessNetWLV3(1:string session_id, 2:string cid),

	//设置容器网络进程白名单开关状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器白名单状态(true:开启， false:关闭)
	*/
	string SetContainerProcessNetWLEnable(1:string node_name, 2:string name, 3:bool enabled),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器白名单状态(true:开启， false:关闭)
	*/
	string SetContainerProcessNetWLEnableV2(1:string cid, 2:bool enabled),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器白名单状态(true:开启， false:关闭)
	*/
	string SetContainerProcessNetWLEnableV3(1:string session_id, 2:string cid, 3:bool enabled),

	//获取容器网络进程白名单开关状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return: 
	{
		"status":0,
		"message":"success",
		"enabled":true			//容器网络进程白名单是否开启，false：关闭，true：开启
	}
	*/
	string GetContainerProcessNetWLEnable(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerProcessNetWLEnableV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerProcessNetWLEnableV3(1:string session_id, 2:string cid),

	/*(13)镜像仓库接口*/
	
	//获取镜像仓库sftp服务用户和密码时需进行用户验证
	/*
	* param1: 用户session的id
	*return: {"status":90,"message":"4c653bd4-e85a-11eb-9ce1-80ff028bfce4"}
	*/
    string GetImageRepoUserWithAuth(1:string session_id),

	//重命名镜像
	/*
	* param1: 原镜像名称
	* param2: 原镜像版本
	* param3: 新镜像名称
	* param4: 新镜像版本
	* param5: 仓库类型(0:仓库区域，1:暂存区域)
	*/
	string RenameImage(1:string image, 2:string ver, 3:string new_image, 4:string new_ver, 5:i8 repo_type),
	//重命名镜像
	/*
	* param1: 用户session的id
	* param2: 原镜像名称
	* param3: 原镜像版本
	* param4: 新镜像名称
	* param5: 新镜像版本
	* param6: 仓库类型(0:仓库区域，1:暂存区域)
	*/
	string RenameImageV3(1:string session_id, 2:string image, 3:string ver, 4:string new_image, 5:string new_ver, 6:i8 repo_type),

	//删除镜像
	/*
	* param1: 镜像名称
	* param2: 镜像版本
	* param3: 仓库类型(0:仓库区域，1:暂存区域)
	*/
	string DeleteImage(1:string image, 2:string ver, 3:i8 repo_type),
	//删除镜像
	/*
	* param1: 用户session的id
	* param2: 镜像名称
	* param3: 镜像版本
	* param4: 仓库类型(0:仓库区域，1:暂存区域)
	*/
	string DeleteImageV3(1:string session_id, 2:string image, 3:string ver, 4:i8 repo_type),

	//获取镜像列表
	/*
	* param1: 仓库类型(0:仓库区域，1:暂存区域)
	* return:
	{
		"status":0,
		"message":"success",
		"images":[	//json数组，镜像列表
			{
				"name":"tomcat",		//镜像名称
				"ver":"v1.0.0",			//镜像版本
				"verify_result":0,		//镜像校验结果，0:不通过，1:通过
				"refuse":""				//镜像回退的原因
			},
			{
				"name":"tomcat",
				"ver":"v1.0.1",
				"verify_result":1,
				"refuse":"回退理由"
			}
		]
	}
	*/
	string GetImages(1:i8 repo_type),
	/*
	* param1: 用户session的id
	* param2: 仓库类型(0:仓库区域，1:暂存区域)
	*/
	string GetImagesV3(1:string session_id, 2:i8 repo_type),

	//验签
	/*
	* param1: 镜像tar包文件路径
	* param2: 镜像签名文件sign的路径
	*/
	string VerifySign(1:string image, 2:string sign_file),
	/*
	* param1: 用户session的id
	* param2: 镜像签名二合一文件
	*/
	string VerifySignV3(1:string session_id, 2:string image_sign_file),

	//任务方式验签
	/*
	* param1: 镜像tar包文件路径
	* param2: 镜像签名文件sign的路径
	*/
	string TaskVerifySign(1:string image, 2:string sign_file),
	/*
	* param1: 用户session的id
	* param2: 镜像签名二合一文件
	*/
	string TaskVerifySignV3(1:string session_id, 2:string image_sign_file),

	//转存镜像到仓库(仅securitor使用)
	/*
	* param1: 镜像名称
	* param2: 镜像版本
	*/
	string SaveImage2Repo(1:string image, 2:string ver),
	/*
	* param1: 用户session的id
	* param2: 镜像名称
	* param3: 镜像版本
	* return: {"status":90,"message":"4c653bd4-e85a-11eb-9ce1-80ff028bfce4"}
	*/
	string SaveImage2RepoWithAuth(1:string session_id, 2:string image, 3:string ver),
	/*
	* param1: 用户session的id
	* param2: 镜像名称
	* param3: 镜像版本
	*/
	string SaveImage2RepoV3(1:string session_id, 2:string image, 3:string ver),

	//回退镜像(仅securitor使用)
	/*
	* param1: 镜像名称
	* param2: 镜像版本
	* param3: 回退原因
	*/
	string RefuseImage2Repo(1:string image, 2:string ver, 3:string msg),
	/*
	* param1: 用户session的id
	* param2: 镜像名称
	* param3: 镜像版本
	* param4: 回退原因
	*/
	string RefuseImage2RepoV3(1:string session_id, 2:string image, 3:string ver, 4:string msg),

	//设置跳过安全审计配置\跳过待审批列表（版本编号）：仅 securitor 用户使用）
	/*
	* param1: 跳过安全审计(true:跳过， false:不跳过)
	* param2: 跳过待审批列表(true:跳过， false:不跳过)
	*/
	string SetSkipSecurity(1:bool skip_audit, 2:bool skip_pending),
	/*
	* param1: 用户session的id
	* param2: 跳过安全审计(true:跳过， false:不跳过)
	* param3: 跳过待审批列表(true:跳过， false:不跳过)
	*/
	string SetSkipSecurityV3(1:string session_id, 2:bool skip_audit, 3:bool skip_pending),

	//获取跳过安全审计配置\跳过待审批列表的设置（版本编号）：仅 securitor 用户使用）
	/*
	*return:
	{
		"status":0,
		"message":"success",
		"skip_audit":true,		//是否跳过安全审计，无需securitor确认，admin即可直接上传到仓库
		"skip_pending":false	//是否跳过待审批列表，如已设置跳过安全申请，但上传镜像如发生校验失败，仍然存放在待审批列表中。
								//如开启该选项，则校验失败时仍将镜像直接上传到仓库中。
	}
	*/
	string GetSkipSecurity(),
	/*
	* param1: 用户session的id
	*/
	string GetSkipSecurityV3(1:string session_id),

	/*(14)模板设置*/

	//设置容器模板等级
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器模板等级(支持:1,2,3,4,5)
	*/
	string SetTemplate(1:string node_name, 2:string name, 3:i8 level)
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器模板等级(支持:1,2,3,4,5)
	*/
	string SetTemplateV2(1:string cid, 2:i8 level),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器模板等级(支持:1,2,3,4,5)
	* return: {"status":90,"message":"4c653bd4-e85a-11eb-9ce1-80ff028bfce4"}
	*/
	string SetTemplateWithAuth(1:string session_id, 2:string cid, 3:i8 level),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器模板等级(支持:1,2,3,4,5)
	*/
	string SetTemplateV3(1:string session_id, 2:string cid, 3:i8 level),

	//获取容器模板等级
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return: {"status":0,"message":"success","level":1}
	* 1 <= level <= 5
	*/
	string GetTemplate(1:string node_name, 2:string name)
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetTemplateV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetTemplateV3(1:string session_id, 2:string cid),

	/*(15)容器环境变量接口*/

	//添加容器环境变量
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 环境变量名称
	* param4: 环境变量值
	*/
	string AddContainerEnv(1:string node_name, 2:string name, 3:string env_name, 4:string env_val),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 环境变量名称
	* param3: 环境变量值
	*/
	string AddContainerEnvV2(1:string cid, 2:string env_name, 3:string env_val),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 环境变量名称
	* param4: 环境变量值
	*/
	string AddContainerEnvV3(1:string session_id, 2:string cid, 3:string env_name, 4:string env_val),

	//删除容器环境变量
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param2: 环境变量名称
	*/
	string DelContainerEnv(1:string node_name, 2:string name, 3:string env_name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 环境变量名称
	*/
	string DelContainerEnvV2(1:string cid, 2:string env_name),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 环境变量名称
	*/
	string DelContainerEnvV3(1:string session_id, 2:string cid, 3:string env_name),

	//更新容器环境变量
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 环境变量名称
	* param4: 新的环境变量名称
	* param5: 新的环境变量值
	* 注：新的值可只修改一个，另一个则与原参数值一致
	*/
	string UpdateContainerEnv(1:string node_name, 2:string name, 3:string env_name, 4:string new_env_name, 5:string new_env_val),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 环境变量名称
	* param3: 新的环境变量名称
	* param4: 新的环境变量值
	* 注：新的值可只修改一个，另一个则与原参数值一致
	*/
	string UpdateContainerEnvV2(1:string cid, 2:string env_name, 3:string new_env_name, 4:string new_env_val),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 环境变量名称
	* param4: 新的环境变量名称
	* param5: 新的环境变量值
	* 注：新的值可只修改一个，另一个则与原参数值一致
	*/
	string UpdateContainerEnvV3(1:string session_id, 2:string cid, 3:string env_name, 4:string new_env_name, 5:string new_env_val),

	//获取容器环境变量
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return: 
	{
		"status":0,
		"message":"获取环境变量成功",
		"env":[
			{
				"name":"LINXSC_PLATFORM_IP",
				"val":"172.30.25.253"
			},
			{
				"name":"TESTENV",
				"val":"env_test"
			}
		]
	}
	*/
	string GetContainerEnv(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerEnvV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerEnvV3(1:string session_id, 2:string cid),

	/*(15)容器显示设置接口*/
	//设置容器显示状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 是否使能容器的图形应用支持（true:使能，false:关闭）
	* param4: 传递默认的$DISPLAY的值，如:0 ; :10.0; localhost:0 ; 172.30.25.123:0
	*/
	string SetContainerDisplay(1:string node_name, 2:string name, 3:bool enabled, 4:string display),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 是否使能容器的图形应用支持（true:使能，false:关闭）
	* param3: 传递默认的$DISPLAY的值，如:0 ; :10.0; localhost:0 ; 172.30.25.123:0
	*/
	string SetContainerDisplayV2(1:string cid, 2:bool enabled, 3:string display),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 是否使能容器的图形应用支持（true:使能，false:关闭）
	* param4: 传递默认的$DISPLAY的值，如:0 ; :10.0; localhost:0 ; 172.30.25.123:0
	*/
	string SetContainerDisplayV3(1:string session_id, 2:string cid, 3:bool enabled, 4:string display),

	//获取容器显示状态,使能状态和DISPLAY环境变量的值
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":" success",
		"display":true,					//容器是否使能图形应用
		"display_env":"172.30.25.123:0" //容器设置的$DISPLAY环境变量的值
	}
	*/
	string GetContainerDisplay(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerDisplayV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerDisplayV3(1:string session_id, 2:string cid),

	//设置容器显示环境变量DISPLAY的值
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 传递到容器中的$DISPLAY的值。eg:172.30.25.123:0
	*/
	string SetContainerDisplayEnv(1:string node_name, 2:string name, 3:string display_env),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 传递到容器中的$DISPLAY的值。eg:172.30.25.123:0
	*/
	string SetContainerDisplayEnvV2(1:string cid, 2:string display_env),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 传递到容器中的$DISPLAY的值。eg:172.30.25.123:0
	*/
	string SetContainerDisplayEnvV3(1:string session_id, 2:string cid, 3:string display_env),
	
	/*(16) 容器磁盘相关接口（持久化）*/

	//创建磁盘
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 新建磁盘的列表
	*/
	string AddContainerDisk(1:string node_name, 2:string name, 3:list<ContainerDisk> disk),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 新建磁盘的列表
	*/
	string AddContainerDiskV2(1:string cid, 2:list<ContainerDisk> disk),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 新建磁盘的列表
	*/
	string AddContainerDiskV3(1:string session_id, 2:string cid, 3:list<ContainerDisk> disk),

	//任务方式创建磁盘,该接口返回成功则立即返回任务id
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 新建磁盘的列表
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskAddContainerDisk(1:string node_name, 2:string name, 3:list<ContainerDisk> disk),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 新建磁盘的列表
	* return: 同上
	*/
	string TaskAddContainerDiskV2(1:string cid, 2:list<ContainerDisk> disk),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 新建磁盘的列表
	*/
	string TaskAddContainerDiskV3(1:string session_id, 2:string cid, 3:list<ContainerDisk> disk),

	//删除磁盘
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 需要删除的磁盘名称列表
	*/
	string DelContainerDisk(1:string node_name, 2:string name, 3:list<string> disk_name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 需要删除的磁盘名称列表
	*/
	string DelContainerDiskV2(1:string cid, 2:list<string> disk_name),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 需要删除的磁盘名称列表
	*/
	string DelContainerDiskV3(1:string session_id, 2:string cid, 3:list<string> disk_name),

	//修改磁盘
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 需要修改的磁盘列表，支持修改磁盘大小
	*/
	string UpdateContainerDisk(1:string node_name, 2:string name, 3:list<ContainerDisk> disk),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 需要修改的磁盘列表，支持修改磁盘大小
	*/
	string UpdateContainerDiskV2(1:string cid, 2:list<ContainerDisk> disk),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 需要修改的磁盘列表，支持修改磁盘大小
	*/
	string UpdateContainerDiskV3(1:string session_id, 2:string cid, 3:list<ContainerDisk> disk),

	//任务方式修改磁盘,该接口返回成功则立即返回任务id
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 需要修改的磁盘列表，支持修改磁盘大小
	* return:
	{
		"status":0,
		"message":"success",
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskUpdateContainerDisk(1:string node_name, 2:string name, 3:list<ContainerDisk> disk),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 需要修改的磁盘列表，支持修改磁盘大小
	* return: 同上
	*/
	string TaskUpdateContainerDiskV2(1:string cid, 2:list<ContainerDisk> disk),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 需要修改的磁盘列表，支持修改磁盘大小
	*/
	string TaskUpdateContainerDiskV3(1:string session_id, 2:string cid, 3:list<ContainerDisk> disk),

	//获取磁盘信息
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"total":15,		//所有磁盘总量,单位GB
		"used":7.1,		//所有磁盘使用量,单位GB
		"disk":[
			{
				"name":"disk1",				//磁盘名称
				"mount_point":"/tmp",		//挂载点
				"total":5,					//单位GB
				"used":2.5					//单位GB
			},
			{
				"name":"disk2",
				"mount_point":"/opt",
				"total":5,
				"used":2.5
			},
			{
				"name":"",
				"mount_point":"/",
				"total":5,
				"used":2.1
			}
		]
	}
	*/
	string GetContainerDisk(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerDiskV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerDiskV3(1:string session_id, 2:string cid),

	//获取安全容器系统集成版本信息与各个组件信息
	/*
	* return:
	{
		"status":0,
		"message":"success",
		"nodes":[
			{
				"name":"node1",
				"uuid":"90efacea-82fc-11eb-8d52-0050568b3151",
				"versions":{
					"linxc":"1.0.0",
					"linxc_ovs":"1.0.0",
					"linxc_agent":"1.0.0",
					"integrated":"R1.0.0"
				}
			},
			{
				"name":"node2",
				"uuid":"aec26d13-cfca-11eb-bdaf-80ff568bf080",
				"versions":{
					"linxc":"1.0.0",
					"linxc_ovs":"1.0.0",
					"linxc_agent":"1.0.0",
					"integrated":"R1.0.0"
				}
			}
		]
	}
	*/
	string GetVersions(),
	/*
	* param1: 用户session的id
	*/
	string GetVersionsV3(1:string session_id),

	//获取指定节点上任务状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 任务ID
	* return:
	{
		"status":0,
		"message":"获取任务状态成功",
		"task_info":{
			"id":"289158f9-2f8a-11eb-87dc-0050568b2765",	//任务id
			"type":5000,									//任务类型：5000:安装容器;5001:创建磁盘;5002:更新容器;5003:更新磁盘
			"type_str":"安装容器",							//任务类型描述
			"status":2,										//任务状态类型：0:create;1:running;2:done;3:error
			"message":"安装容器完成",						//任务状态信息
			"create_at":1606355599							//任务创建时间,Unix时间戳,精确到秒
		}
	}
	*/
	string GetTaskInfo(1:string node_name, 2:string task_id),
	/*
	* param1: 节点ID
	* param2: 任务ID
	* return: 同上
	*/
	string GetTaskInfoV2(1:string nid, 2:string task_id),
	/*
	* param1: 用户session的id
	* param2: 节点ID
	* param3: 任务ID
	*/
	string GetTaskInfoV3(1:string session_id, 2:string nid, 3:string task_id),

	/*IPC相关接口*/
	//添加ipc基准标签
	/*
	* param1: 节点名称
	* param2: Ipc基准标签列表
	*/
	string AddIpcTag(1:string node_name, 2:list<string> ipc_tags),
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: Ipc基准标签列表
	*/
	string AddIpcTagV3(1:string session_id, 2:string node_name, 3:list<string> ipc_tags),

	//删除ipc基准标签
	/*
	* param1: 节点名称
	* param2: Ipc基准标签列表
	* return:
	{"status":0,"message":"success"}
	如已存在使用该ipc标签的容器情况下的返回值示例：
	{"status":1,"message":"ipc-a已被容器container-1、container-2使用，无法删除该基准!"}
	*/
	string DelIpcTag(1:string node_name, 2:list<string> ipc_tags),
	/*
	* param1: 用户session的id
	* param2: 节点名称
	* param3: Ipc基准标签列表
	*/
	string DelIpcTagV3(1:string session_id, 2:string node_name, 3:list<string> ipc_tags),

	//获取ipc基准标签
	/*
	* param1: 节点名称
	* return:
	{
		"status":0,
		"message":"获取IPC标签成功",
		"ipc_tags":[
			{
				"name":"ipc-a",
				"containers":[
					"app1",
					"app2"
				]
			},
			{
				"name":"ipc-b",
				"containers":[
					"app3"
				]
			}
		]
	}
	*/
	string GetIpcTag(1:string node_name),
	/*
	* param1: 用户session的id
	* param2: 节点名称
	*/
	string GetIpcTagV3(1:string session_id, 2:string node_name),

	//设置容器ipc使能状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 使能状态(true：开启，false：关闭)
	*/
	string SetContainerIpcEnable(1:string node_name, 2:string name, 3:bool enable),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 使能状态(true：开启，false：关闭)
	*/
	string SetContainerIpcEnableV2(1:string cid, 2:bool enable),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 使能状态(true：开启，false：关闭)
	*/
	string SetContainerIpcEnableV3(1:string session_id, 2:string cid, 3:bool enable),

	//获取容器ipc使能状态
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"enable":true
	}
	*/
	string GetContainerIpcEnable(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerIpcEnableV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerIpcEnableV3(1:string session_id, 2:string cid),

	//设置容器ipc基准标签
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: ipc基准标签
	*/
	string SetContainerIpcTag(1:string node_name, 2:string name, 3:string ipc_tag),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: ipc基准标签
	*/
	string SetContainerIpcTagV2(1:string cid, 2:string ipc_tag),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: ipc基准标签
	*/
	string SetContainerIpcTagV3(1:string session_id, 2:string cid, 3:string ipc_tag),

	//获取容器ipc基准标签
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"ipc_tag":"ipc-a",		//容器当前使用的ipc基准
		"avl_ipc_tags":[		//可用的其他ipc基准列表
			"ipc-b",
			"ipc-c"
		]
	}
	*/
	string GetContainerIpcTag(1:string node_name, 2:string name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerIpcTagV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerIpcTagV3(1:string session_id, 2:string cid),

	//获取节点ssh端口
	/*
	* return:
	{
		"status":0,
		"message":"获取ssh端口成功",
		"ssh_port": "24"		//节点ssh端口号
	}
	*/
	string GetSSHPort(),
	/*
	* param1: 用户session的id
	*/
	string GetSSHPortV3(1:string session_id),

	/*---------------------------------------------------------------------------------------------------*/

	//重置容器HA配置信息
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	*/
	string ResetContainerHaConf(1:string node_name, 2:string container_name),
	/*
	* param1:容器UUID 或者 模板UUID
	*/
	string ResetContainerHaConfV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string ResetContainerHaConfV3(1:string session_id, 2:string cid),

	//应用容器HA配置信息
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param3: 容器高可用配置项,详情参考 struct ContainerHA
	*/
	string ApplyContainerHaConf(1:string node_name, 2:string container_name, 3:ContainerHA cha),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器高可用配置项,详情参考 struct ContainerHA
	*/
	string ApplyContainerHaConfV2(1:string cid, 2:ContainerHA cha),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器高可用配置项,详情参考 struct ContainerHA
	*/
	string ApplyContainerHaConfV3(1:string session_id, 2:string cid, 3:ContainerHA cha),

	//容器HA服务使能
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* param2: 容器高可用使能状态（启用或者关闭）
	*/
	string SetContainerHaEnable(1:string node_name, 2:string container_name, 3:bool ha_enable),
	/*
	* param1: 容器UUID 或者 模板UUID
	* param2: 容器高可用使能状态（启用或者关闭）
	*/
	string SetContainerHaEnableV2(1:string cid, 2:bool ha_enable),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 容器高可用使能状态（启用或者关闭）
	*/
	string SetContainerHaEnableV3(1:string session_id, 2:string cid, 3:bool ha_enable),

	//获取容器HA配置信息
	/*
	* @Deprecated: since 1.0.2
	* param1: 节点名称
	* param2: 容器名称
	* return:
	{
		"status":0,
		"message":"success",
		"container_ha_conf":{  		//容器高可用配置参数
			"ha_enable":true,			//高可用使能状态
			"ha_start_times":10,		//高可用最大启动次数限制
			"container_timeout":20,		//高可用中容器异常超时时间
			"ha_alarm_type":true,		//容器启动失败告警方式
			"nor_stop":true				//true表示正常停止的容器,false表示异常停止的容器
		}
	}
	*/
	string GetContainerHaConf(1:string node_name, 2:string container_name),
	/*
	* param1: 容器UUID 或者 模板UUID
	* return: 同上
	*/
	string GetContainerHaConfV2(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	*/
	string GetContainerHaConfV3(1:string session_id, 2:string cid),

	//接入告警信息接口
	/*
	* param1: 节点UUID
	* param2: 容器UUID
	* param3: 告警信息内容
	* param4: 告警信息类型
	* 注：
	* 1.平台节点的UUID，通过读取本地文件获取，文件定义为/var/lib/linx-scontainer/linxc-agent/nodeid。该文件仅存储node_uuid，即文本内容为：31e2c63f-7a05-4dc3-b476-66a06eda390b。
	* 2.容器的UUID在安装成功后默认写入其环境变量中，应用通过读取该环境变量获取，该环境变量名称定义为LINXSC_CONTAINER_UUID。
	* 3.告警类型分为两大块，平台告警与应用告警。每块划分5000个ID作为告警类型区分，对同类型未读告警，则当再次告警时，增加其告警次数times，更新其最后一次告警时间update_time，而不会新增一条告警信息。
	* 区间10000-14999为用户平台的告警ID，区间15000-19999为用户应用的告警ID。另新增独立ID=20000，使用该ID的告警信息不判断重复，直接新增告警信息。
	* 4.平台告警：仅传递nid即可,cid为空。 应用告警：仅传递cid即可，nid为空。 如同时传递了nid与cid参数，该条告警以平台告警处理。
	* 示例：
	* 添加平台告警(参数2为空)：
	* Alarm（"3b77f0e5-3d51-11eb-9cf4-0050568b3151"，""，"平台告警信息内容"，10000）
	* 添加应用告警(参数1为空)：
	* Alarm（""，"2984bb15-3d51-11eb-87a1-0050568bf080"，"应用告警信息内容"，15000）
	*/
	string Alarm(1:string nid, 2:string cid, 3:string msg, 4:i32 alarm_type),
	/*
	* param1: 用户session的id
	* param2: 节点UUID
	* param3: 容器UUID
	* param4: 告警信息内容
	* param5: 告警信息类型
	*/
	string AlarmV3(1:string session_id, 2:string nid, 3:string cid, 4:string msg, 5:i32 alarm_type),

	/*容器内进程相关*/

	//容器内进程查看
	/*
	* param1: 容器UUID
	* return:
	{
	"status":0,
	"message":"获取容器进程信息成功",
	"plist":[
	{
		"pid":1,
		"root":true,
		"child":[2,3],
		"cmdline":"",
		"cpu":15.5,
		"mem":4.5,
		"disk_read":1,
		"disk_write":"1.5"
	},
	{
		"pid":2,
		"root":false,
		"child":[],
		"cmdline":"process1",
		"cpu":15.5,
		"mem":4.5,
		"disk_read":1,
		"disk_write":"1.5"
	},
	{
		"pid":3,
		"root":false,
		"child":[4],
		"cmdline":"process2",
		"cpu":15.5,
		"mem":4.5,
		"disk_read":1,
		"disk_write":"1.5"
	},
	{
		"pid":4,
		"root":false,
		"child":[],
		"cmdline":"process3",
		"cpu":15.5,
		"mem":4.5,
		"disk_read":1,
		"disk_write":"1.5"
	}
	]
	}
	//注：
	//root为true时表示该pid为init_pid，为pstree中的根节点
	//plist中可能有多个root为true的pid，对应多个pstree

	失败时：
	{
	"status":1,
	"message":"获取容器进程信息失败",
	"plist":null
	}
	*/
	string ContainerProcessInfoList(1:string cid),
	/*
	* param1: 用户session的id
	* param2: 容器UUID
	*/
	string ContainerProcessInfoListV3(1:string session_id, 2:string cid),

	//任务方式应用模板到容器
	/*
	*param1:容器UUID
	*param2:模板UUID
	*param3:模板子项目应用选择
	return:
	{
		"status":0,
		"message":"success"
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskApplyACTemplate(1:string cid, 2:string tid, 3:TSelect tselect),
	/*
	*param1: 用户session的id
	*param2:容器UUID
	*param3:模板UUID
	*param4:模板子项目应用选择
	*/
	string TaskApplyACTemplateV3(1:string session_id, 2:string cid, 3:string tid, 4:TSelect tselect),

	//以任务方式从模板安装容器
	/*
	*param1:节点UUID
	*param2:镜像名称
	*param3:镜像版本
	*param4:容器名称
	*param5:容器描述信息
	*param6:模板UUID
	*param7:模板子项目应用选择
	return:
	{
		"status":0,
		"message":"success"
		"task_id":"0f850168-2570-11eb-adf3-005056b925b8"
	}
	*/
	string TaskInstallContainerFromTemplate(1:string nid, 2:string image, 3:string ver, 4:string name, 5:string desc, 6:string tid, 7:TSelect tselect),
	/*
	*param1:用户session的id
	*param2:节点UUID
	*param3:镜像名称
	*param4:镜像版本
	*param5:容器名称
	*param6:容器描述信息
	*param7:模板UUID
	*param8:模板子项目应用选择
	*/
	string TaskInstallContainerFromTemplateV3(1:string session_id, 2:string nid, 3:string image, 4:string ver, 5:string name, 6:string desc, 7:string tid, 8:TSelect tselect),

	//新增模板
	/*
	*param1:模板名称
	*param2:模板描述信息
	*/
	string AddACTemplate(1:string tname, 2:string tdesc),
	/*
	*param1:用户session的id
	*param2:模板名称
	*param3:模板描述信息
	*/
	string AddACTemplateV3(1:string session_id, 2:string tname, 3:string tdesc),

	//删除模板
	/*
	*param1:模板UUID
	*/
	string DelACTemplate(1:string tid),
	/*
	*param1:用户session的id
	*param2:模板UUID
	*/
	string DelACTemplateV3(1:string session_id, 2:string tid),

	//修改模板名称
	/*
	*param1:模板UUID
	*param2:模板新名称
	*param3:模板新描述信息
	*/
	string UpdateACTemplate(1:string tid, 2:string tname, 3:string tdesc),
	/*
	*param1:用户session的id
	*param2:模板UUID
	*param3:模板新名称
	*param4:模板新描述信息
	*/
	string UpdateACTemplateV3(1:string session_id, 2:string tid, 3:string tname, 4:string tdesc),

	//获取模板列表
	/*
	return:
	对于ipc、display、net_process_wl、ha等参数，true表示启用，false表示关闭；对process_wl、protected_file， 0表示启用 | 1表示关闭  | 2表示调试
	{
		"status":0,
		"message":"获取模板成功",
		"templates":[
		{
			"name":"template2",
			"uuid":"af878348-96cb-11eb-9295-0050568b3151",
			"desc":"这是第2个模板",
			"cpu":1,
			"mem":100,
			"disk_size":5,
			"disk_nums":4,
			"eth_nums":0,
			"ipc":false,
			"display":false,
			"net_process_wl":false,
			"process_wl":1,
			"protected_file":1,
			"ha":false,
			"level":5
		}
		]
	}
	*/
	string GetACTemplates(),
	/*
	*param1:用户session的id
	*/
	string GetACTemplatesV3(1:string session_id),

	//容器转化为模板
	/*
	*param1:容器UUID
	*param2:模板名称
	*param3:模板描述信息
	*/
	string ConvertACTemplate(1:string cid, 2:string tname, 3:string tdesc),
	/*
	*param1:用户session的id
	*param2:容器UUID
	*param3:模板名称
	*param4:模板描述信息
	*/
	string ConvertACTemplateV3(1:string session_id, 2:string cid, 3:string tname, 4:string tdesc),

	//导出模板
	/*
	*param1:容器UUID
	return:
	{
		"status":0,
		"message":"导出模板成功",
		"template":{……}
	}
	template字段内容为json，直接保存为模板文件内容即可
	*/
	string ExportACTemplate(1:string tid),
	/*
	*param1:用户session的id
	*param2:容器UUID
	*/
	string ExportACTemplateV3(1:string session_id, 2:string tid),

	//导入模板
	/*
	*param1:模板名称
	*param2:导出的模板配置文件中的json内容
	*/
	string ImportACTemplate(1:string name, 2:string tp_json),
	/*
	*param1:用户session的id
	*param2:模板名称
	*param3:导出的模板配置文件中的json内容
	*/
	string ImportACTemplateV3(1:string session_id, 2:string name, 3:string tp_json),

	//获取平台网卡列表
	/*
	return:
	{
		"status":0,
		"message":"获取可用平台网卡成功",
		"eths":[
			"eth1"
		]
	}
	*/
	string GetAvailbleEths(),
	/*
	*param1:用户session的id
	*/
	string GetAvailbleEthsV3(1:string session_id),

	/*-----------------------------------------------其他接口（内部使用）----------------------------------------------------*/
	/*获取linxc的版本号、序列号、支持的最大容器数等信息*/
	/*
	* return:
	{
		"status":0,
		"message":"success",
		"version":"1.1.2",						//linxc的版本号
		"serial_number":"287B61BF3F52C56F",		//序列号
		"max_containers":10						//最大支持的容器数
	}
	*/
	string GetLinxcInfo(),

	/*节点连接测试*/
	string ConnectTest(),

	//获取具体节点的安全容器系统集成版本信息与各个组件信息
	/*
	* return:
	{
		"status":0,
		"message":"success",
		"versions":{
			"linxc":"1.0.0",
			"linxc_ovs":"1.0.0",
			"linxc_agent":"1.0.0",
			"integrated":"R1.0.0"
		}
	}
	*/
	string NodeVersion(),

	//获取具体节点的平台网卡列表信息
	/*
	* return:
	{
		"status":0,
		"message":"获取平台网卡列表成功",
		"platform_eth":[						//json数组，平台网卡信息
			{
				"eth_name":"eth0",				//网卡名称，固定eth-platform
				"ip":"172.30.25.201",			//网卡ip
				"netmask":"255.255.255.0",		//网卡掩码
				"gateway":"172.30.25.254",		//网卡网关
				"mac":"00:50:56:c0:00:01",		//网卡mac地址
				"status":"正常"					//网卡状态，正常、异常
			}
		]
	}
	*/
	string NodeEths(),

	//设置容器图形启动中转发客户端并同时绑定获取到的客户端ip和cid
	
	string SetContainerDisplayIpAndCid(1:string cip, 2:string cid, 3:string stype),

	//测试用户密码过期接口
	/*
	*return:
	{
		"status":0,
		"message":"设置用户密码过期成功"
	}
	*/
	string SetUserExpire(1:string user, 2:bool expire),

	//容器软停止使能设置
	/*
	*param1:用户session的id
	*param2:容器UUID
	*param3:使能状态
	*/
	string SetContainerSoftStopEnable(1:string session_id, 2:string cid, 3:bool enable),

	//获取容器软停止使能状态
	/*
	*param1:用户session的id
	*param2:容器UUID
	*/
	string GetContainerSoftStopEnable(1:string session_id, 2:string cid),

	//容器软停止超时时间设置
	/*
	*param1:用户session的id
	*param2:容器UUID
	*param3:超时时间
	*/
	string SetContainerSoftStopTimeout(1:string session_id, 2:string cid, 3:i64 timeout),

	//获取容器软停止超时时间
	/*
	*param1:用户session的id
	*param2:容器UUID
	*/
	string GetContainerSoftStopTimeout(1:string session_id, 2:string cid),

	//添加容器网卡跳过容器ip冲突判断
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string AddContainerNetworkSkipClashCheck(1:string session_id, 2:string cid, 3:list<ContainerNetwork> net),

	// 更新容器网卡跳过容器ip冲突判断
	/*
	* param1: 用户session的id
	* param2: 容器UUID 或者 模板UUID
	* param3: 设置容器网卡的结构体，参考 struct ContainerNetwork
	*/
	string UpdateContainerNetworkSkipClashCheck(1:string session_id, 2:string cid, 3:list<ContainerNetwork> net),

	// 获取容器ip冲突列表
	string GetAllContainerIpClashList(1:string session_id),

        //容器的数据备份
        /*
        *param1:session_id: 用户session的id
        *param2:nid: 节点UUID
        *param3:cid: 容器UUID
        *param4:backup_path: 容器备份文件存放目录
        *param5:backup_name: 容器备份目录名称
        *param6:containerselect: 容器备份容器配置项选择
        */
        string BackupContainerDataV3(1:string session_id, 2:string nid, 3:string cid, 4:string backup_path, 5:string backup_name, 6:BackupSelect containerselect),

        //扫描备份目录
        /*
        *param1:session_id: 用户session的id
        *param2:nid: 节点UUID
        *param3:backup_path：容器备份文件存放目录
        */
        string GetContainerDataBackupListV3(1:string session_id, 2:string nid, 3:string backup_path),

        //容器的数据还原
        /*
        *param1:session_id: 用户session的id
        *param2:nid: 节点UUID
        *param3:backup_path: 容器备份文件存放目录
        *param4:backup_name:容器备份名称
        *param5:containerselect：容器备份容器配置项选择
        */
        string RestoreContainerDataV3(1:string session_id, 2:string nid, 3:string backup_path, 4:string backup_name, 5:BackupSelect containerselect)

}
