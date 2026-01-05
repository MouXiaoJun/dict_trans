-- 字典表结构（标准格式）
-- 适用于 dict-trans 的字典表翻译功能

USE `dict_trans`;

-- 创建字典表
CREATE TABLE IF NOT EXISTS `sys_dict` (
  `id` INT(11) NOT NULL AUTO_INCREMENT,
  `dict_type` VARCHAR(50) NOT NULL COMMENT '字典类型',
  `dict_key` VARCHAR(50) NOT NULL COMMENT '字典键',
  `dict_value` VARCHAR(200) NOT NULL COMMENT '字典值',
  `status` CHAR(1) DEFAULT '1' COMMENT '状态：1-启用，0-禁用',
  `sort_order` INT(11) DEFAULT 0 COMMENT '排序',
  `remark` VARCHAR(500) DEFAULT NULL COMMENT '备注',
  `create_time` DATETIME DEFAULT CURRENT_TIMESTAMP COMMENT '创建时间',
  `update_time` DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT '更新时间',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_dict_type_key` (`dict_type`, `dict_key`),
  KEY `idx_dict_type` (`dict_type`),
  KEY `idx_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='系统字典表';

-- 插入测试数据
INSERT INTO `sys_dict` (`dict_type`, `dict_key`, `dict_value`, `status`, `sort_order`, `remark`) VALUES
-- 性别字典
('sex', '1', '男', '1', 1, '性别-男'),
('sex', '2', '女', '1', 2, '性别-女'),
-- 状态字典
('status', '0', '禁用', '1', 1, '状态-禁用'),
('status', '1', '启用', '1', 2, '状态-启用'),
-- 优先级字典
('priority', '1', '低', '1', 1, '优先级-低'),
('priority', '2', '中', '1', 2, '优先级-中'),
('priority', '3', '高', '1', 3, '优先级-高'),
-- 设备状态字典
('device_status', '1', '未使用', '1', 1, '设备状态-未使用'),
('device_status', '2', '试运行', '1', 2, '设备状态-试运行'),
('device_status', '3', '运行中', '1', 3, '设备状态-运行中'),
('device_status', '4', '已停用', '1', 4, '设备状态-已停用'),
-- 订单状态字典
('order_status', '1', '待支付', '1', 1, '订单状态-待支付'),
('order_status', '2', '已支付', '1', 2, '订单状态-已支付'),
('order_status', '3', '已发货', '1', 3, '订单状态-已发货'),
('order_status', '4', '已完成', '1', 4, '订单状态-已完成'),
('order_status', '5', '已取消', '1', 5, '订单状态-已取消')
ON DUPLICATE KEY UPDATE `dict_value`=VALUES(`dict_value`);

-- 查询验证
SELECT '字典表数据:' AS '';
SELECT `dict_type`, `dict_key`, `dict_value`, `status` FROM `sys_dict` ORDER BY `dict_type`, `sort_order`;

