def extract_from_text(text):
    """从单段 HTML 或 JavaScript 文本中提取疑似 API 接口字符串。"""
    raise NotImplementedError("TODO: implement API extraction rules")


def extract_all(html, js_files):
    """整合 HTML 和多个 JavaScript 文件内容，统一提取所有疑似 API 候选。"""
    raise NotImplementedError("TODO: implement combined API extraction")
