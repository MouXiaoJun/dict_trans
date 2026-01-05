package dict

import (
	"strings"
)

// parseDBTag 解析数据库翻译标签
// 格式: db:"table=user,key=id,value=name"
// 或: db:"user:id:name" (简化格式)
func parseDBTag(tag string) Translator {
	if tag == "" {
		return nil
	}

	var table, keyField, valueField string

	// 检查是否是简化格式 "table:key:value"
	if strings.Contains(tag, ":") && !strings.Contains(tag, "=") {
		parts := strings.Split(tag, ":")
		if len(parts) == 3 {
			table = strings.TrimSpace(parts[0])
			keyField = strings.TrimSpace(parts[1])
			valueField = strings.TrimSpace(parts[2])
		}
	} else {
		// 解析完整格式 "table=user,key=id,value=name"
		parts := strings.Split(tag, ",")
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if strings.HasPrefix(part, "table=") {
				table = strings.TrimPrefix(part, "table=")
			} else if strings.HasPrefix(part, "key=") {
				keyField = strings.TrimPrefix(part, "key=")
			} else if strings.HasPrefix(part, "value=") {
				valueField = strings.TrimPrefix(part, "value=")
			}
		}
	}

	if table == "" || keyField == "" || valueField == "" {
		return nil
	}

	return createDBTranslator(table, keyField, valueField)
}
