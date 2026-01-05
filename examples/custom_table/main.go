package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
	"github.com/mouxiaojun/dict-trans"
)

func main() {
	fmt.Println("=== 自定义表结构示例 ===\n")

	// 连接 MySQL 数据库
	dsn := "root:MSms0427@tcp(127.0.0.1:3306)/dict_trans?charset=utf8mb4&parseTime=True&loc=Local"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		log.Fatalf("数据库连接测试失败: %v\n请确保:\n1. MySQL 服务已启动\n2. 数据库 dict_trans 已创建", err)
	}
	fmt.Println("✓ 数据库连接成功\n")

	// ========== 示例1: 使用默认表结构 ==========
	fmt.Println("【示例1】使用默认表结构")
	example1_DefaultTable(db)

	// ========== 示例2: 自定义单表字典结构 ==========
	fmt.Println("\n【示例2】自定义单表字典结构")
	example2_CustomSingleTable(db)

	// ========== 示例3: 自定义双表字典结构 ==========
	fmt.Println("\n【示例3】自定义双表字典结构")
	example3_CustomTwoTable(db)

	// ========== 示例4: 自定义状态字段值 ==========
	fmt.Println("\n【示例4】自定义状态字段值")
	example4_CustomStatus(db)
}

// 示例1: 使用默认表结构
func example1_DefaultTable(db *sql.DB) {
	// 使用默认配置（sys_dict 表，字段：dict_type, dict_key, dict_value, status）
	translator := dict.CreateDictTableTranslatorFromDB(db, "sys_dict")
	dict.RegisterDictTableTranslator(translator)

	type User struct {
		Sex     string `dictTable:"sex" dictField:"SexName"`
		SexName string
	}

	user := User{Sex: "1"}
	dict.Translate(&user)
	fmt.Printf("  性别 %s -> %s (使用默认表结构)\n", user.Sex, user.SexName)
}

// 示例2: 自定义单表字典结构
func example2_CustomSingleTable(db *sql.DB) {
	// 假设你的表结构是：
	// CREATE TABLE custom_dict (
	//   type_code VARCHAR(50),
	//   code VARCHAR(50),
	//   label VARCHAR(200),
	//   is_active CHAR(1)
	// )

	config := &dict.TableConfig{
		TableName: "custom_dict",
		Fields: dict.TableFields{
			TypeField:  "type_code", // 自定义类型字段名
			KeyField:   "code",      // 自定义键字段名
			ValueField: "label",     // 自定义值字段名
		},
		StatusField: &dict.StatusFieldConfig{
			FieldName:     "is_active",
			EnabledValue:  "Y", // 启用值是 "Y"
			DisabledValue: "N", // 禁用值是 "N"
		},
	}

	translator := dict.CreateDictTableTranslatorFromDBWithConfig(db, config)
	dict.RegisterDictTableTranslator(translator)

	type Item struct {
		Category     string `dictTable:"product_type" dictField:"CategoryName"`
		CategoryName string
	}

	item := Item{Category: "001"}
	dict.Translate(&item)
	fmt.Printf("  分类 %s -> %s (使用自定义表结构)\n", item.Category, item.CategoryName)
}

// 示例3: 自定义双表字典结构
func example3_CustomTwoTable(db *sql.DB) {
	// 假设你的表结构是：
	// CREATE TABLE dict_category (
	//   category_code VARCHAR(50),
	//   category_name VARCHAR(100),
	//   enabled TINYINT
	// )
	// CREATE TABLE dict_item (
	//   category_code VARCHAR(50),
	//   item_code VARCHAR(50),
	//   item_name VARCHAR(200),
	//   enabled TINYINT
	// )

	typeConfig := &dict.TableConfig{
		TableName: "dict_category",
		Fields: dict.TableFields{
			TypeField:  "category_code",
			ValueField: "category_name",
		},
		StatusField: &dict.StatusFieldConfig{
			FieldName:     "enabled",
			EnabledValue:  "1",
			DisabledValue: "0",
		},
	}

	dataConfig := &dict.TableConfig{
		TableName: "dict_item",
		Fields: dict.TableFields{
			TypeField:  "category_code",
			KeyField:   "item_code",
			ValueField: "item_name",
		},
		StatusField: &dict.StatusFieldConfig{
			FieldName:     "enabled",
			EnabledValue:  "1",
			DisabledValue: "0",
		},
	}

	translator := dict.CreateDictTableTwoTranslatorFromDBWithConfig(db, typeConfig, dataConfig)
	dict.RegisterDictTableTwoTranslator(translator)

	type Product struct {
		Status     string `dictTableTwo:"order_status" dictField:"StatusName"`
		StatusName string
	}

	product := Product{Status: "1"}
	dict.Translate(&product)
	fmt.Printf("  状态 %s -> %s (使用自定义双表结构)\n", product.Status, product.StatusName)
}

// 示例4: 自定义状态字段值
func example4_CustomStatus(db *sql.DB) {
	// 假设你的表使用不同的状态值：true/false, Y/N, 1/0 等

	config := &dict.TableConfig{
		TableName: "status_dict",
		Fields: dict.TableFields{
			TypeField:  "type",
			KeyField:   "key",
			ValueField: "value",
		},
		StatusField: &dict.StatusFieldConfig{
			FieldName:     "active",
			EnabledValue:  "true", // 使用布尔字符串
			DisabledValue: "false",
		},
	}

	translator := dict.CreateDictTableTranslatorFromDBWithConfig(db, config)
	dict.RegisterDictTableTranslator(translator)

	fmt.Println("  已配置自定义状态字段值（true/false）")
}
