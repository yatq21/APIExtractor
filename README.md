# APIExtractor

APIExtractor 是一个面向授权安全测试、SRC 漏洞挖掘和接口安全评估场景的前端 API 发现 MVP。当前目标是从前端页面、静态资源和公开发现入口中提取 API 线索，完成规范化、受限验证、风险线索标记和结构化输出。

它不是完整前端反编译器，也不是自动漏洞扫描器。工具输出的风险标签只用于人工复核，不等同于漏洞结论。

## 当前能力

- 抓取目标入口 HTML，并按同源策略发现可分析资源。
- 从 HTML 元素中发现资源或 API-like 入口，包括 `script src`、`script type=module src`、`link rel=modulepreload`、`link rel=manifest`、`iframe src`、`form action`、`a href`、`meta refresh url=`。
- 从 JS、module、chunk、JSON/Text、SourceMap、OpenAPI/Swagger、robots、sitemap、manifest 中提取 API 线索。
- 对 SourceMap 的 `sourcesContent` 做有限结构化解析，把其中的接口线索标记为 `restored-source`；没有 `sourcesContent` 时不会把 `sources` 文件名当 API。
- 使用 fetch、axios、XHR、jQuery、对象属性、字符串等规则做兜底提取；这些是启发式线索，不是 AST 级还原。
- 规范化候选 URL，过滤明显静态资源，保留来源 URL、来源类型、发现规则、method hint、置信度和标签。
- 默认真实请求仅使用 GET 做安全验证。POST/PUT/DELETE/PATCH 等只作为 `MethodGuess` / method hint 保留，不主动构造真实非 GET 请求。
- 记录状态码、耗时、响应大小、Content-Type、响应摘要、错误类型、风险标签、风险证据和可复核 curl。
- 对 Next、Nuxt、Vite、Webpack、Angular、Vue、React 做轻量特征识别，结果进入资源 tags；不承诺完整 runtime、路由或 chunk 还原。
- 支持终端表格输出和 JSON 文件输出。

## 项目结构

```text
APIExtractor/
├── main.go                        # 命令行入口
├── README.md                      # 项目说明
├── docs/requirements.md           # MVP 需求与验收边界
├── internal/
│   ├── config/                    # 默认配置
│   ├── core/                      # 抓取、提取、归一化、验证、分析流水线
│   ├── exporter/                  # table/json 输出
│   ├── logger/                    # 轻量日志
│   ├── model/                     # 结构化数据模型
│   └── urlutil/                   # URL 规范化工具
├── output/                        # 默认输出目录
└── testdata/                      # HTML/JS/JSON/字典/响应测试样例
```

## 执行流程

```text
输入目标 URL
  -> 抓取 HTML
  -> 发现资源入口和 HTML API-like 入口
  -> 抓取 JS/module/chunk/source map/manifest 等资源
  -> 从源码、还原源码线索和发现资源中提取 API 候选
  -> 归一化、过滤、去重、保留来源信息
  -> 使用 GET 做受限验证
  -> 生成风险线索和响应摘要
  -> 输出 table 或 JSON
```

## 使用方式

运行表格输出：

```bash
go run . -u https://example.com
```

输出 JSON：

```bash
go run . -u https://example.com -format json -o output/result.json
```

常用参数：

- `-u`：目标页面 URL，必填。
- `-format`：输出格式，支持 `table` 和 `json`，默认 `table`。
- `-o`：输出文件路径，仅在文件输出时使用。
- `-allow-cross-origin`：允许保留跨域候选接口，默认关闭。
- `-header`：自定义请求头，可重复。
- `-cookie`：显式 Cookie 请求头。
- `-wordlist` / `-dict`：本地字典文件。
- `-no-dir-scan`：关闭字典资源发现。
- `-c` / `-concurrency`：请求并发数。

## 边界说明

- 默认真实请求只做 GET 安全验证；非 GET 方法只作为前端 method hint 输出。
- SourceMap `sourcesContent` 属于有限“还原源码线索”，普通正则匹配属于兜底提取，两者会用 `SourceType` / `DiscoverRule` / tags 区分。
- Next/Nuxt/Vite/Webpack/Angular/Vue/React 只做轻量特征识别，不做完整构建产物还原。
- 风险标签、敏感字段命中和风险证据只表示“值得人工复核”，不能直接作为漏洞结论。
- 当前不包含登录态自动复用、HAR/cURL 导入、Playwright/CDP 动态抓流量、签名参数恢复、完整 AST 分析、自动漏洞利用。

## 合法使用说明

本项目仅用于授权范围内的安全测试、企业内部资产检查、SRC 漏洞挖掘、课程实践和安全研究。禁止用于未授权扫描、认证绕过、数据窃取、破坏目标系统可用性或其他非法用途。
