def is_http_url(url):
    """判断 URL 是否使用 HTTP 或 HTTPS 协议。"""
    raise NotImplementedError("TODO: implement HTTP URL check")


def is_static_resource(url):
    """判断 URL 是否指向图片、样式、字体等静态资源。"""
    raise NotImplementedError("TODO: implement static resource check")


def is_same_origin(url, base_url):
    """判断两个 URL 是否同源，即协议、域名和端口是否一致。"""
    raise NotImplementedError("TODO: implement same-origin check")
