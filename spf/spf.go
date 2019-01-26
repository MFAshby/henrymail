package spf

import (
	"fmt"
	"henrymail/config"
)

func GetSpfRecordString() string {
	return fmt.Sprintf("v=spf1 a:%s -all", config.GetString(config.ServerName))
}
