package log

import (
	"encoding/json"
	"fmt"
	"time"
)

type JsonFormatter struct {
	TimeDisplay bool
}

// Format impl the LogFormatter interface method
func (j *JsonFormatter) Format(param *LogFormatParam) string {
	if param.LoggerFields == nil {
		param.LoggerFields = make(Fields)
	}
	now := time.Now()
	if j.TimeDisplay {
		param.LoggerFields["log_time"] = now.Format("2006/01/02 - 15:04:05")
	}
	param.LoggerFields["msg"] = param.Msg
	/// Level is int, Level.Level() to get the string
	param.LoggerFields["log_level"] = param.Level.Level()
	marshal, err := json.Marshal(param.LoggerFields)
	if err != nil {
		panic(err)
	}
	return fmt.Sprintf("%s", string(marshal))
}
