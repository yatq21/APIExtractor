# APIExtractor

APIExtractor 是一个面向渗透测试、SRC 漏洞挖掘和接口安全评估场景的 Go 语言 API 提取与检测工具骨架。当前版本已经完成从 Python 骨架到 Go 框架的迁移，保留了抓取、提取、归一化、请求、分析和导出这些核心职责边界，便于后续继续扩展真实能力。

## 当前能力

- 从目标页面抓取 HTML 内容。
- 从 HTML 中提取外链 JavaScript 地址，并继续抓取 JS 文件。
- 从 HTML 和 JS 文本里正则提取疑似 API 候选。
- 归一化候选 URL，过滤静态资源并按同源策略去重。
- 对候选接口发起直接请求，记录状态码、耗时、响应大小和内容类型。
- 基于响应结果生成初步分析标签。
- 支持终端表格输出和 JSON 文件输出。

当前实现仍然是最小可用框架，不包含登录态复用、HAR/cURL 导入、Playwright/CDP 动态抓流量、签名参数恢复、并发调度和高级风险判断。

## 项目结构

```text
APIExtractor/
├── go.mod                         # Go 模块定义，声明模块名和 Go 版本
├── main.go                        # 命令行入口，解析参数并调度整条扫描流程
├── README.md                      # 项目说明文档，描述结构、用法和后续规划
├── internal/                      # 内部业务代码目录，不对外暴露
│   ├── config/                    # 默认配置定义
│   │   └── config.go              # 超时、默认请求头、输出格式等配置项
│   ├── core/                      # 核心流水线模块
│   │   ├── analyzer.go            # 基于响应结果生成初步风险判断
│   │   ├── crawler.go             # 抓取 HTML 页面并收集外链 JS
│   │   ├── exporter.go            # 把请求结果转换为可输出的行结构
│   │   ├── extractor.go           # 从 HTML/JS 文本里正则提取疑似 API 候选
│   │   ├── normalizer.go          # 归一化候选 URL，并负责去重前的清洗入口
│   │   ├── pipeline.go            # 串联抓取、提取、请求、分析的主流程
│   │   └── requester.go           # 对候选接口发起请求并记录响应摘要
│   ├── exporter/                  # 输出层实现
│   │   └── exporter.go            # 负责表格输出和 JSON 文件导出
│   ├── logger/                    # 轻量日志工具
│   │   └── logger.go              # 提供信息、警告、错误日志输出函数
│   ├── model/                     # 统一数据结构定义
│   │   └── model.go               # 扫描结果、请求结果、分析结果等结构体
│   └── urlutil/                   # URL 相关工具
│       └── urlutil.go             # 处理 URL 补全、同源判断和静态资源过滤
└── output/                        # 默认输出目录
    └── .gitkeep                   # 占位文件，保证空目录可被 Git 跟踪
```

## 执行流程

```text
输入目标 URL
  -> 抓取 HTML
  -> 提取 JS 地址
  -> 抓取 JS 内容
  -> 提取 API 候选
  -> 归一化与去重
  -> 请求候选接口
  -> 分析响应
  -> 输出结果
```

`main.go` 负责命令行入口和总流程调度。`internal/core` 保留主要业务流水线。`internal/model` 放统一数据结构。`internal/exporter` 专门负责输出逻辑。`internal/urlutil` 处理 URL 归一化和过滤规则。

## 使用方式

运行表格输出：

```bash
go run . -u https://example.com
```

输出 JSON：

```bash
go run . -u https://example.com -format json -o output/result.json
```

命令行参数：

- `-u`：目标页面 URL，必填。
- `-format`：输出格式，支持 `table` 和 `json`，默认 `table`。
- `-o`：输出文件路径，仅在文件输出时使用。
- `-allow-cross-origin`：允许保留跨域候选接口，默认关闭。

## 设计取向

这个版本的重点不是“尽量多堆规则”，而是先把后续可演进的 Go 框架搭稳：

- 抓取层：后续可接入自定义 Header、代理、Cookie、登录态和浏览器自动化来源。
- 提取层：后续可补充更稳的 JS/HTML 模式、source map、GraphQL 和 fetch/XHR 特征识别。
- 请求层：后续可扩展并发、重试、速率限制、HAR/cURL 重放。
- 分析层：后续可接入未授权、敏感字段、错误信息泄露和越权测试规则。
- 输出层：后续可增加 CSV、OpenAPI 草稿、Postman Collection。

## 后续建议

如果要让这个项目真正进入可用状态，建议按这个顺序继续做：

1. 增加 `HAR/cURL -> 统一请求结构 -> 自动重放`。
2. 接入 `Playwright + CDP` 做真实浏览器流量采集。
3. 增强静态提取规则，把 JS 扫描降为“候选发现器”，而不是唯一数据源。
4. 增加并发请求、结果持久化和更可靠的风险分析。

## 合法使用说明

本项目仅用于授权范围内的安全测试、企业内部资产检查、SRC 漏洞挖掘和学习研究。请勿在未获得授权的情况下对第三方系统进行扫描、测试或攻击。使用者应自行承担因非法使用造成的全部后果。
