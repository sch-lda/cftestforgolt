package main

import (
	"flag"
	"fmt"
	"time"

	"github.com/XIU2/CloudflareSpeedTest/task"
	"github.com/XIU2/CloudflareSpeedTest/utils"
)

var (
	version, versionNew string
)

func init() {
	var help = ``
	var minDelay, maxDelay, downloadTime int
	var maxLossRate float64
	flag.IntVar(&task.Routines, "n", 400, "Tnum")
	flag.IntVar(&task.PingTimes, "t", 1, "Tcount")
	flag.IntVar(&downloadTime, "dt", 10, "UK2")
	flag.IntVar(&task.TCPPort, "tp", 443, "PT")

	flag.IntVar(&maxDelay, "tl", 500, "MaxDL")
	flag.IntVar(&minDelay, "tll", 0, "MinDL")
	flag.Float64Var(&maxLossRate, "tlr", 1, "LOSTMAX")

	flag.IntVar(&utils.PrintNum, "p", 1, "RLTC")
	flag.StringVar(&task.IPFile, "f", "ip.txt", "U3")
	flag.StringVar(&task.IPText, "ip", "", "U4")

	flag.Usage = func() { fmt.Print(help) }
	flag.Parse()

	utils.InputMaxDelay = time.Duration(maxDelay) * time.Millisecond
	utils.InputMinDelay = time.Duration(minDelay) * time.Millisecond
	utils.InputMaxLossRate = float32(maxLossRate)
}

func main() {
	task.InitRandSeed() // 置随机数种子
	fmt.Printf("#https://github.com/sch-lda/cftestforgolt \n")
	fmt.Printf("Powered by https://github.com/XIU2/CloudflareSpeedTest \n")
	// 开始延迟测速 + 过滤延迟/丢包
	pingData := task.NewPing().Run().FilterDelay().FilterLossRate()
	// 开始下载测速
	speedData := task.TestDownloadSpeed(pingData)
	speedData.Print()          // 打印结果

	fmt.Scanln()
}


