package outerda

import (
	"encoding/base64"
	"fmt"
	"strings"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

func bs2str(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func PrettyRecord(record map[interface{}]interface{}, depth int) {
	for k, v := range record {
		fmt.Printf(strings.Repeat("\t", depth)+"k: %s, ", k)
		switch v.(type) {
		case []byte:
			fmt.Printf("v: %s\n", string(v.([]byte)))
		case float64:
			fmt.Printf("v: %f\n", v.(float64))
		case uint64:
			fmt.Printf("v: %d\n", v.(uint64))
		case string:
			fmt.Printf("v: %s\n", v)
		case map[interface{}]interface{}:
			fmt.Println()
			PrettyRecord(record[k].(map[interface{}]interface{}), depth+1)
		default:
		}
	}
}

type RecordStruct struct {
	Time   string `json:"time,omitempty"`
	Log    string `json:"log,omitempty"`
	Stream string `json:"stream,omitempty"`
	Offset uint64 `json:"offset,omitempty"`
}

func MarshalRecord(record map[interface{}]interface{}) ([]byte, error) {
	rs := &RecordStruct{}
	if record["time"] != nil {
		rs.Time = string(record["time"].([]byte))
	}
	if record["log"] != nil {
		rs.Log = string(record["log"].([]byte))
	}
	if record["stream"] != nil {
		rs.Stream = string(record["stream"].([]byte))
	}
	if record["offset"] != nil {
		rs.Offset = record["offset"].(uint64)
	}
	return json.Marshal(rs)
}

func getAndConvert(key string, record map[interface{}]interface{}, defaultVal interface{}) (interface{}, error) {
	val, ok := record[key]
	if !ok {
		if defaultVal == nil {
			return nil, fmt.Errorf("key %s: %w", key, ErrKeyMustExist)
		} else {
			logrus.Infof("key %s not existed, use default %+v", key, defaultVal)
			// PrettyRecord(record, 1)
			return defaultVal, nil
		}
	}

	var data interface{}
	switch val.(type) {
	case float64:
		data = uint64(int(val.(float64)))
	case uint64:
		data = val.(uint64)
	case string:
		data = val.(string)
	case []byte:
		data = val.([]byte)
	default:
		return nil, fmt.Errorf("uncaughted type <%T> with key<%s>: %w", val, key, ErrTypeInvalid)
	}
	return data, nil
}

func getTime(record map[interface{}]interface{}) (time.Time, error) {
	timeStr, err := getAndConvert("time", record, nil)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, bs2str(timeStr.([]byte)))
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}

	return t, nil
}

func getLogPath(record map[interface{}]interface{}) string {
	path, ok := record["log_path"]
	if !ok {
		return ""
	}
	return bs2str(path.([]byte))
}

func LogError(message string, err error) {
	logrus.Errorf("[out_erda] ERROR %s: %s", message, err)
}

func LogInfo(message string, err error) {
	logrus.Infof("[out_erda] INFO %s: %s", message, err)
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
