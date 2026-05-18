# APIExtractor

APIExtractor 是一个面向渗透测试、SRC 漏洞挖掘和接口安全评估场景的 API 提取与检测工具项目。

当前仓库处于项目初始化阶段，只保留了基础目录结构和单文件职责划分，还没有实现真实的 API 提取、请求和检测逻辑。

## 项目目标

在渗透测试和 SRC 挖掘过程中，前端页面和 JavaScript 文件中经常会暴露接口路径。手动从浏览器插件、开发者工具或 JS 文件中整理接口，再复制到 Burp Suite、Postman、curl 或脚本中测试，流程比较繁琐。

本项目后续希望围绕以下方向开发：

- 从目标网页及其 JavaScript 文件中提取 API。
- 对接口地址进行清洗、补全和去重。
- 批量请求接口，记录状态码、响应长度、响应时间等信息。
- 以终端表格或本地文件形式输出结果。
- 辅助发现未授权访问、敏感接口暴露等 API 安全问题。

## 当前状态

当前版本只保留项目框架。

运行入口：

```bash
python apiextractor.py -u https://example.com
```

当前输出：

```text
APIExtractor project skeleton is ready.
Target URL: https://example.com
Next step: implement modules under core/ one by one.
```

## 项目结构

```text
APIExtractor/
├── README.md
├── requirements.txt
├── apiextractor.py
├── config.py
├── core/
│   ├── __init__.py
│   ├── crawler.py
│   ├── extractor.py
│   ├── normalizer.py
│   ├── requester.py
│   ├── analyzer.py
│   └── exporter.py
├── utils/
│   ├── __init__.py
│   ├── logger.py
│   └── url_utils.py
└── output/
    └── .gitkeep
```

## 框架组织思路

项目按一条 API 发现与检测流水线拆分文件：

```text
输入目标 URL
    ↓
获取网页内容
    ↓
提取接口候选
    ↓
规范化接口地址
    ↓
请求接口
    ↓
分析响应
    ↓
输出结果
```

对应的模块关系：

```text
apiextractor.py
    ↓
core/
    ↓
utils/
```

`apiextractor.py` 作为入口负责组织流程；`core/` 放核心业务模块；`utils/` 放通用辅助能力。

## 文件说明

| 文件或目录 | 当前作用 |
| --- | --- |
| `README.md` | 项目说明文档 |
| `requirements.txt` | 预留 Python 第三方依赖列表 |
| `apiextractor.py` | 命令行入口文件，目前只输出项目骨架提示 |
| `config.py` | 预留默认配置，例如超时时间、线程数、请求头等 |
| `core/` | 核心模块目录，后续存放 API 提取、请求、分析、输出等逻辑 |
| `core/__init__.py` | 标记 `core` 为 Python 包 |
| `core/crawler.py` | 预留网页和 JavaScript 获取模块 |
| `core/extractor.py` | 预留 API 候选提取模块 |
| `core/normalizer.py` | 预留接口地址规范化模块 |
| `core/requester.py` | 预留接口请求模块 |
| `core/analyzer.py` | 预留响应分析模块 |
| `core/exporter.py` | 预留结果输出模块 |
| `utils/` | 通用工具目录 |
| `utils/__init__.py` | 标记 `utils` 为 Python 包 |
| `utils/logger.py` | 简单日志输出工具 |
| `utils/url_utils.py` | 预留 URL 相关工具函数 |
| `output/` | 预留结果输出目录 |
| `output/.gitkeep` | 占位文件，用于让 Git 保留空的 `output/` 目录 |

## 依赖

当前 `requirements.txt` 中预留了后续可能使用的依赖：

```txt
requests
beautifulsoup4
rich
```

安装依赖：

```bash
pip install -r requirements.txt
```

## 合法使用声明

本项目仅用于授权范围内的安全测试、企业内部资产检查、SRC 漏洞挖掘和学习研究。请勿在未获得授权的情况下对第三方系统进行扫描、测试或攻击。使用者应自行承担因非法使用造成的全部后果。
