package logrus

import (
	"encoding/json"
	"fmt"
	"io"
)

type fieldKey string
type FieldMap map[fieldKey]string

const (
	FieldKeyMsg   = "msg"
	FieldKeyLevel = "level"
	FieldKeyTime  = "time"
)

func (f FieldMap) resolve(key fieldKey) string {
	if k, ok := f[key]; ok {
		return k
	}

	return string(key)
}

type JSONFormatter struct {
	// TimestampFormat sets the format used for marshaling timestamps.
	TimestampFormat string

	// DisableTimestamp allows disabling automatic timestamps in output
	DisableTimestamp bool

	// FieldMap allows users to customize the names of keys for various fields.
	// As an example:
	// formatter := &JSONFormatter{
	//   	FieldMap: FieldMap{
	// 		 FieldKeyTime: "@timestamp",
	// 		 FieldKeyLevel: "@level",
	// 		 FieldKeyLevel: "@message",
	//    },
	// }
	FieldMap FieldMap
}

// The internal representation
type jsonFormatter struct {
	timestampFormat string
	keyTime         string
	keyLevel        string
	keyMsg          string
}

func (factory *JSONFormatter) Build(out io.Writer, minimumLevel Level) (Formatter, error) {
	fmt := jsonFormatter{
		timestampFormat: factory.TimestampFormat,
		keyTime:         factory.FieldMap.resolve(FieldKeyTime),
		keyLevel:        factory.FieldMap.resolve(FieldKeyLevel),
		keyMsg:          factory.FieldMap.resolve(FieldKeyMsg),
	}
	if factory.DisableTimestamp {
		fmt.keyTime = ""
	} else {
		if fmt.timestampFormat == "" {
			fmt.timestampFormat = DefaultTimestampFormat
		}
		// TODO more TimestampFormat validation
	}

	return &fmt, nil
}

func (f *jsonFormatter) Format(entry *Entry) ([]byte, error) {
	data := make(Fields, len(entry.Data)+3)
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			// Otherwise errors are ignored by `encoding/json`
			// https://github.com/Sirupsen/logrus/issues/137
			data[k] = v.Error()
		default:
			data[k] = v
		}
	}
	prefixFieldClashes(data)

	if f.keyTime != "" {
		data[f.keyTime] = entry.Time.Format(f.timestampFormat)
	}
	data[f.keyMsg] = entry.Message
	data[f.keyLevel] = entry.Level.String()

	serialized, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Failed to marshal fields to JSON, %v", err)
	}
	return append(serialized, '\n'), nil
}
