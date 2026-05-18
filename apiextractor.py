import argparse


def parse_args():
    """解析命令行参数，后续会在这里增加 URL、代理、输出格式等参数。"""
    parser = argparse.ArgumentParser(
        description="APIExtractor project skeleton."
    )
    parser.add_argument("-u", "--url", help="Target page URL")
    return parser.parse_args()


def main():
    """项目入口函数，后续负责串联 core 目录下的各个功能模块。"""
    args = parse_args()
    print("APIExtractor project skeleton is ready.")
    if args.url:
        print(f"Target URL: {args.url}")
    print("Next step: implement modules under core/ one by one.")


if __name__ == "__main__":
    main()
