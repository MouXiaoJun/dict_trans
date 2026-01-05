-- dict-trans MySQL 示例数据库初始化脚本

-- 创建数据库（如果不存在）
CREATE DATABASE IF NOT EXISTS `dict_trans` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

USE `dict_trans`;

-- 创建用户表
CREATE TABLE IF NOT EXISTS `user` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(50) NOT NULL COMMENT '用户名',
  `email` VARCHAR(100) DEFAULT NULL COMMENT '邮箱',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='用户表';

-- 创建部门表
CREATE TABLE IF NOT EXISTS `department` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `name` VARCHAR(50) NOT NULL COMMENT '部门名称',
  `code` VARCHAR(20) DEFAULT NULL COMMENT '部门编码',
  PRIMARY KEY (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='部门表';

-- 插入测试数据
INSERT INTO `user` (`id`, `name`, `email`) VALUES
(1, '张三', 'zhangsan@example.com'),
(2, '李四', 'lisi@example.com'),
(3, '王五', 'wangwu@example.com'),
(4, '赵六', 'zhaoliu@example.com'),
(5, '钱七', 'qianqi@example.com')
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

INSERT INTO `department` (`id`, `name`, `code`) VALUES
(1, '技术部', 'TECH'),
(2, '产品部', 'PROD'),
(3, '运营部', 'OPS'),
(4, '市场部', 'MKT'),
(5, '人事部', 'HR')
ON DUPLICATE KEY UPDATE `name`=VALUES(`name`);

-- 查询验证
SELECT '用户表数据:' AS '';
SELECT * FROM `user`;

SELECT '部门表数据:' AS '';
SELECT * FROM `department`;

