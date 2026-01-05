# MySQL 数据库翻译示例

这个示例演示如何使用 dict-trans 进行 MySQL 数据库翻译。

## 前置要求

1. 安装 MySQL 数据库
2. 安装 Go MySQL 驱动：
   ```bash
   go get github.com/go-sql-driver/mysql
   ```

## 数据库配置

- 主机: 127.0.0.1
- 端口: 3306
- 用户名: root
- 密码: MSms0427
- 数据库: dict_trans

## 使用步骤

### 1. 创建数据库和表

```bash
mysql -u root -pMSms0427 < setup.sql
```

或者手动执行 SQL：

```sql
CREATE DATABASE IF NOT EXISTS dict_trans;
USE dict_trans;
-- 然后执行 setup.sql 中的 SQL 语句
```

### 2. 运行示例

```bash
cd examples/mysql
go run main.go
```

## 示例说明

### main.go - 数据库表翻译示例
- **示例1**: 用户表翻译 - 使用 `db:"user:id:name"` 标签从 user 表查询用户名
- **示例2**: 部门表翻译 - 使用完整格式查询部门名
- **示例3**: 缓存测试 - 第二次查询相同的数据会使用缓存
- **示例4**: 批量翻译 - 批量翻译多个用户
- **示例5**: 混合使用 - 同时使用字典翻译和数据库翻译

### dict_table_example.go - 字典表翻译示例 ⭐
- **示例1**: 性别字典翻译 - 从字典表读取性别数据
- **示例2**: 状态字典翻译 - 从字典表读取状态数据
- **示例3**: 优先级字典翻译 - 从字典表读取优先级数据
- **示例4**: 设备状态字典翻译 - 从字典表读取设备状态
- **示例5**: 缓存测试 - 字典表查询结果缓存
- **示例6**: 批量翻译 - 批量翻译字典表数据
- **示例7**: 混合使用 - 字典表+数据库表混合翻译
- **示例8**: 禁用缓存测试

## 字典表翻译使用步骤

### 单表字典（dictTable）

1. **创建字典表**（运行 dict_table.sql）：
   ```bash
   mysql -u root -pMSms0427 < dict_table.sql
   ```

2. **运行单表字典示例**：
   ```bash
   go run dict_table_example.go
   ```

### 双表字典（dictTableTwo）⭐

1. **创建双表字典**（运行 dict_table_two.sql）：
   ```bash
   mysql -u root -pMSms0427 < dict_table_two.sql
   ```

2. **运行双表字典示例**：
   ```bash
   go run dict_table_two_example.go
   ```

## 单表字典 vs 双表字典

| 特性 | 单表字典 (`dictTable`) | 双表字典 (`dictTableTwo`) |
|------|---------------------|------------------------|
| 表结构 | 1张表 (sys_dict) | 2张表 (sys_dict_type + sys_dict_data) |
| 适用场景 | 简单字典场景 | 复杂字典场景 |
| 类型管理 | 无 | 支持类型元数据管理 |
| 优势 | 结构简单，查询快速 | 更规范，支持类型独立管理 |
| 使用方式 | `dictTable:"sex"` | `dictTableTwo:"sex"` |

**推荐使用双表字典**，更符合数据库规范化设计，支持字典类型的独立管理。

## 注意事项

1. 确保 MySQL 服务已启动
2. 确保数据库连接信息正确
3. 数据库翻译结果会自动缓存，可以通过 `dict.ClearDBCache()` 清空缓存
4. 可以通过 `dict.EnableDBCache(false)` 禁用缓存

