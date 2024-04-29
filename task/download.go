package task

import (

	"github.com/XIU2/CloudflareSpeedTest/utils"

)

func TestDownloadSpeed(ipSet utils.PingDelaySet) (speedSet utils.DownloadSpeedSet) {

	return utils.DownloadSpeedSet(ipSet)
}

