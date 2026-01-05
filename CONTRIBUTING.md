# Contributing to dict-trans

感谢您对 dict-trans 项目的关注！我们欢迎所有形式的贡献。

## 如何贡献

### 报告问题

如果您发现了 bug 或有功能建议，请通过 GitHub Issues 提交。

### 提交代码

1. Fork 本仓库
2. 创建您的特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交您的更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启一个 Pull Request

### 代码规范

- 遵循 Go 官方代码规范
- 所有公共 API 需要添加注释
- 新功能需要添加测试用例
- 确保所有测试通过

### 测试

```bash
go test ./...
```

## 开发指南

### 项目结构

```
dict-trans/
├── *.go              # 核心代码
├── *_test.go         # 测试文件
├── examples/         # 示例代码
└── docs/             # 文档
```

### 添加新功能

1. 在相应的文件中实现功能
2. 添加测试用例
3. 更新文档
4. 添加示例代码

## 许可证

贡献的代码将使用与项目相同的许可证（Mulan PSL v2）。

