def print_table(results):
    """将扫描结果以终端表格形式打印，方便快速查看。"""
    raise NotImplementedError("TODO: implement table output")


def export_csv(results, output_path):
    """将扫描结果导出为 CSV 文件，方便用 Excel 或其他工具分析。"""
    raise NotImplementedError("TODO: implement CSV export")


def export_json(results, output_path):
    """将扫描结果导出为 JSON 文件，方便二次开发或接入其他平台。"""
    raise NotImplementedError("TODO: implement JSON export")


def export_results(results, output_format="table", output_path=None):
    """根据用户指定的输出格式，分发到表格、CSV 或 JSON 输出逻辑。"""
    raise NotImplementedError("TODO: implement output dispatcher")
