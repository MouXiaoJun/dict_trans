#!/bin/bash

# MySQL 示例运行脚本

echo "=== dict-trans MySQL 示例 ==="
echo ""

# 检查 MySQL 连接
echo "1. 检查 MySQL 连接..."
mysql -u root -pMSms0427 -h 127.0.0.1 -e "SELECT 1" > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ MySQL 连接失败，请检查："
    echo "   - MySQL 服务是否启动"
    echo "   - 用户名密码是否正确"
    exit 1
fi
echo "✓ MySQL 连接成功"
echo ""

# 创建数据库和表
echo "2. 初始化数据库..."
mysql -u root -pMSms0427 -h 127.0.0.1 < setup.sql > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo "❌ 数据库初始化失败"
    exit 1
fi
echo "✓ 数据库初始化成功"
echo ""

# 安装依赖
echo "3. 安装 Go 依赖..."
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ 依赖安装失败"
    exit 1
fi
echo "✓ 依赖安装成功"
echo ""

# 运行示例
echo "4. 运行示例..."
echo ""
go run main.go

