package tools

import (
	"bytes"
	"encoding/binary"
	"log"
	"time"
)

func IntToHex(num int64) []byte  {
	buff := new(bytes.Buffer)
	err := binary.Write(buff, binary.BigEndian, num)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

//将时间戳转成成当前时间
func UnixTimeToDate(timestamp int64) string {
	timeLayout := "2006-01-02 15:04:05"
	datetime := time.Unix(timestamp, 0).Format(timeLayout)

	return datetime
}
