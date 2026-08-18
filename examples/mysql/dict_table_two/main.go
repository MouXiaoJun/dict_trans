package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mouxiaojun/dict-trans"
)

func main() {
	fmt.Println("=== 双表字典翻译示例（字典类型表 + 字典数据表）===\n")

	// 连接 MySQL 数据库
	dsn := "root:MSms0427@tcp(127.0.0.1:3306)/dict_trans?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v\n请确保:\n1. MySQL 服务已启动\n2. 数据库 dict_trans 已创建\n3. 已运行 dict_table_two.sql 创建字典表", err)
	}
	fmt.Println("✓ 数据库连接成功\n")

	// 注册双表字典翻译器（从 sys_dict_type 和 sys_dict_data 表读取）
	dictTableTwoTranslator := dict.CreateDictTableTwoTranslatorFromDB(db, "sys_dict_type", "sys_dict_data")
	dict.RegisterDictTableTwoTranslator(dictTableTwoTranslator)

	// ========== 示例1: 性别字典 ==========
	fmt.Println("【示例1】性别字典翻译（双表结构）")
	type User struct {
		Sex     string `dictTableTwo:"sex" dictField:"SexName"`
		SexName string
	}

	user := User{Sex: "1"}
	if err := dict.Translate(&user); err != nil {
		log.Printf("翻译失败: %v", err)
	} else {
		fmt.Printf("  性别 %s -> %s\n", user.Sex, user.SexName)
	}

	user2 := User{Sex: "2"}
	dict.Translate(&user2)
	fmt.Printf("  性别 %s -> %s\n", user2.Sex, user2.SexName)

	// ========== 示例2: 状态字典 ==========
	fmt.Println("\n【示例2】状态字典翻译")
	type Item struct {
		Status     string `dictTableTwo:"status" dictField:"StatusName"`
		StatusName string
	}

	item := Item{Status: "1"}
	dict.Translate(&item)
	fmt.Printf("  状态 %s -> %s\n", item.Status, item.StatusName)

	item2 := Item{Status: "0"}
	dict.Translate(&item2)
	fmt.Printf("  状态 %s -> %s\n", item2.Status, item2.StatusName)

	// ========== 示例3: 优先级字典 ==========
	fmt.Println("\n【示例3】优先级字典翻译")
	type Task struct {
		Priority     string `dictTableTwo:"priority" dictField:"PriorityName"`
		PriorityName string
	}

	task := Task{Priority: "3"}
	dict.Translate(&task)
	fmt.Printf("  优先级 %s -> %s\n", task.Priority, task.PriorityName)

	// ========== 示例4: 设备状态字典 ==========
	fmt.Println("\n【示例4】设备状态字典翻译")
	type Device struct {
		Status     string `dictTableTwo:"device_status" dictField:"StatusName"`
		StatusName string
	}

	device := Device{Status: "2"}
	dict.Translate(&device)
	fmt.Printf("  设备状态 %s -> %s\n", device.Status, device.StatusName)

	// ========== 示例5: 测试缓存 ==========
	fmt.Println("\n【示例5】缓存测试")
	user3 := User{Sex: "1"}
	dict.Translate(&user3)
	fmt.Printf("  性别 %s -> %s (使用缓存，不会重复查询数据库)\n", user3.Sex, user3.SexName)

	// ========== 示例6: 批量翻译 ==========
	fmt.Println("\n【示例6】批量翻译")
	users := []User{
		{Sex: "1"},
		{Sex: "2"},
		{Sex: "1"},
	}
	dict.Translate(&users)
	for i, u := range users {
		fmt.Printf("  用户%d: 性别 %s -> %s\n", i+1, u.Sex, u.SexName)
	}

	// ========== 示例7: 混合使用（双表字典+数据库表） ==========
	fmt.Println("\n【示例7】混合使用（双表字典+数据库表）")

	// 注册数据库翻译器（用于查询用户表）
	dict.RegisterDBTranslator(dict.DBTranslatorFunc(func(table, keyField, valueField string, key interface{}) (string, error) {
		query := fmt.Sprintf("SELECT %s FROM %s WHERE %s = ?", valueField, table, keyField)
		var result string
		err := db.QueryRow(query, key).Scan(&result)
		if err != nil {
			if err == sql.ErrNoRows {
				return "", nil
			}
			return "", err
		}
		return result, nil
	}))

	type Employee struct {
		UserID     string `db:"user:id:name" dictField:"UserName"`
		UserName   string
		Sex        string `dictTableTwo:"sex" dictField:"SexName"`
		SexName    string
		Status     string `dictTableTwo:"status" dictField:"StatusName"`
		StatusName string
	}

	emp := Employee{
		UserID: "1",
		Sex:    "1",
		Status: "1",
	}
	dict.Translate(&emp)
	fmt.Printf("  员工信息:\n")
	fmt.Printf("    用户ID %s -> 用户名: %s (从user表查询)\n", emp.UserID, emp.UserName)
	fmt.Printf("    性别 %s -> %s (从双表字典查询)\n", emp.Sex, emp.SexName)
	fmt.Printf("    状态 %s -> %s (从双表字典查询)\n", emp.Status, emp.StatusName)

	// ========== 示例8: 禁用缓存 ==========
	fmt.Println("\n【示例8】禁用缓存测试")
	dict.EnableDictTableTwoCache(false)
	user4 := User{Sex: "1"}
	dict.Translate(&user4)
	fmt.Printf("  性别 %s -> %s (缓存已禁用，会重新查询数据库)\n", user4.Sex, user4.SexName)

	// 重新启用缓存
	dict.EnableDictTableTwoCache(true)

	// ========== 示例9: 对比单表字典和双表字典 ==========
	fmt.Println("\n【示例9】对比：单表字典 vs 双表字典")
	fmt.Println("  单表字典 (dictTable): 使用 sys_dict 表，dict_type 字段区分类型")
	fmt.Println("  双表字典 (dictTableTwo): 使用 sys_dict_type + sys_dict_data 两张表")
	fmt.Println("  双表字典的优势：")
	fmt.Println("    - 字典类型信息更完整（名称、排序、备注等）")
	fmt.Println("    - 字典类型可以独立管理（启用/禁用）")
	fmt.Println("    - 更符合数据库规范化设计")

	fmt.Println("\n✓ 所有示例执行完成")
}
