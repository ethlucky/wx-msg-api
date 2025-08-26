-- 更新现有表的字符集和排序规则为utf8mb4_0900_ai_ci
-- 执行前请备份数据库

-- 更新微信机器人配置表
ALTER TABLE `wx_robot_configs` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 更新微信用户登录信息表
ALTER TABLE `wx_user_logins` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 更新微信群列表表
ALTER TABLE `wx_groups` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 更新微信账单信息表
ALTER TABLE `wx_bill_info` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;

-- 更新微信群消息表
ALTER TABLE `wx_group_messages` CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;