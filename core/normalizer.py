def normalize_url(raw_url, base_url, same_origin=True):
    """清洗单个原始接口候选，并将其转换为可直接请求的完整 URL。"""
    raise NotImplementedError("TODO: implement URL normalization")


def normalize_urls(raw_urls, base_url, same_origin=True):
    """批量规范化接口候选，后续负责补全 URL、过滤无效项和去重。"""
    raise NotImplementedError("TODO: implement batch URL normalization")
