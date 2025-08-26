-- 微信机器人管理系统数据库表结构

-- 微信机器人配置表
CREATE TABLE `wx_robot_configs` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `address` varchar(255) NOT NULL COMMENT '机器人地址',
    `admin_key` varchar(255) NOT NULL COMMENT '管理密钥',
    `owner_id` bigint(20) unsigned NOT NULL COMMENT '所属公司ID',
    `description` varchar(500) COMMENT '文本描述',
    `admin_users` text COMMENT '管理员用户列表，用逗号分隔',
    `create_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `update_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '修改时间',
    PRIMARY KEY (`id`),
    INDEX `idx_owner_id` (`owner_id`),
    INDEX `idx_create_time` (`create_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='微信机器人配置表';

-- 微信用户登录信息表
CREATE TABLE `wx_user_logins` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `robot_id` bigint(20) unsigned NOT NULL COMMENT '关联的机器人ID',
    `token` varchar(500) DEFAULT NULL COMMENT '登录令牌',
    `wx_id` varchar(100) DEFAULT NULL COMMENT '微信ID',
    `nick_name` varchar(100) DEFAULT NULL COMMENT '微信昵称',
    `extension_time` datetime(3) DEFAULT NULL COMMENT '延期时间',
    `has_security_risk` tinyint(1) DEFAULT '0' COMMENT '是否有安全风险 0否 1是',
    `expiration_time` datetime(3) DEFAULT NULL COMMENT '过期时间',
    `status` int(11) DEFAULT '1' COMMENT '状态 1正常 2风控 3过期',
    `is_initialized` int(11) DEFAULT '0' COMMENT '是否初始化完成 0未初始化 1初始化完成',
    `is_message_bot` int(11) DEFAULT '0' COMMENT '是否是消息机器人 0不是 1是',
    `create_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `update_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '修改时间',
    PRIMARY KEY (`id`),
    -- 复合索引：优先级高的查询组合
    INDEX `idx_robot_wx` (`robot_id`, `wx_id`),
    INDEX `idx_robot_token` (`robot_id`, `token`),
    INDEX `idx_init_status` (`is_initialized`, `status`),
    INDEX `idx_wx_status_msgbot_risk` (`wx_id`, `status`, `is_message_bot`, `has_security_risk`),
    INDEX `idx_status_msgbot_risk` (`status`, `is_message_bot`, `has_security_risk`),
    -- 单列索引：为其他查询模式保留
    INDEX `idx_wx_id` (`wx_id`),
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='微信用户登录信息表';

-- 微信群列表表
CREATE TABLE `wx_groups` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '主键ID',
    `wx_id` varchar(100) NOT NULL COMMENT '微信ID',
    `group_id` varchar(100) NOT NULL COMMENT '群组ID',
    `group_nick_name` varchar(200) DEFAULT NULL COMMENT '群组昵称',
    `create_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `update_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '修改时间',
    PRIMARY KEY (`id`),
    -- 复合索引：覆盖主要查询模式
    INDEX `idx_group_wx` (`group_id`, `wx_id`),
    -- 单列索引：LIKE搜索和时间查询
    INDEX `idx_wx_id` (`wx_id`),
    INDEX `idx_group_nick_name` (`group_nick_name`),
    INDEX `idx_create_time` (`create_time`),
    UNIQUE KEY `uk_wx_group` (`wx_id`, `group_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='微信群列表表';

-- 微信账单信息表
CREATE TABLE `wx_bill_info` (
    `id` bigint(20) unsigned NOT NULL AUTO_INCREMENT COMMENT '账单ID',
    `group_name` varchar(50) NOT NULL COMMENT '群组名称',
    `group_id` varchar(50) NOT NULL COMMENT '群组Id',
    `dollar` varchar(20) DEFAULT NULL COMMENT '金额(外币)',
    `rate` varchar(20) DEFAULT NULL COMMENT '汇率',
    `amount` decimal(15,2) DEFAULT NULL COMMENT '金额(RMB)',
    `remark` text DEFAULT NULL COMMENT '备注',
    `operator` varchar(20) DEFAULT NULL COMMENT '操作人名称',
    `msg_time` bigint(20) DEFAULT NULL COMMENT '账单时间',
    `status` char(2) DEFAULT NULL COMMENT '清账状态(0 为未清账, 1 为已清账)',
    `owner_id` bigint(20) unsigned NOT NULL COMMENT '所属公司ID',
    `create_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) COMMENT '创建时间',
    `update_time` datetime(3) DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3) COMMENT '修改时间',
    PRIMARY KEY (`id`),
    -- 复合索引：优化常用查询组合
    INDEX `idx_owner_group` (`owner_id`, `group_id`),
    INDEX `idx_owner_status` (`owner_id`, `status`),
    INDEX `idx_owner_msgtime` (`owner_id`, `msg_time`),
    -- 单列索引：GROUP BY和LIKE查询
    INDEX `idx_group_name` (`group_name`),
    INDEX `idx_msg_time` (`msg_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='wx账单源表';


-- 插入示例数据（可选）
-- INSERT INTO `wx_robot_configs` (`address`, `admin_key`, `owner_id`) VALUES 
-- ('http://127.0.0.1:8080', 'admin_key_123', 1),
-- ('http://192.168.1.100:8080', 'admin_key_456', 2);