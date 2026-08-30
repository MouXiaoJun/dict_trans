# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.3.1] - 2026-08-30

### Fixed / Changed

- Apply cache TTL, capacity and enable switches; isolate backend generations and use unambiguous cache keys.
- Keep batch missing/empty results only for the current call to avoid repeated single-key lookups; fix masking edge cases.
- Compatibility: custom-cache keys become cold and are private to a registration. Use positive TTL or external eviction to reclaim old namespaces.
- Align LICENSE and current documentation with MIT, as confirmed by the maintainer; preserve existing copyright notices.

## [1.3.0] - 2026-02-XX

### Added
- 数据脱敏：struct tag 驱动（`mask:"phone"` / `mask:"3,4"` / `mask:"-"` / `mask:"*"`）
- 内置脱敏格式：phone / idcard / bankcard / email / name / address / password / 通配
- 通用保留格式（前 n 后 m 字符，按 rune 处理，中文安全）
- 嵌套结构体 / 结构体指针 / 结构体切片 / 顶层切片自动递归脱敏
- 泛型入口 MaskOf；自定义格式 RegisterMaskFormat（按管理器隔离，注册后即时生效）
- 脱敏与翻译共存：先 Translate 后 Mask，配置缓存独立

## [Unreleased]

### Added
- 字典翻译功能（内存字典）
- 字典表翻译功能（单表结构）
- 双表字典翻译功能（字典类型表+字典数据表）
- 枚举转换功能
- 数据库翻译功能（类似 Easy Trans）
- 嵌套翻译功能
- 自定义翻译器
- 包装类型支持
- 批量并行翻译
- 高性能翻译框架（中间件、插件、策略、监控）
- 配置缓存机制
- 结果缓存机制
- 批量查询优化
- 预加载机制
- 性能监控

### Changed
- 优化了反射性能
- 改进了缓存策略

## [1.0.0] - 2024-XX-XX

### Added
- 初始版本发布
- 基础字典翻译功能
- 数据库翻译功能
- 框架模式支持
