# 自定义表结构示例

本示例展示如何使用自定义表结构进行字典翻译。

## 功能说明

dict-trans 支持完全自定义表结构，包括：

- ✅ 自定义表名
- ✅ 自定义字段名（类型字段、键字段、值字段）
- ✅ 自定义状态字段名和值
- ✅ 支持单表字典和双表字典

## 使用场景

### 场景1: 使用默认表结构

如果你的表结构是标准的 `sys_dict` 表：

```sql
CREATE TABLE sys_dict (
  dict_type VARCHAR(50),
  dict_key VARCHAR(50),
  dict_value VARCHAR(200),
  status CHAR(1)
);
```

直接使用默认配置：

```go
translator := dict.CreateDictTableTranslatorFromDB(db, "sys_dict")
dict.RegisterDictTableTranslator(translator)
```

### 场景2: 自定义字段名

如果你的表字段名不同：

```sql
CREATE TABLE custom_dict (
  type_code VARCHAR(50),  -- 类型字段
  code VARCHAR(50),      -- 键字段
  label VARCHAR(200),    -- 值字段
  is_active CHAR(1)      -- 状态字段
);
```

使用自定义配置：

```go
config := &dict.TableConfig{
    TableName: "custom_dict",
    Fields: dict.TableFields{
        TypeField:  "type_code",
        KeyField:   "code",
        ValueField: "label",
    },
    StatusField: &dict.StatusFieldConfig{
        FieldName:     "is_active",
        EnabledValue:  "Y",
        DisabledValue: "N",
    },
}

translator := dict.CreateDictTableTranslatorFromDBWithConfig(db, config)
dict.RegisterDictTableTranslator(translator)
```

### 场景3: 自定义状态值

如果你的表使用不同的状态值：

```go
config := &dict.TableConfig{
    TableName: "status_dict",
    Fields: dict.TableFields{
        TypeField:  "type",
        KeyField:   "key",
        ValueField: "value",
    },
    StatusField: &dict.StatusFieldConfig{
        FieldName:     "active",
        EnabledValue:  "true",   // 布尔字符串
        DisabledValue: "false",
    },
}
```

### 场景4: 无状态字段

如果你的表没有状态字段：

```go
config := &dict.TableConfig{
    TableName: "simple_dict",
    Fields: dict.TableFields{
        TypeField:  "type",
        KeyField:   "key",
        ValueField: "value",
    },
    StatusField: nil, // 不配置状态字段
}
```

## 运行示例

```bash
cd examples/custom_table
go run main.go
```

## 更多信息

查看 [README.md](../../README.md) 了解完整的 API 文档。

