package tool

import (
	"fmt"
	"time"
)

// 格式化
const DATE_LAYOUT = "2006-01-02"

// 以秒为基本单位的时间枚举常量
const (
	SECOND = 1
	MINUTE = SECOND * 60
	HOUR   = MINUTE * 60
	DAY    = HOUR * 24
	DAY_8  = DAY * 8
)

/**
* 格式化日期
* timeValue 10位时间戳
 */
func TimeForHuman(timeValue int64) string {
	// 今年则显示07-08即可，非今年，则显示完整的2022-07-08
	nowTime := time.Now().Unix()
	diffTime := nowTime - timeValue

	if diffTime <= MINUTE {
		return "刚刚"
	} else if diffTime < HOUR {
		return fmt.Sprintf("%d分钟前", int(diffTime/MINUTE))
	} else if diffTime <= DAY {
		return fmt.Sprintf("%d小时前", int(diffTime/HOUR))
	} else if diffTime <= DAY_8 {
		return fmt.Sprintf("%d天前", int(diffTime/DAY))
	} else {
		return time.Unix(timeValue, 0).Format(DATE_LAYOUT)
	}
}