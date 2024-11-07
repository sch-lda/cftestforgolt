package utils

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	maxDelay            = 9999 * time.Millisecond
	minDelay            = 0 * time.Millisecond
	maxLossRate float32 = 1.0
)

var (
	InputMaxDelay    = maxDelay
	InputMinDelay    = minDelay
	InputMaxLossRate = maxLossRate
	PrintNum         = 10
	istestok         = false
)

type PingData struct {
	IP       *net.IPAddr
	Sended   int
	Received int
	Delay    time.Duration
}

type CloudflareIPData struct {
	*PingData
	lossRate      float32
	DownloadSpeed float64
}

// 计算丢包率
func (cf *CloudflareIPData) getLossRate() float32 {
	if cf.lossRate == 0 {
		pingLost := cf.Sended - cf.Received
		cf.lossRate = float32(pingLost) / float32(cf.Sended)
	}
	return cf.lossRate
}

func (cf *CloudflareIPData) toString() []string {
	result := make([]string, 6)
	result[0] = cf.IP.String()
	result[1] = strconv.Itoa(cf.Sended)
	result[2] = strconv.Itoa(cf.Received)
	result[3] = strconv.FormatFloat(float64(cf.getLossRate()), 'f', 2, 32)
	result[4] = strconv.FormatFloat(cf.Delay.Seconds()*1000, 'f', 2, 32)
	result[5] = strconv.FormatFloat(cf.DownloadSpeed/1024/1024, 'f', 2, 32)
	return result
}

func convertToString(data []CloudflareIPData) [][]string {
	result := make([][]string, 0)
	for _, v := range data {
		result = append(result, v.toString())
	}
	return result
}

// 延迟丢包排序
type PingDelaySet []CloudflareIPData

// 延迟条件过滤
func (s PingDelaySet) FilterDelay() (data PingDelaySet) {
	if InputMaxDelay > maxDelay || InputMinDelay < minDelay { // 当输入的延迟条件不在默认范围内时，不进行过滤
		return s
	}
	if InputMaxDelay == maxDelay && InputMinDelay == minDelay { // 当输入的延迟条件为默认值时，不进行过滤
		return s
	}
	for _, v := range s {
		if v.Delay > InputMaxDelay { // 平均延迟上限，延迟大于条件最大值时，后面的数据都不满足条件，直接跳出循环
			break
		}
		if v.Delay < InputMinDelay { // 平均延迟下限，延迟小于条件最小值时，不满足条件，跳过
			continue
		}
		data = append(data, v) // 延迟满足条件时，添加到新数组中
	}
	return
}

// 丢包条件过滤
func (s PingDelaySet) FilterLossRate() (data PingDelaySet) {
	if InputMaxLossRate >= maxLossRate { // 当输入的丢包条件为默认值时，不进行过滤
		return s
	}
	for _, v := range s {
		if v.getLossRate() > InputMaxLossRate { // 丢包几率上限
			break
		}
		data = append(data, v) // 丢包率满足条件时，添加到新数组中
	}
	return
}

func (s PingDelaySet) Len() int {
	return len(s)
}
func (s PingDelaySet) Less(i, j int) bool {
	iRate, jRate := s[i].getLossRate(), s[j].getLossRate()
	if iRate != jRate {
		return iRate < jRate
	}
	return s[i].Delay < s[j].Delay
}
func (s PingDelaySet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// 下载速度排序
type DownloadSpeedSet []CloudflareIPData

func (s DownloadSpeedSet) Len() int {
	return len(s)
}
func (s DownloadSpeedSet) Less(i, j int) bool {
	return s[i].DownloadSpeed > s[j].DownloadSpeed
}
func (s DownloadSpeedSet) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s DownloadSpeedSet) Print() {

	if len(s) <= 0 { // IP数组长度(IP数量) 大于 0 时继续
		fmt.Println("\n[无法优化当前网络] 您当前的网络环境完全无法访问Cloudflare,请更换网络或离线使用小助手。")
		return
	}
	dateString := convertToString(s) // 转为多维数组 [][]String
	if len(dateString) < PrintNum {  // 如果IP数组长度(IP数量) 小于  打印次数，则次数改为IP数量
		PrintNum = len(dateString)
	}
	for i := 0; i < 10; i++ {
		fmt.Println("\n最优ip")
		fmt.Println(dateString[i][0])
		systemDrive := os.Getenv("SystemDrive")
		if systemDrive == "" {
			systemDrive = "C:\\"
		} else if !strings.HasSuffix(systemDrive, "\\") {
			systemDrive += "\\"
		}
		hostsPath := filepath.Join(systemDrive, "Windows", "System32", "drivers", "etc", "hosts")
		ipString := dateString[i][0]

		err := updateHostsFile(hostsPath, ipString)
		if err != nil {
			fmt.Println("无法修改Hosts:", err)
			return
		}

		fmt.Println("已成功更新Hosts.即将验证是否真的能够连接小助手服务器...")
		url := "https://sstaticstp.cc2077.site/testcn.txt"

		body, err := downloadJSON(url)
		if err != nil {
			fmt.Println("验证失败...将继续测试下一个IP地址...")
			fmt.Println(err)
			continue
			fmt.Println(body)
		}

		fmt.Println("验证成功!")
		istestok = true
		break

	}
	if istestok {
		fmt.Println("您的电脑现在可以访问小助手服务器了")
		fmt.Printf("按下 回车键 或 Ctrl+C 退出。若重启小助手仍无法联网请重启电脑以刷新DNS缓存")
		return
	}
	fmt.Println("10个候选ip全部连接失败.")
	fmt.Printf("按下 回车键 或 Ctrl+C 退出.此程序无法拯救您的网络...")

}

func updateHostsFile(filePath, ipString string) error {
	file, err := os.OpenFile(filePath, os.O_RDWR, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	tempFile, err := os.CreateTemp("", "hosts_temp")
	if err != nil {
		return err
	}
	defer tempFile.Close()

	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return err
		}

		if !strings.Contains(line, "crazyzhang.cn") && !strings.Contains(line, "cc2077.site") {
			_, err := tempFile.WriteString(line)
			if err != nil {
				return err
			}
		}

		if err == io.EOF {
			break
		}
	}

	newLine := fmt.Sprintf("%s\tapi.crazyzhang.cn\n", ipString)
	newLine2 := fmt.Sprintf("%s\tcrazyzhang.cn\n", ipString)
	newLine3 := fmt.Sprintf("%s\tblog.cc2077.site\n", ipString)
	newLine4 := fmt.Sprintf("%s\tsstaticstp.cc2077.site\n", ipString)
	_, err = tempFile.WriteString(newLine)
	_, err = tempFile.WriteString(newLine2)
	_, err = tempFile.WriteString(newLine3)
	_, err = tempFile.WriteString(newLine4)
	if err != nil {
		return err
	}

	err = file.Close()
	if err != nil {
		return err
	}

	err = copyFile(tempFile.Name(), filePath)
	if err != nil {
		return err
	}

	return nil
}
func copyFile(source, destination string) error {
	sourceFile, err := os.Open(source)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(destination)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}
func downloadJSON(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("无法下载 JSON 文件: %s", err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("无法读取响应内容: %s", err.Error())
	}

	return body, nil
}
