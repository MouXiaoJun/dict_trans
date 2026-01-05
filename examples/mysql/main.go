package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mouxiaojun/dict-trans"
)

func main() {
	fmt.Println("=== MySQL 数据库翻译示例 ===\n")

	// 连接 MySQL 数据库
	dsn := "root:MSms0427@tcp(127.0.0.1:3306)/dict_trans?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v\n请确保:\n1. MySQL 服务已启动\n2. 数据库 dict_trans 已创建\n3. 已运行 setup.sql 创建测试数据", err)
	}
	fmt.Println("✓ 数据库连接成功\n")

	// 注册数据库翻译器
	dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		// 构建 SQL 查询
		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", valueField, table, keyField)

		var result string
		err := db.QueryRow(query, key).Scan(&result)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", nil // 未找到记录，返回空字符串
			}
			return "", fmt.Errorf("查询失败: %v", err)
		}

		return result, nil
	}))

	// ========== 示例1: 用户表翻译 ==========
	fmt.Println("【示例1】用户表翻译")
	type User struct {
		UserID   string `db:"user:id:name" dictField:"UserName"`
		UserName string
	}

	user := User{UserID: "1"}
	if err := dict.Translate(&user); err != nil {
		log.Printf("翻译失败: %v", err)
	} else {
		fmt.Printf("  用户ID %s -> 用户名: %s\n", user.UserID, user.UserName)
	}

	// ========== 示例2: 部门表翻译 ==========
	fmt.Println("\n【示例2】部门表翻译")
	type Employee struct {
		DeptID   string `db:"table=department,key=id,value=name" dictField:"DeptName"`
		DeptName string
	}

	emp := Employee{DeptID: "1"}
	if err := dict.Translate(&emp); err != nil {
		log.Printf("翻译失败: %v", err)
	} else {
		fmt.Printf("  部门ID %s -> 部门名: %s\n", emp.DeptID, emp.DeptName)
	}

	// ========== 示例3: 测试缓存 ==========
	fmt.Println("\n【示例3】缓存测试")
	user2 := User{UserID: "1"}
	if err := dict.Translate(&user2); err != nil {
		log.Printf("翻译失败: %v", err)
	} else {
		fmt.Printf("  用户ID %s -> 用户名: %s (使用缓存)\n", user2.UserID, user2.UserName)
	}

	// ========== 示例4: 批量翻译 ==========
	fmt.Println("\n【示例4】批量翻译")
	type UserList struct {
		UserID   string `db:"user:id:name" dictField:"UserName"`
		UserName string
	}

	users := []UserList{
		{UserID: "1"},
		{UserID: "2"},
		{UserID: "3"},
	}
	if err := dict.Translate(&users); err != nil {
		log.Printf("翻译失败: %v", err)
	} else {
		for _, u := range users {
			fmt.Printf("  用户ID %s -> 用户名: %s\n", u.UserID, u.UserName)
		}
	}

	// ========== 示例5: 混合使用（字典+数据库） ==========
	fmt.Println("\n【示例5】混合使用（字典+数据库）")
	dict.RegisterDict("status", map[string]string{
		"1": "启用",
		"0": "禁用",
	})

	type UserWithStatus struct {
		UserID     string `db:"user:id:name" dictField:"UserName"`
		UserName   string
		Status     string `dict:"status" dictField:"StatusName"`
		StatusName string
	}

	userWithStatus := UserWithStatus{
		UserID: "1",
		Status: "1",
	}
	if err := dict.Translate(&userWithStatus); err != nil {
		log.Printf("翻译失败: %v", err)
	} else {
		fmt.Printf("  用户ID %s -> 用户名: %s\n", userWithStatus.UserID, userWithStatus.UserName)
		fmt.Printf("  状态 %s -> %s\n", userWithStatus.Status, userWithStatus.StatusName)
	}

	fmt.Println("\n✓ 所有示例执行完成")
}
