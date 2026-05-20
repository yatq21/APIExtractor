# APIExtractor v0.1.2 需求确认稿

## 0. 文档状态

本文档为 APIExtractor 的 v0.1.2 需求确认稿，用于在正式编码重构前统一项目方向、需求边界、宏观框架、MVP 范围、模块职责、数据流转关系、版本演进路线和验收标准。

需要特别区分：

1. **需求确认稿版本为 v0.1.2**：表示在原始 v0.1 计数体系内，对需求确认稿进行第二次重要修订，重点补全宏观框架、远景能力池和字典更新策略；
2. **产品实现版本对应 v0.1.2**：v0.1.2 仍属于 MVP 闭环阶段，不等同于 v0.2.0 增强版本；
3. **当前不更改项目名称**：项目继续使用 `APIExtractor`；
4. **当前不进行大规模代码重构**：先在现有 Go 项目骨架上扩展，后续再按模块逐步迁移；
5. **当前核心目标不变**：优先形成“API 发现、提取、规范化、验证、风险初筛、结构化输出”的最小闭环；
6. **保留宏观拓展能力**：远景能力池保留，但不改变 v0.1.2 的验收范围。

当前项目名称继续使用：

```text
APIExtractor
```

后续版本允许在稳定版发布前评估是否更换正式项目名称，但名称调整不得影响当前需求重构、分支协作、模块设计和历史文档引用。

---

## 1. 项目定位

APIExtractor 是一个面向授权渗透测试、SRC 漏洞挖掘、接口安全评估和 Web 前端暴露面分析场景的 API 自动发现与初步验证工具。

本项目当前阶段不定位为完整漏洞扫描器，也不承诺自动判定漏洞成立。第一阶段重点是构建一个稳定、可测试、可维护的 API 资产发现与风险初筛流程，减少安全测试人员在前端接口收集、路径整理、批量请求和结果筛选中的重复工作。

项目重点解决以下问题：

1. 目标站点中有哪些页面、目录、静态资源和接口入口值得分析；
2. 前端 HTML、JavaScript、动态 chunk、SourceMap、JSON/Text 响应中暴露了哪些 API 线索；
3. 提取出的 API 如何规范化、去重、分类和批量验证；
4. 哪些接口响应具有进一步人工分析价值；
5. 如何通过结构化输出帮助后续复现、分工、测试和迭代；
6. 如何在授权红队演练、SRC 挖掘和实验室训练场景中辅助前期侦察、API 暴露面梳理、认证态接口线索整理和风险证据留存；
7. 如何在不把项目膨胀成完整漏洞扫描器的前提下，为后续前端还原、动态流量采集、风险分析增强和报告能力预留扩展空间。

---

## 1.1 网络安全场景适配说明

APIExtractor 面向网络安全实验室、授权红队演练、SRC 漏洞挖掘和企业内部安全评估场景。其核心定位不是“自动化攻击工具”，而是**授权范围内的 API 暴露面发现、接口资产整理、初步验证和证据辅助工具**。

在偏红方的实验室工作流中，APIExtractor 应主要服务于以下环节：

1. **前期侦察**：从目标首页、目录扫描结果、前端资源、SourceMap、JSON/Text 响应中发现 API 入口；
2. **暴露面梳理**：将零散路径、前端路由、JS 中的接口字符串和目录扫描结果归并为结构化 API 候选；
3. **认证态线索整理**：为 Header、Cookie、Token、代理、HAR/curl 导入等后续能力预留接口，但 v0.1.2 不默认绕过鉴权或构造攻击请求；
4. **初步验证**：对规范化 API 执行受限 GET 验证，记录状态码、响应长度、Content-Type、响应摘要和错误原因；
5. **风险初筛**：通过敏感字段、调试信息、接口语义、SourceMap 暴露、Swagger/OpenAPI 暴露等规则生成风险标签；
6. **证据留存**：保留来源资源、触发规则、脱敏样本、响应摘要和可复核字段，辅助人工判断和后续报告；
7. **复盘与教学**：通过稳定 JSON、测试样例和分工结构支持实验室成员复盘、训练和继续开发。

v0.1.2 需要明确保留红方工作流适配能力，但不改变合法使用边界：

```text
允许：授权范围内的信息收集、API 资产梳理、接口初步验证、敏感信息初筛、风险证据辅助留存。
不做：未授权扫描、认证绕过、爆破、攻击载荷生成、自动漏洞利用、自动漏洞定性。
```

因此，Header/Cookie、代理、HAR/curl 导入、Burp 数据导入、可复现 curl 导出、动态浏览器流量采集等能力应作为红方工作流优先增强方向保留，但不全部进入 v0.1.2 必须实现范围。

---

## 2. 需求重构背景

原有项目已经具备基础 Go 框架，当前能力主要包括：输入目标 URL、抓取 HTML、提取 JavaScript、从 HTML/JS 中正则提取 API 候选、归一化 URL、请求候选接口、分析响应并输出结果。

但原需求存在以下问题：

1. **需求上限偏低**  
   原有描述更接近“网页和 JS 正则提取器”，没有完整覆盖目录扫描、资源发现、前端构建产物递归分析、SourceMap、接口响应递归分析和敏感信息判断。

2. **模块边界不清晰**  
   目录扫描、资源发现、API 提取、API 规范化、请求验证、敏感信息识别、风险标签和输出导出之间没有明确的数据边界。

3. **MVP 口径不统一**  
   字典扫描、SourceMap、并发请求、风险标签等能力在不同章节中存在“必须实现”和“尽量实现”混用的问题，容易导致开发分歧。

4. **数据模型缺失**  
   资源发现结果、API 候选、请求验证结果、风险分析结果没有建立清晰的引用关系，不利于结果追踪和后续扩展。

5. **验收标准不够可测**  
   原有验收标准偏功能清单，缺少测试样例、预期输出和默认运行边界。

6. **远景能力承接不足**  
   原需求中提到的目录扫描、前端分析、还原、反编译、敏感信息提取和危害判断等高目标，需要被收纳到远景能力池中，避免一方面把 MVP 写得过大，另一方面又显得项目上限不足。

本次重构将项目需求统一为三大核心模块和五层宏观能力架构：

```text
三大核心模块：
目录扫描与资源发现
        ↓
前端 API 深度提取
        ↓
API 规范化、验证与敏感信息判断

五层宏观能力架构：
入口发现层
资源分析层
API 资产层
验证分析层
输出协作层
```

---

## 3. 项目名称与版本演进策略

### 3.1 当前项目名称

当前阶段继续使用：

```text
APIExtractor
```

不在需求重构期更换名称，避免影响现有仓库、文档、分支协作和组内沟通。

### 3.2 名称更改预留

后续稳定版发布前可评估正式名称。建议版本策略如下：

```text
需求文档版本：
v0.1.0：需求确认初稿
v0.1.1：数据模型与 MVP 边界补强稿
v0.1.2：宏观框架、远景能力池与字典更新策略补全稿
v0.1.x：开发任务拆解与验收补充稿

产品实现版本：
v0.1.2：MVP 闭环确认版，允许保留发行前字典更新链路
v0.2.0：目录扫描与资源发现增强
v0.3.0：前端分析与 SourceMap 增强
v0.4.0：动态采集与请求重放增强
v1.0.0：稳定演示版，评估是否更换正式名称
```

### 3.3 工程预留要求

为降低后续改名成本，要求：

1. 工具显示名称由统一常量或配置维护；
2. CLI banner、报告标题、README 标题可统一替换；
3. 输出结果中保留 `tool_name` 和 `tool_version`；
4. 核心包名和模块名优先使用功能语义；
5. 后续改名不影响数据结构和核心接口。

输出元信息示例：

```json
{
  "tool_name":"APIExtractor",
  "tool_version":"0.1.2",
  "schema_version":"0.1.2",
  "scan_target":"https://example.com",
  "scan_time":"2026-xx-xx xx:xx:xx"
}
```

---

## 4. 宏观框架总览

APIExtractor 的宏观框架划分为五层。

```text
输入与配置
    ↓
入口发现层
    ↓
资源分析层
    ↓
API 资产层
    ↓
验证分析层
    ↓
输出协作层
```

### 4.1 输入与配置

负责接收用户输入和扫描参数。

包括：

1. 目标 URL；
2. 本地字典；
3. 内置字典开关；
4. 同源策略；
5. 并发数；
6. 超时时间；
7. 最大递归深度；
8. 最大资源数量；
9. 最大响应体大小；
10. 输出格式。

### 4.2 入口发现层

负责扩大目标入口覆盖面。

包括：

1. 入口 HTML 抓取；
2. 内置字典扫描；
3. 本地字典扫描；
4. 目录路径探测；
5. API 文档入口发现；
6. 后台路径发现；
7. 静态资源入口发现。

### 4.3 资源分析层

负责分析各种响应资源。

包括：

1. HTML 分析；
2. JavaScript 分析；
3. chunk 发现；
4. SourceMap 发现和基础提取；
5. JSON/Text 响应分析；
6. 资源队列递归控制；
7. 前端框架轻量特征识别。

### 4.4 API 资产层

负责把零散线索转换为结构化 API 候选。

包括：

1. API 候选提取；
2. API 来源记录；
3. API 规范化；
4. API 过滤；
5. API 去重；
6. 同源判断；
7. 置信度标记；
8. API 候选队列管理。

### 4.5 验证分析层

负责对 API 候选进行请求验证和风险初筛。

包括：

1. GET 请求验证；
2. 状态码记录；
3. 响应长度记录；
4. 响应时间记录；
5. Content-Type 记录；
6. 响应摘要截断；
7. 敏感信息识别；
8. 风险标签生成；
9. 置信度调整。

### 4.6 输出协作层

负责结果展示、文件导出和后续协作。

包括：

1. table 输出；
2. JSON 输出；
3. 统计汇总；
4. 来源追踪；
5. 后续 CSV、curl、HTML 报告、Postman Collection、OpenAPI 草稿等扩展。

---

## 5. 术语定义

为避免沟通和开发时混淆，本文档统一使用以下术语。

| 术语 | 定义 |
|---|---|
| 目标 URL | 用户输入的起始 URL，例如 `https://example.com` |
| Origin | 由协议、主机名和端口组成的站点根，例如 `https://example.com` |
| 资源 | HTML、JS、CSS、SourceMap、JSON、Text 等可请求对象 |
| 有效资源 | 响应状态、类型或内容具有继续分析价值的资源，不要求必须是 200 |
| API 候选 | 从资源中提取出的疑似接口路径或接口 URL，尚未完成验证 |
| 规范化 API | 经过补全、过滤、去重和同源判断后的 API |
| API 验证结果 | 对规范化 API 发起请求后得到的响应摘要 |
| 风险标签 | 根据路径语义、状态码、响应内容和敏感字段生成的提示标签，不等同于漏洞结论 |
| 敏感信息识别 | 对响应内容中可能存在的手机号、邮箱、token、调试信息等进行规则匹配 |
| 递归分析 | 从一个资源中发现新资源或 API，再将其加入队列继续分析的过程 |
| 字典 | 用于目录扫描和常见路径探测的候选路径集合 |
| 资源队列 | 用于继续抓取和分析的资源集合 |
| API 候选队列 | 用于规范化和验证的疑似 API 集合 |
| 风险证据 | 触发风险标签的路径、状态码、响应类型、关键词或脱敏样本 |

---

## 6. 当前阶段非目标

当前阶段明确不做以下内容：

1. 不更改项目名称；
2. 不进行大规模无依据代码重构；
3. 不承诺完整反编译 Vue、React、Angular 等前端框架；
4. 不将 AI 作为核心工作流依赖；
5. 不自动判断漏洞一定成立；
6. 不做认证绕过、爆破或攻击载荷生成；
7. 不进行未授权目标的大规模扫描；
8. 不将 Pwn、RE、Patch 自动化等方向混入当前 Web 项目核心需求；
9. 不在需求未稳定前设计复杂插件系统；
10. 不在用户扫描目标时自动从 GitHub 或其他平台在线拉取大规模字典；
11. 不把 GitHub 字典更新作为 v0.1.2 普通用户侧默认在线功能；
12. 不在 v0.1.2 中强行接入浏览器自动化、Burp 联动或登录态对比。

当前阶段只追求一个核心目标：完成 API 发现、提取、规范化、验证、敏感信息初筛和结构化输出的最小闭环。

---

## 7. MVP 总体业务流程

MVP 主流程如下：

```text
用户输入目标 URL
        ↓
加载基础配置
        ↓
抓取入口 HTML
        ↓
执行最小目录扫描，发现入口资源
        ↓
提取 HTML 中的 JS、链接和疑似 API
        ↓
下载并分析 JS / chunk / SourceMap
        ↓
从 HTML / JS / SourceMap / JSON / Text 中提取 API 候选
        ↓
API 规范化、过滤、去重、同源判断
        ↓
使用 GET 批量验证规范化 API
        ↓
记录状态码、响应长度、响应时间、Content-Type、响应摘要
        ↓
基于关键词、正则和路径语义生成风险标签
        ↓
输出 table / JSON
```

增强流程包括但不限于：

```text
自定义 Header / Cookie
HTTP/HTTPS 代理
HAR / curl 导入
Burp / 浏览器流量衔接
CSV 输出
可复现 curl 导出
登录前后响应对比
多身份响应对比
AST 分析
框架路由还原
远程字典更新
动态浏览器流量采集
HTML 报告
Postman Collection / OpenAPI 草稿导出
```

其中 Header/Cookie、代理、HAR/curl、Burp 数据导入和可复现请求导出属于红方工作流高价值能力，应在 v0.2.0 之后优先评估。

---

## 8. 数据模型分层

为避免资源、API、验证结果和风险结果混淆，项目内部应至少区分五层数据。

### 8.1 ScanMeta：扫描元信息

```text
scan_id
tool_name
tool_version
target_url
target_origin
start_time
end_time
config_summary
```

### 8.2 ResourceRecord：资源发现结果

资源发现结果用于描述被请求和分析过的 HTML、JS、SourceMap、JSON/Text 等对象。

字段建议：

```text
resource_id
url
method
status_code
content_length
content_type
title
redirect_location
resource_type
discover_source
parent_resource_id
same_origin
fetch_error
should_analyze
risk_hint
```

资源类型示例：

```text
html
javascript
chunk_js
source_map
json
text
api_doc
admin_page
static_resource
unknown
```

### 8.3 APICandidate：API 候选结果

API 候选表示从资源中提取出的疑似 API，尚未完成最终验证。

字段建议：

```text
candidate_id
raw_value
normalized_url
method_guess
source_resource_id
source_url
source_type
discover_rule
same_origin
confidence
tags
```

发现规则示例：

```text
html_link
script_src
form_action
fetch_call
axios_call
xhr_call
absolute_url
relative_api_path
graphql_keyword
source_map_text
json_text
wordlist_path
```

### 8.4 APIResult：API 验证结果

API 验证结果表示对规范化 API 发起请求后的响应摘要。

字段建议：

```text
result_id
candidate_id
api_url
method
status_code
content_length
response_time
content_type
redirect_location
response_sample
error_reason
risk_tags
sensitive_matches
confidence
```

### 8.5 RiskEvidence：风险证据

风险证据用于记录风险标签的触发依据。v0.1.2 可以先合并在 APIResult 中，后续版本可独立成结构。

字段建议：

```text
evidence_id
result_id
risk_tag
evidence_type
evidence_source
masked_sample
confidence
```

### 8.6 ReportSummary：汇总结果

```text
total_resources
total_candidates
total_verified_apis
risk_tag_count
sensitive_match_count
error_count
```

---

## 9. 核心模块一：目录扫描与资源发现

### 9.1 模块目标

目录扫描与资源发现模块用于解决“入口不足”的问题。工具不能只依赖用户输入的单个首页，而应通过内置字典、本地字典和资源探测发现更多可能包含接口线索的入口。

目录扫描的结果不直接等同于 API 结果。目录扫描输出的是资源发现结果，后续由资源类型决定是否进入 HTML、JS、SourceMap、JSON/Text 或 API 候选提取流程。

### 9.2 输入

MVP 输入：

```text
target_url
builtin_wordlist
local_wordlist
same_origin
max_concurrency
max_requests
timeout
```

增强输入：

```text
custom_headers
cookie
proxy
remote_wordlist
wordlist_profile
```

### 9.3 字典来源

MVP 支持两类字典：

1. **内置默认字典**：项目自带的小型常见路径字典，用于无参数情况下的基础扫描；
2. **本地字典文件**：用户通过命令行参数指定的自定义字典文件。

后续增强支持：

1. GitHub 等公开字典源整理；
2. 按场景拆分 API、后台、Swagger、框架资源字典；
3. 字典去重、分类、更新和质量评估。

### 9.4 字典文件格式

本地字典文件采用纯文本格式，默认 UTF-8 编码。

规则：

1. 一行表示一个候选路径；
2. 空行自动忽略；
3. 以 `#` 开头的行视为注释；
4. 路径可以带 `/`，也可以不带 `/`；
5. MVP 阶段不支持通配符、变量模板和复杂表达式；
6. MVP 阶段默认不允许完整 URL，完整 URL 交由后续增强处理；
7. 路径统一按相对目标 origin 处理。

示例：

```text
# common paths
api
/api
admin
/admin/login
swagger-ui.html
v2/api-docs
actuator
static/js
```

### 9.5 字典清洗规则

加载字典后需要执行：

1. 去除首尾空白；
2. 忽略空行和注释行；
3. 将反斜杠统一为 `/`；
4. 统一路径前导 `/`；
5. 去除重复路径；
6. 过滤明显非法路径；
7. 保留原始字典项和清洗后路径，便于追踪。

### 9.6 URL 拼接规则

目录扫描默认以目标 origin 为基准拼接路径，而不是以入口页面所在目录为基准。

示例：

```text
目标 URL：https://example.com/app/index.html
目标 origin：https://example.com
字典项：admin
生成 URL：https://example.com/admin
```

```text
目标 URL：https://example.com/app/index.html
目标 origin：https://example.com
字典项：/api/user
生成 URL：https://example.com/api/user
```

### 9.7 扫描任务字段

每个字典扫描任务至少包含：

```text
task_id
target_origin
raw_word
normalized_path
target_url
dictionary_source
scan_type
```

`dictionary_source` 示例：

```text
builtin_common
user_file
remote_future
```

### 9.8 有效资源判断

以下响应应视为具有继续分析价值：

1. `200` 且 Content-Type 为 HTML、JS、JSON、Text、SourceMap；
2. `301/302/307/308` 且 Location 指向同源路径；
3. `401/403`，作为受保护资源记录，不默认递归；
4. `500`，作为异常资源记录；
5. 响应内容或路径命中 API 文档、Swagger、OpenAPI、Knife4j、SourceMap 等特征。

以下响应默认不进入递归分析：

1. 明显静态图片、字体、视频；
2. 空响应；
3. 软 404 或错误模板页面；
4. 超过最大响应体大小的响应；
5. 跨域资源，除非显式允许。

### 9.9 结果回流规则

字典扫描发现的资源按类型进入不同流程：

1. HTML 响应进入 HTML 提取流程；
2. JavaScript 响应进入 JS 提取流程；
3. SourceMap 响应进入 SourceMap 提取流程；
4. JSON/Text 响应进入文本 API 提取流程；
5. 疑似 API 路径进入 API 候选队列；
6. 无效响应仅记录基础信息，不进入后续递归流程。

### 9.10 MVP 范围

MVP 必须实现：

1. 内置小型默认字典；
2. 本地字典文件加载；
3. 字典清洗和去重；
4. 基于 origin 的 URL 拼接；
5. 并发请求扫描；
6. 记录字典来源；
7. 将有效资源回流到后续提取模块；
8. 为后续 GitHub 字典源整理预留数据结构和目录位置。

MVP 暂不实现：

1. 普通用户扫描时自动从 GitHub 在线爬取大字典；
2. 扫描运行时远程字典自动更新；
3. 通配符和变量模板；
4. 字典质量评分；
5. 分布式目录扫描。

### 9.11 GitHub 字典源更新策略

GitHub 字典源整理属于原始需求中的重要方向，但在 v0.1.2 中应采用“发行前更新为主、用户本地运行为辅、扫描时不自动联网”的策略。

#### 9.11.1 推荐策略

v0.1.2 推荐采用以下模式：

```text
维护者 / 开发者在发行前运行字典同步脚本
        ↓
从 GitHub 等公开来源收集常见目录字典
        ↓
清洗、去重、分类、压缩
        ↓
生成项目内置小型默认字典或 wordlists 文件
        ↓
随版本发布
        ↓
普通用户默认使用随版本发布的内置字典或本地 -w 字典
```

#### 9.11.2 产品用户侧行为

v0.1.2 普通用户在本地运行产品时，默认不自动访问 GitHub 更新字典。

原因：

1. 保证扫描结果可复现；
2. 避免运行时依赖外网和 GitHub 可用性；
3. 避免 GitHub 访问失败、限速、代理问题影响主流程；
4. 避免把字典源可信度和供应链风险直接转嫁给用户；
5. 避免扫描工具在目标测试期间产生额外无关网络流量。

#### 9.11.3 v0.1.2 可保留能力

v0.1.2 可以保留以下能力，但不作为普通用户默认扫描流程：

1. 预留 `wordlists/` 目录或内置字典生成入口；
2. 预留 `tools/wordlist-sync` 或 `scripts/update_wordlists` 维护者脚本位置；
3. 预留字典元信息字段，例如来源、更新时间、哈希、分类、条目数量；
4. 支持用户通过 `-w / --wordlist` 指定本地自定义字典；
5. 在远景能力池中保留 GitHub 字典源整理、字典分类和质量评分。

#### 9.11.4 后续版本演进

```text
v0.1.2：发行前更新字典；用户运行时默认不联网更新
v0.2.0：支持字典分类、来源记录、字典质量评分调研
v0.3.0+：可考虑显式命令 apiextractor wordlist update
v1.x：可考虑远程字典源插件化，但必须显式启用
```

#### 9.11.5 结论

v0.1.2 可以实现或保留 GitHub 字典爬取更新能力，但应定位为开发者维护链路或发行前构建链路，而不是普通用户扫描目标时自动运行的在线功能。

---

## 10. 核心模块二：前端 API 深度提取

### 10.1 模块目标

前端 API 深度提取模块用于解决“API 收集不完整”的问题。该模块负责从 HTML、JavaScript、动态 chunk、SourceMap、JSON/Text 响应和前端构建产物中递归发现 API 端点。

### 10.2 HTML 提取需求

从 HTML 中提取：

1. `script src`；
2. `link href`；
3. `iframe src`；
4. `form action`；
5. `a href`；
6. `meta refresh`；
7. 内联 JavaScript；
8. 页面文本中的疑似 API 路径。

提取结果需要按类型分流：

1. JS、CSS、SourceMap 等进入资源队列；
2. 表单 action、疑似 `/api/` 路径进入 API 候选队列；
3. 普通页面链接作为资源候选处理，不直接视为 API。

### 10.3 JavaScript 提取需求

从 JS 中提取：

1. 完整 URL；
2. 相对 API 路径；
3. `fetch` 调用；
4. `axios` 调用；
5. `XMLHttpRequest` 调用；
6. WebSocket 地址；
7. GraphQL endpoint；
8. `baseURL` 配置；
9. 动态 `import`；
10. chunk 文件路径；
11. `sourceMappingURL`；
12. 字符串拼接形成的疑似接口路径。

MVP 阶段以正则和字符串模式为主，不要求完整 AST 分析。

### 10.4 SourceMap 需求

MVP 阶段：

1. 从 JS 中识别 `sourceMappingURL`；
2. 尝试下载 `.map` 文件；
3. 判断 SourceMap 是否暴露；
4. 将 `.map` 文本作为输入复用 API 提取规则；
5. 在资源和 API 候选中记录来源为 `source_map`。

后续增强：

1. 解析 `sources`；
2. 解析 `sourcesContent`；
3. 还原源码文件路径；
4. 从源码模块中提取路由和接口调用关系。

### 10.5 递归分析队列

递归分析应区分资源队列和 API 候选队列。

资源队列用于继续抓取和分析：

```text
HTML
JS
chunk_js
SourceMap
JSON/Text
```

API 候选队列用于后续规范化和验证：

```text
/api/user
https://example.com/api/user
/graphql
wss://example.com/socket
```

默认递归策略：

```text
默认最大递归深度：2
默认最大资源数量：200
默认最大响应体大小：2MB
默认只递归同源资源
默认资源去重 key：method + normalized_resource_url
默认 API 去重 key：method_guess + normalized_api_url
```

递归规则：

1. HTML 可继续提取 JS、链接、表单和疑似 API；
2. JS 可继续提取 chunk、SourceMap 和 API；
3. SourceMap 可提取 API 和源码路径，但 MVP 不继续深度还原源码结构；
4. JSON/Text 可提取 URL/API，但默认不无限递归；
5. 跨域资源默认只记录，不递归，除非显式允许。

### 10.6 前端框架识别需求

第一阶段只做轻量特征识别，不做完整反编译。

识别对象：

```text
Vue
React
Angular
Next.js
Nuxt
Webpack
Vite
jQuery
uni-app
```

识别依据：

1. 静态资源路径；
2. JS 文件名；
3. 全局变量；
4. HTML 特征；
5. SourceMap sources 路径；
6. chunk 命名规律。

识别结果在 MVP 阶段主要作为标签输出和后续规则扩展依据，不作为必需的核心能力。后续可以根据框架特征补充规则：

```text
Next.js -> 关注 /_next/static/
Nuxt -> 关注 /_nuxt/
Vite -> 关注 /assets/*.js
Webpack -> 关注 runtime chunk 和 sourceMappingURL
```

### 10.7 输出

输出 API 候选记录：

```text
candidate_id
raw_value
normalized_url
method_guess
source_resource_id
source_url
source_type
discover_rule
same_origin
confidence
tags
```

---

## 11. 核心模块三：API 规范化、验证与敏感信息判断

### 11.1 模块目标

该模块用于解决“哪些接口值得进一步分析”的问题。它不直接给出漏洞结论，而是根据请求响应特征生成风险标签，辅助人工判断。

### 11.2 API 规范化规则

需要处理以下形式：

```text
/api/user
//example.com/api/user
https://example.com/api/user
../api/user
${baseURL}/user/info
/api/{id}/detail
```

规范化规则：

1. 相对路径优先基于发现该 API 的资源 URL 解析；
2. 目录扫描字典路径基于目标 origin 拼接；
3. 协议相对 URL 使用目标 URL 的协议补全；
4. 默认去除 fragment；
5. query 默认保留，但可在去重时按需归一；
6. 默认端口按协议归一化；
7. URL 解码后保留必要转义；
8. 明显静态资源后缀过滤；
9. `{id}`、`:id` 等占位符保留并标记为参数化路径；
10. `${baseURL}` 等变量形式在无法解析时保留 raw_value，并降低置信度。

去重 key：

```text
method_guess + normalized_url
```

### 11.3 请求验证策略

MVP 阶段使用 GET 请求验证，记录：

```text
status_code
content_length
response_time
content_type
redirect_location
error_reason
response_sample
```

默认请求边界：

```text
默认只请求同源 API
默认超时：10s
默认最大响应摘要：2KB
默认最大响应体扫描长度：2MB
默认跟随同源重定向
默认不保存完整响应正文
默认不重试高风险写操作方法
```

MVP 阶段不主动构造 POST/PUT/DELETE 请求，不自动生成参数，不尝试绕过鉴权。

在授权红队或 SRC 场景中，认证态请求具有实际价值，但 v0.1.2 默认不将认证态能力作为硬要求。后续应优先支持：

1. 用户显式传入 Header；
2. 用户显式传入 Cookie；
3. HTTP/HTTPS 代理；
4. 从 HAR/curl/Burp 数据中恢复真实请求；
5. 导出可复现请求用于人工复核。

所有认证态能力均应以用户显式输入为前提，不自动获取、猜测、绕过或伪造认证信息。

后续增强支持：

1. HEAD；
2. POST；
3. OPTIONS；
4. 自定义 Header；
5. 自定义 Cookie；
6. 代理；
7. 忽略 TLS 证书校验；
8. 登录前后对比；
9. 导出可复现 curl。

### 11.4 敏感信息识别

MVP 阶段采用关键词和正则规则识别敏感信息类型。

识别类型：

```text
phone
email
id_card
jwt
token
access_token
refresh_token
session
cookie
password
secret
ak/sk
username
real_name
address
user_id
order_id
role
permission
admin
debug
exception
stacktrace
internal_ip
```

处理要求：

1. 默认不保存完整敏感原文；
2. 对命中样本进行脱敏或截断；
3. 记录命中类型、命中次数和脱敏样本；
4. 区分字段名命中和值命中；
5. 响应体扫描长度受最大响应体限制；
6. 对明显误报类型降低置信度。

### 11.5 风险标签与触发规则

风险标签只代表辅助判断，不等同于漏洞结论。

| 标签 | 触发条件 |
|---|---|
| `sensitive_data_possible` | 状态码为 200，响应为 JSON/Text，且命中敏感字段规则 |
| `unauth_access_possible` | 未提供认证信息时访问成功，且响应包含结构化业务数据或敏感字段 |
| `admin_api_exposed` | API 路径含 admin/manage/system 等管理语义，且响应非明显 404 |
| `source_map_exposed` | SourceMap 文件可访问 |
| `swagger_exposed` | 路径或响应内容命中 Swagger/OpenAPI/Knife4j 特征 |
| `debug_info_exposed` | 响应内容命中 exception、stacktrace、debug、traceback 等调试特征 |
| `large_json_response` | 响应为 JSON 且响应长度超过默认阈值 |
| `auth_required` | 状态码为 401，或响应内容明确提示认证需求 |
| `forbidden` | 状态码为 403 |
| `redirect_to_login` | 3xx 跳转目标包含 login/auth/sso 等登录语义 |
| `static_resource` | 目标被识别为静态资源 |
| `low_confidence` | 变量路径、拼接路径、不可解析路径或误报风险较高 |

默认阈值建议：

```text
large_json_response：content_length >= 100KB
response_sample：最多保存 2KB
sensitive_sample：仅保存脱敏样本
```

### 11.6 输出

API 验证结果字段：

```text
result_id
candidate_id
api_url
method
status_code
content_length
response_time
content_type
redirect_location
risk_tags
sensitive_matches
source_resource_id
confidence
error_reason
```

---

## 12. 输出格式需求

### 12.0 JSON Schema 版本

v0.1.2 的 JSON 输出需要在 `meta` 中保留结构版本号，避免后续 v0.2.0、v0.3.0 扩展字段时无法区分输出结构。

建议最小字段：

```json
{
  "meta":{
    "tool_name":"APIExtractor",
    "tool_version":"0.1.2",
    "schema_version":"0.1.2",
    "scan_id":"uuid-or-hash",
    "scan_time":"2026-xx-xx xx:xx:xx"
  }
}
```

要求：

1. `tool_version` 表示工具产品版本；
2. `schema_version` 表示输出 JSON 结构版本；
3. v0.1.2 阶段 `tool_version` 和 `schema_version` 可以相同；
4. 后续输出结构变化时，应优先调整 `schema_version`；
5. JSON 输出结构必须保持向后兼容或显式标记不兼容。

### 12.1 Table 输出

MVP 阶段 table 输出用于终端快速查看，建议展示：

```text
method
api_url
status_code
content_length
response_time
content_type
risk_tags
source_type
```

### 12.2 JSON 输出

MVP 阶段 JSON 输出应采用稳定结构，建议：

```json
{
  "meta": {},
  "target": {},
  "resources": [],
  "api_candidates": [],
  "api_results": [],
  "summary": {}
}
```

字段说明：

1. `meta`：工具名称、版本、schema 版本、扫描时间；
2. `target`：目标 URL、origin、扫描配置摘要；
3. `resources`：资源发现结果；
4. `api_candidates`：API 候选；
5. `api_results`：API 验证结果；
6. `summary`：数量统计和风险标签统计；
7. `wordlists`：可选字段，用于记录内置字典或本地字典的元信息；
8. `errors`：可选字段，用于汇总扫描过程中的错误类型和数量。

后续输出扩展：

1. CSV；
2. HTML 报告；
3. 可复现 curl；
4. Postman Collection；
5. OpenAPI 草稿；
6. 风险证据链报告。

---

## 13. CLI 与配置需求

### 13.0 配置优先级

v0.1.2 以 CLI 参数和默认配置为主，配置文件作为后续增强能力预留。

建议配置优先级：

```text
CLI 参数 > 配置文件 > 默认配置
```

说明：

1. v0.1.2 必须支持 CLI 参数和默认配置；
2. `--config` 可作为后续增强参数预留；
3. 如果后续加入配置文件，CLI 参数应覆盖配置文件中的同名字段；
4. 默认配置应集中维护，避免散落在多个模块中。

### 13.1 MVP CLI 参数

MVP 阶段建议支持：

```text
-u, --url                 目标 URL，必填
-w, --wordlist            本地字典文件，可选
--no-builtin-wordlist     禁用内置字典，可选
--depth                   最大递归深度，默认 2
--max-resources           最大资源数量，默认 200
--max-body-size           最大响应体大小，默认 2MB
--concurrency             最大并发数，默认 10
--timeout                 请求超时，默认 10s
--same-origin             仅处理同源资源，默认开启
--allow-cross-origin      允许跨域资源进入候选，默认关闭
--format                  输出格式，table/json，默认 table
-o, --output              输出文件路径，json 输出时使用
```

### 13.2 后续增强参数

```text
--header                  自定义请求头
--cookie                  自定义 Cookie
--proxy                   HTTP/HTTPS 代理
--insecure                忽略 TLS 证书错误
--export-curl             导出可复现 curl
--config                  配置文件路径
--import-har              导入 HAR 文件
--import-curl             导入 curl 命令
--browser                 启用浏览器动态采集
--dry-run                 后续版本可选：扫描计划预览，不发起真实请求
--update-wordlist         后续版本可选：显式更新远程字典，v0.1.2 不作为默认功能
```

### 13.3 网络安全工作流适配优先级

针对授权红队、SRC 和实验室训练场景，后续参数实现优先级建议如下：

| 优先级 | 参数 / 能力 | 价值 | 建议阶段 |
|---|---|---|---|
| P1 | `--header` | 支持 Bearer Token、自定义认证头、特殊 UA | v0.1.3 / v0.2.0 |
| P1 | `--cookie` | 支持登录态接口发现和验证 | v0.1.3 / v0.2.0 |
| P1 | `--proxy` | 适配 Burp、Yakit、Caido 等代理工作流 | v0.2.0 |
| P2 | `--export-curl` | 辅助人工复核和报告复现 | v0.2.0+ |
| P2 | `--import-har` | 导入浏览器真实流量 | v0.4.0+ |
| P2 | `--import-curl` | 复用单个真实请求 | v0.4.0+ |
| P3 | `--browser` | 动态采集运行时接口 | v1.x |

约束：

1. v0.1.2 可以保留参数设计和数据结构预留，但不强制实现；
2. Header/Cookie 必须由用户显式提供，不自动窃取或推断；
3. 代理只用于授权测试流量转发和人工复核，不用于绕过访问控制；
4. 导出的 curl 应避免默认包含完整敏感 Token，除非用户显式要求并确认保存范围。

---

## 14. v0.1.2 工程化补强项

本节用于补充 v0.1.2 的稳定性、可复现性、工程化和测试可落地能力。以下内容不改变 MVP 的功能边界，但应作为需求确认稿的一部分保留。

### 14.1 扫描预算与运行边界

v0.1.2 应集中定义扫描预算，防止递归分析失控，避免工具行为变成不可控爬虫。

| 项目 | 默认值 | 说明 |
|---|---:|---|
| 最大递归深度 | 2 | 控制 HTML、JS、chunk、SourceMap、JSON/Text 的递归层级 |
| 最大资源数量 | 200 | 控制资源队列规模 |
| 最大响应体大小 | 2MB | 超过后截断或跳过深度分析 |
| 默认并发数 | 10 | 限制请求压力 |
| 请求超时 | 10s | 保证扫描可以结束 |
| 默认同源策略 | 开启 | 默认只递归同源资源 |
| 完整响应保存 | 关闭 | 默认只保存摘要，降低敏感信息风险 |
| 响应摘要长度 | 2KB | 仅用于人工判断和风险证据 |

要求：

1. 所有默认值应集中维护；
2. 超出预算的资源应记录原因，不应静默丢弃；
3. 预算命中不应导致程序异常退出；
4. 输出 summary 中可记录预算命中次数。

### 14.2 错误处理与失败原因分类

v0.1.2 应统一资源抓取和 API 验证中的错误类型，避免 `error_reason` 字段随意填写。

建议错误类型：

```text
dns_error
connect_timeout
read_timeout
tls_error
redirect_too_many
body_too_large
unsupported_content_type
decode_error
invalid_url
request_blocked
wordlist_error
output_write_error
unknown_error
```

要求：

1. `ResourceRecord.fetch_error` 和 `APIResult.error_reason` 应尽量使用统一错误类型；
2. 错误信息可以保留简短摘要，但不应输出过长堆栈；
3. JSON summary 中可统计各类错误数量；
4. 错误不应中断整个扫描流程，除非属于参数错误或输出文件不可写等致命错误。

### 14.3 字典 Manifest 元信息

v0.1.2 保留 GitHub 字典源发行前更新策略，因此需要为内置字典和本地字典预留 manifest 元信息。

建议字段：

```text
wordlist_name
wordlist_version
source_type
source_url
updated_at
entry_count
sha256
category
maintainer
```

`source_type` 示例：

```text
builtin
user_file
github_pre_release
manual
```

要求：

1. 内置字典应记录版本、条目数和更新时间；
2. 本地字典至少记录文件名、条目数和哈希；
3. 发行前从 GitHub 整理的字典应记录来源和更新时间；
4. 普通用户扫描时默认不联网更新字典；
5. 输出 JSON 可在 `wordlists` 字段中记录字典元信息。

### 14.4 风险证据最小字段

v0.1.2 不需要实现完整风险证据链，但应在 APIResult 中预留最小风险证据字段，说明风险标签的触发原因。红方场景下，风险证据的目标是辅助人工复核和报告编写，而不是自动给出漏洞结论。

建议最小结构：

```json
{
  "risk_evidence":[
    {
      "tag":"sensitive_data_possible",
      "reason":"matched sensitive keyword",
      "sample":"tok***",
      "confidence":0.7
    }
  ]
}
```

要求：

1. `sample` 必须脱敏或截断；
2. `reason` 应简短描述触发规则；
3. 同一标签可以有多个证据，但 v0.1.2 不要求完整证据链；
4. 没有证据时不应强行生成风险标签；
5. 证据应尽量包含来源资源、触发规则、状态码、Content-Type、响应长度等可复核信息；
6. 不默认保存完整响应正文和完整认证凭据。

### 14.5 轻量错误页与软 404 记录

完整软 404 判断属于 v0.2.0 增强能力，v0.1.2 不强制实现复杂算法。但可以记录疑似错误页特征，辅助后续过滤误报。

可选记录项：

```text
same_length_as_error_page
title_contains_404
body_contains_not_found
redirect_to_error_page
```

要求：

1. v0.1.2 不把软 404 判断作为硬性验收；
2. 如果实现，只作为 `risk_hint` 或 `tags`；
3. 不因疑似软 404 直接删除记录，避免漏报。

### 14.6 日志级别与退出码

v0.1.2 应保留基本日志级别和退出码设计，便于调试、CI 和组内协作。

建议日志级别：

```text
silent
info
warn
debug
```

建议退出码：

```text
0  扫描完成
1  参数错误
2  目标无法访问
3  字典文件错误
4  输出文件写入失败
5  内部错误
```

要求：

1. 默认日志级别为 `info`；
2. 调试信息只在 `debug` 下输出；
3. 参数错误和输出文件错误属于致命错误；
4. 单个资源请求失败不应导致整体退出。

### 14.7 测试样例目录结构

v0.1.2 应明确测试样例目录结构，方便模块负责人分别补测试。

建议结构：

```text
testdata/
├── html/
│   └── basic.html
├── js/
│   ├── fetch_axios.js
│   └── chunk_source_map.js
├── json/
│   └── api_response.json
├── wordlists/
│   └── common.txt
└── responses/
    └── sensitive.json
```

要求：

1. HTML 样例用于测试 script、link、form、a 等提取；
2. JS 样例用于测试 fetch、axios、XHR、chunk、sourceMappingURL；
3. JSON/Text 样例用于测试响应中的 URL/API 提取；
4. wordlists 样例用于测试字典加载、清洗、去重和拼接；
5. sensitive 响应用于测试敏感信息识别和脱敏。

### 14.8 扫描计划预览预留

`--dry-run` 不属于 v0.1.2 必须实现，但可以作为宏观拓展能力保留。

预期能力：

```text
不发起真实请求，仅展示：
1. 目标 origin；
2. 字典条目数量；
3. 预计请求数量；
4. 同源策略；
5. 最大递归深度；
6. 输出路径；
7. 是否使用内置字典和本地字典。
```

建议阶段：v0.2.0 或后续版本。

---

## 15. v0.1.2 MVP 最小实现范围

### 15.1 v0.1.2 必须实现

```text
1. 输入目标 URL；
2. 抓取入口 HTML；
3. 支持内置小型默认字典；
4. 支持本地字典文件；
5. 字典清洗、去重和基于 origin 的路径拼接；
6. 执行最小目录扫描并发现有效资源；
7. 提取 HTML 中的 JS、链接和疑似 API；
8. 下载并分析 JS 文件；
9. 递归发现 JS/chunk；
10. 发现 sourceMappingURL，尝试下载 SourceMap；
11. 从 HTML/JS/SourceMap/JSON/Text 中提取 API 候选；
12. API 规范化、补全、过滤、去重；
13. 使用 GET 批量请求验证；
14. 记录状态码、响应长度、响应时间、Content-Type；
15. 基于关键词和正则识别敏感信息；
16. 生成风险标签；
17. 输出 table 和 JSON；
18. 输出中保留 tool_name 和 tool_version；
19. 保留 GitHub 字典源整理的工程扩展点，但不在用户扫描时自动联网更新；
20. 输出 JSON 中保留 `schema_version`；
21. 统一错误类型和失败原因分类；
22. 保留字典 manifest 元信息；
23. APIResult 中预留最小风险证据字段，作为可选输出结构；
24. 明确测试样例目录结构；
25. 测试后保持仓库工作区干净；
26. 保留网络安全工作流适配说明，明确红方场景下的允许能力、禁止能力和后续增强方向。
```

### 15.2 v0.1.2 不做

```text
1. 不更改项目名称；
2. 不大规模移动目录结构；
3. 不完整反编译前端框架；
4. 不做 AST 深度分析；
5. 不依赖 AI 判断结果；
6. 不自动判断漏洞成立；
7. 不自动绕过认证；
8. 不生成攻击载荷；
9. 不在用户扫描目标时自动从 GitHub 或其他远程源更新字典；
10. 不做 Burp Suite 深度联动；
11. 不做登录前后响应对比；
12. 不做浏览器动态流量采集；
13. 不自动获取、窃取、推断或伪造认证信息；
14. 不默认导出完整敏感 Token、Cookie 或响应正文。
```

---

## 16. 远景能力池

本节内容不属于 v0.1.2 必须实现范围，仅用于记录后续可能演进方向。是否进入具体版本，需要根据 MVP 完成情况、团队人力、测试效果和实际安全价值再确认。

| 方向 | 能力 | 价值 | 依赖 | 建议阶段 | 当前状态 |
|---|---|---|---|---|---|
| 入口扩展 | GitHub 字典源整理 | 扩大目录扫描覆盖面 | 本地字典扫描稳定 | v0.1.2 预留 / v0.2.0+ 增强 | 发行前更新链路 |
| 入口扩展 | 字典分类和质量评分 | 降低无效请求和误报 | 字典加载稳定 | v0.2.0+ | 增强 |
| 入口扩展 | Swagger/OpenAPI/Knife4j 深度解析 | 直接获取接口结构 | API 文档识别 | v0.2.0+ | 增强 |
| 入口扩展 | robots.txt / sitemap.xml / manifest.json 入口发现 | 扩大入口覆盖面 | 资源发现稳定 | v0.2.0+ | 调研 |
| 前端分析 | SourceMap sourcesContent 解析 | 还原源码级接口线索 | SourceMap 基础下载 | v0.3.0+ | 增强 |
| 前端分析 | Webpack runtime chunk 分析 | 提升 chunk 发现能力 | JS/chunk 递归稳定 | v0.3.0+ | 调研 |
| 前端分析 | Vite / Next.js / Nuxt 静态资源规则识别 | 提升框架适配能力 | 框架轻量识别 | v0.3.0+ | 增强 |
| 前端分析 | Vue Router / React Router / Next.js route 提取 | 发现前端路由入口 | 框架识别稳定 | v0.3.0+ | 调研 |
| 前端分析 | AST 级接口调用提取 | 降低正则误报漏报 | JS 提取规则稳定 | v0.3.0+ | 调研 |
| 红方工作流 | Header / Cookie 显式输入 | 支持认证态接口发现和验证 | 请求模型稳定 | v0.1.3 / v0.2.0 | 优先增强 |
| 红方工作流 | HTTP/HTTPS 代理 | 接入 Burp、Yakit、Caido 等工具链 | 请求模块稳定 | v0.2.0+ | 优先增强 |
| 红方工作流 | 可复现 curl 导出 | 支持人工复核和报告复现 | APIResult 稳定 | v0.2.0+ | 增强 |
| 动态采集 | HAR 文件导入 | 复用真实浏览器请求 | 请求模型稳定 | v0.4.0+ | 调研 |
| 动态采集 | curl 导入 | 快速复现单个请求 | 请求模型稳定 | v0.4.0+ | 调研 |
| 动态采集 | Burp 数据导入 | 接入渗透测试工作流 | 请求模型稳定 | v0.4.0+ | 调研 |
| 动态采集 | Playwright + CDP 动态抓取 | 发现运行时接口 | 浏览器自动化 | v1.x | 远期 |
| 验证增强 | POST / PUT / DELETE / OPTIONS 方法探测 | 更接近真实接口行为 | 请求验证稳定 | v0.4.0+ | 增强 |
| 验证增强 | 从前端代码或 HAR 中恢复真实请求方法 | 提高验证准确率 | 动态采集或 AST | v0.4.0+ | 调研 |
| 验证增强 | 登录前后响应对比 | 辅助判断未授权 | 登录态支持 | v1.x | 远期 |
| 验证增强 | 多身份响应对比 | 辅助判断越权 | 多身份配置 | v1.x | 远期 |
| 报告增强 | HTML 报告 | 提升展示和复盘效率 | JSON schema 稳定 | v1.x | 远期 |
| 报告增强 | 风险证据链报告 | 方便人工复核 | RiskEvidence 稳定 | v1.x | 远期 |
| 报告增强 | Postman Collection / OpenAPI 草稿导出 | 对接测试和开发流程 | APIResult 稳定 | v1.x | 远期 |
| 工程化 | 规则插件化 | 支持团队扩展规则 | 核心接口稳定 | v1.x | 远期 |
| 工程化 | 多目标批量任务 | 扩展项目适用范围 | 单目标稳定 | v1.x | 远期 |
| 工程化 | 结果数据库存储 | 支持历史对比 | JSON 输出稳定 | v1.x | 远期 |
| 工程化 | Web UI 或本地报告页面 | 提升演示效果 | 报告能力稳定 | v1.x | 远期 |

远景能力池的约束：

1. 不改变 v0.1.2 MVP 范围；
2. 不作为当前阶段验收标准；
3. 不写入当前开发任务，除非组内明确调整版本目标；
4. 不包含自动绕过认证、自动攻击、自动漏洞利用等能力；
5. 所有增强能力仍应服务于授权安全测试和人工复核；
6. 红方工作流增强不得变成自动绕过、自动攻击或自动漏洞利用能力。

---

## 17. 产品版本规划

### 17.1 v0.1.2：MVP 闭环确认版

目标：完成 API 发现、提取、规范化、验证、风险初筛和 JSON/table 输出的最小闭环，并保留宏观拓展能力。

交付物：

1. 可运行 CLI；
2. 内置小型字典；
3. 本地字典文件支持；
4. HTML/JS/chunk/SourceMap 基础提取；
5. API 规范化与 GET 验证；
6. 风险标签和敏感信息初筛；
7. JSON/table 输出；
8. 基础测试样例；
9. 远景能力池；
10. GitHub 字典源发行前更新策略和工程预留；
11. JSON schema_version；
12. 字典 manifest；
13. 统一错误类型；
14. 最小风险证据字段。

### 17.2 v0.2.0：目录扫描、资源发现与红方工作流适配增强

目标：扩大入口覆盖面、提升目录扫描效果，并优先补齐授权安全测试中常用的 Header/Cookie/代理链路。

增强内容：

1. 字典分类；
2. 常见 API/后台/Swagger/框架资源字典；
3. API 文档识别；
4. robots.txt / sitemap.xml / manifest.json 入口发现；
5. 软 404 判断；
6. 资源类型标签增强；
7. 扫描统计增强；
8. Header / Cookie 显式输入；
9. HTTP/HTTPS 代理支持；
10. 可复现 curl 导出调研。

### 17.3 v0.3.0：前端分析增强

目标：提升前端构建产物分析能力。

增强内容：

1. 框架特征识别增强；
2. SourceMap 深度解析；
3. 前端路由提取；
4. 更完善的 JS 提取规则；
5. Webpack / Vite / Next.js / Nuxt 资源规则；
6. AST 分析调研落地。

### 17.4 v0.4.0：动态采集与请求重放增强

目标：补充静态分析无法覆盖的运行时接口。

增强内容：

1. HAR 导入；
2. curl 导入；
3. Burp 数据导入调研；
4. 请求方法恢复；
5. 请求头和 Cookie 复用；
6. 可复现 curl 导出。

### 17.5 v1.0.0：稳定演示版

目标：形成稳定、可演示、可复现、可维护版本。

包含：

1. 稳定 CLI；
2. 稳定 JSON schema；
3. 完整 README；
4. 测试样例；
5. 版本化发布；
6. HTML 或 Markdown 报告；
7. 评估是否更换正式项目名称。

---

## 18. 分工建议

建议按模块分工，而不是按文件分工。

| 方向 | 主要任务 | 产出 |
|---|---|---|
| 目录扫描与资源发现 | 字典加载、路径生成、并发探测、资源筛选 | ResourceRecord 列表 |
| 前端 API 深度提取 | HTML/JS/chunk/SourceMap/JSON 提取规则 | APICandidate 列表 |
| API 验证与风险判断 | 规范化、请求验证、敏感信息识别、风险标签 | APIResult 列表 |
| 输出与报告 | table/json 输出、后续报告扩展 | 结构化报告 |
| 文档与测试 | 需求文档、测试样例、使用说明、验收记录 | 可维护资料 |

每个模块负责人应先写清楚：

```text
输入是什么
处理流程是什么
输出是什么
MVP 做到什么程度
测试样例是什么
验收标准是什么
```

---

## 19. 现有项目结构与目标模块映射

当前不要求立即重排目录。建议先在现有结构上扩展，后续通过单独重构分支迁移。

| 目标模块 | 当前可能对应位置 | 处理建议 |
|---|---|---|
| collector | `internal/core/crawler.go` | 先扩展资源收集能力，后续再拆分 |
| extractor | `internal/core/extractor.go` | 扩展 HTML/JS/SourceMap/JSON 提取规则 |
| normalizer | `internal/core/normalizer.go`、`internal/urlutil` | 保持现有结构，补充规范化规则 |
| requester | `internal/core/requester.go` | 增加请求边界和响应摘要 |
| analyzer | `internal/core/analyzer.go` | 增加风险标签和敏感信息识别 |
| exporter | `internal/exporter` | 稳定 table/json 输出 |
| model | `internal/model` | 增加 ResourceRecord、APICandidate、APIResult |
| config | `internal/config` | 增加扫描参数和默认边界 |
| docs/test | `docs/`、测试文件 | 承接需求文档和测试样例 |

重构原则：

1. v0.1.2 不进行大规模目录迁移；
2. 优先补齐模型和功能；
3. 每次结构调整必须对应明确需求；
4. 重构通过单独 `refactor/xxx` 分支完成；
5. 修改核心逻辑必须补测试。

---

## 20. 第一阶段验收标准

第一阶段完成时，应满足以下标准。

### 20.1 功能验收

1. 输入目标 URL 后，工具能够抓取入口 HTML；
2. 能够加载内置字典和本地字典；
3. 能够基于 origin 生成目录扫描 URL；
4. 能够筛选有效资源；
5. 能够提取 HTML 中引用的 JS；
6. 能够递归发现 JS/chunk；
7. 能够发现并尝试下载 SourceMap；
8. 能够从 HTML/JS/SourceMap/JSON/Text 中提取 API 候选；
9. 能够对 API 做补全、过滤、去重和同源判断；
10. 能够使用 GET 批量请求 API；
11. 能够记录状态码、响应长度、响应时间和 Content-Type；
12. 能够基于正则和关键词标记疑似敏感信息；
13. 能够生成风险标签；
14. 能够输出 table 和 JSON；
15. 能够在 JSON `meta` 中输出 `schema_version`；
16. 能够在资源抓取和 API 验证失败时输出统一错误类型；
17. 能够记录字典来源或 manifest 元信息；
18. APIResult 能够预留 `risk_evidence` 字段，用于说明风险标签触发原因；
19. 文档能够明确网络安全场景下的红方工作流适配边界，包括认证态输入、代理、证据留存和禁止自动化利用的约束。

### 20.2 测试样例

至少准备以下样例：

1. HTML 引用外部 JS 的样例；
2. JS 中包含 fetch/axios/API 字符串的样例；
3. JS 中包含 chunk 或 SourceMap 线索的样例；
4. JSON/Text 中包含 URL/API 的样例；
5. 响应内容包含 token/email/debug 等敏感字段的样例；
6. 字典中包含 `/api`、`/admin`、`swagger-ui.html` 等路径的样例；
7. 字典 manifest 样例；
8. 错误处理样例；
9. JSON schema_version 输出样例。

### 20.3 质量验收

1. `go test ./...` 通过；
2. 工作区保持 clean；
3. 核心模块有基本注释；
4. JSON 输出结构稳定；
5. 不提交临时文件、敏感信息和本地配置；
6. PR 描述包含修改内容、测试结果和已知风险；
7. 错误类型字段稳定；
8. 敏感样本默认脱敏或截断；
9. 字典来源可追踪。

---

## 21. 待确认事项

以下事项需要组内进一步确认，但不阻塞当前需求文档成稿：

1. v0.1.2 默认并发数是否采用 10；
2. v0.1.2 默认最大资源数量是否采用 200；
3. v0.1.2 默认最大响应体大小是否采用 2MB；
4. v0.1.2 默认最大递归深度是否采用 2；
5. large_json_response 阈值是否采用 100KB；
6. 是否在 v0.1.2 中支持 `--header` 和 `--cookie`；
7. SourceMap 在 v0.1.2 中是否只做文本提取，不解析 sourcesContent；
8. 字典文件是否放入仓库 `wordlists/` 目录，还是暂时以内置 Go 切片形式维护；
9. GitHub 字典源更新是否采用“维护者发行前更新，用户扫描时默认不联网”的策略；
10. 是否将本需求文档作为 `docs/requirements.md` 合入仓库；
11. 远景能力池是否作为正式附录保留；
12. 是否接受 v0.4.0 作为动态采集与请求重放增强阶段；
13. v0.1.2 是否将日志级别和退出码作为必须实现；
14. v0.1.2 是否将字典 manifest 默认写入 JSON 输出；
15. v0.1.2 是否在 table 输出中展示 `risk_evidence` 摘要；
16. Header / Cookie 是否作为 v0.1.3 或 v0.2.0 优先增强项；
17. HTTP/HTTPS 代理是否作为 v0.2.0 优先增强项；
18. 可复现 curl 导出是否进入 v0.2.0 增强范围。

---

## 22. 合法使用声明

本项目仅用于授权范围内的安全测试、企业内部资产检查、SRC 漏洞挖掘、课程实践和安全研究。

禁止用于：

1. 未授权扫描第三方系统；
2. 批量探测无关目标；
3. 绕过认证或访问控制；
4. 窃取、导出或传播敏感数据；
5. 破坏目标系统可用性。

工具输出的风险标签仅代表辅助判断，不等同于漏洞结论。最终是否构成漏洞，需要人工结合授权范围、业务逻辑、访问控制机制和复现结果进行确认。

---

## 23. 文档定位总结

本文档的定位是：

```text
APIExtractor v0.1.2 需求确认稿
```

它用于确认：

1. 项目当前不改名；
2. v0.1.2 先完成 MVP 闭环；
3. 当前不大规模重构目录；
4. 三大核心模块和五层宏观框架成立；
5. 远景能力池正式保留，作为后续演进参考，不进入当前验收；
6. GitHub 字典源更新能力保留为发行前维护链路和后续增强点，用户扫描时默认不自动联网；
7. 网络安全场景适配以授权红队前期侦察、API 暴露面梳理、认证态线索整理和证据辅助留存为主，不包含自动绕过、自动攻击或自动利用；
8. 后续开发应从需求确认转入任务拆解和 Git 分支协作。
```

