package logger

import "fmt"

// Info 输出普通信息日志。
func Info(message string) {
	fmt.Printf("[+] %s\n", message)
}

// Warning 输出警告日志。
func Warning(message string) {
	fmt.Printf("[!] %s\n", message)
}

// Error 输出错误日志。
func Error(message string) {
	fmt.Printf("[-] %s\n", message)
}
