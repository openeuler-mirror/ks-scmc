CREATE DATABASE IF NOT EXISTS `ks-scmc`
CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;	-- utf8mb4_0900_ai_ci utf8_general_ci

USE `ks-scmc`;

CREATE TABLE `user_infos` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `username` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `real_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `password_en` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_active` tinyint(1) NOT NULL DEFAULT '0',
  `is_editable` tinyint(1) NOT NULL DEFAULT '0',
  `role_id` bigint(20) NOT NULL DEFAULT '0',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `username` (`username`),
  KEY `role_id` (`role_id`)
) ENGINE=InnoDB AUTO_INCREMENT=1000 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_roles` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `is_editable` tinyint(1) NOT NULL DEFAULT '0',
  `perms_json` text COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '权限数组 用JSON形式保存 [{"id":1, "name":"NODE", "read":1, "write":0}, ...]',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `user_sessions` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `user_id` bigint(20) unsigned NOT NULL,
  `session_key` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `source` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `expired_at` int(20) NOT NULL DEFAULT '0',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `user_id` (`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `node_infos` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `address` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `comment` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `unread_warn` bigint(20) NOT NULL DEFAULT '0' COMMENT '未读告警消息计数 ',
  `cpu_limit` decimal(10,1) NOT NULL DEFAULT '0.0' COMMENT 'CPU核心数',
  `memory_limit` decimal(20,1) NOT NULL DEFAULT '0.0' COMMENT '内存使用 单位MB',
  `disk_limit` decimal(20,1) NOT NULL DEFAULT '0.0' COMMENT '磁盘使用 单位MB',
  `deleted` tinyint(4) NOT NULL DEFAULT '0' COMMENT '删除标记位 0:未删除',
  `uuid` varchar(255) NOT NULL DEFAULT '' COMMENT '节点添加/删除时使用, 防止节点被不同集群的controller服务添加',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`),
  UNIQUE KEY `address` (`address`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `image_infos` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `version` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `description` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `file_size` int(20) NOT NULL DEFAULT '0',
  `file_type` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `check_sum` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '',
  `image_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'docker镜像ID',
  `file_path` text COLLATE utf8mb4_unicode_ci,
  `sign_path` text COLLATE utf8mb4_unicode_ci,  /* deprecated */
  `reject_reason` text COLLATE utf8mb4_unicode_ci,
  `approval_status` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0:待审批 1:审批拒绝 2:审批通过',
  `verify_status` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0:验签失败 1:异常 2:验签成功',
  `signature` blob COMMENT '镜像文件签名内容',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `index_unique` (`name`,`version`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `runtime_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `event_type` int(10) NOT NULL DEFAULT '0' COMMENT '事件类型',
  `event_type_` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT 'event_type调试',
  `event_module` int(10) NOT NULL DEFAULT '0' COMMENT '事件模块',
  `node_id` bigint(20) NOT NULL DEFAULT '0',
  `node_info` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '节点名+节点IP',
  `user_id` bigint(20) NOT NULL DEFAULT '0' COMMENT '操作用户ID',
  `target` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '操作对象',
  `detail` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '详情',
  `status_code` int(10) NOT NULL DEFAULT '0' COMMENT '错误码 0:OK',
  `error` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '错误信息',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `warn_logs` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `event_type` int(10) NOT NULL DEFAULT '0' COMMENT '事件类型',
  `event_module` int(10) NOT NULL DEFAULT '0' COMMENT '事件模块',
  `node_id` bigint(20) NOT NULL DEFAULT '0',
  `node_info` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '节点名+节点IP',
  `container_id` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '容器ID',
  `container_name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '容器ID',
  `detail` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '详情',
  `have_read` tinyint(1) NOT NULL DEFAULT '0' COMMENT '0:未读 1:已读',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE `container_templates` (
  `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT,
  `name` varchar(255) COLLATE utf8mb4_unicode_ci NOT NULL DEFAULT '' COMMENT '容器名',
  `node_id` bigint(20) NOT NULL DEFAULT '0' COMMENT '节点ID',
  `config_json` text COLLATE utf8mb4_unicode_ci NOT NULL COMMENT '配置参数 以JSON形式存储',
  `created_at` int(20) NOT NULL DEFAULT '0',
  `updated_at` int(20) NOT NULL DEFAULT '0',
  PRIMARY KEY (`id`),
  UNIQUE KEY `name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS `security_switches` (
  `id` BIGINT(20) UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
  `is_on` TINYINT NOT NULL UNIQUE DEFAULT 0 COMMENT '安全开关状态 0:关 1:开',
  `created_at` INT(20) NOT NULL DEFAULT 0,
  `updated_at` INT(20) NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

