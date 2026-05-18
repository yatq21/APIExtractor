def request_api(url, headers=None, timeout=10, proxies=None, verify=True):
    """请求单个 API 接口，并返回状态码、响应长度、耗时等响应元信息。"""
    raise NotImplementedError("TODO: implement single API request")


def request_all(urls, headers=None, timeout=10, proxies=None, verify=True, threads=10):
    """批量请求多个 API 接口，后续可在这里实现并发、超时和代理支持。"""
    raise NotImplementedError("TODO: implement batch API requests")
