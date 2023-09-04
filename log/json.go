package log

import (
	"encoding/json"
	"fmt"
	"time"
)

type JsonFormatter struct {
	TimeDisplay bool
}

func (f *JsonFormatter) Formatter(params *LoggingFormatterParam) string {
	if params.LoggerFields == nil {
		params.LoggerFields = make(Fields)
	}
	if f.TimeDisplay {
		params.LoggerFields["log_time"] = time.Now().Format("2006/01/02 - 15:04:05")
	}
	params.LoggerFields["msg"] = params.Msg
	params.LoggerFields["log_level"] = params.Level.Level()
	marshal, _ := json.Marshal(params.LoggerFields)
	return fmt.Sprint(string(marshal))
}
