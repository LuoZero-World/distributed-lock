package utils

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
)

// 获取当前的进程ID
func GetCurrentProcessID() string {
	return strconv.Itoa(os.Getpid())
}

// 获取当前的协程ID
func GetCurrentGoroutineID() string {
	buf := make([]byte, 128)
	buf = buf[:runtime.Stack(buf, false)]
	stackInfo := string(buf)
	return strings.TrimSpace(strings.Split(strings.Split(stackInfo, "[running]")[0], "goroutine")[1])
}

// 获取当前主机Ipv4局域网地址
func getLocalIPv4() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.IsPrivate() {
			if ip := ipnet.IP.To4(); ip != nil {
				return ip.String(), nil
			}
		}
	}
	return "", fmt.Errorf("未找到合适的 IPv4 地址")
}

func GenerateID() string {
	ip, err := getLocalIPv4()
	if err != nil {
		// 如果获取IP地址失败，记录错误并使用默认IP地址
		fmt.Printf("获取本地IPv4地址失败: %v, 使用本地回环地址\n", err)
		ip = "127.0.0.1" // 使用本地回环地址作为默认值
	}
	return fmt.Sprintf("%s_%s_%s", ip, GetCurrentProcessID(), GetCurrentGoroutineID())
}
