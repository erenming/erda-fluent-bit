package outerda

import (
	"encoding/base64"
	"fmt"
	"time"
	"unsafe"

	"github.com/sirupsen/logrus"
)

func bs2str(bs []byte) string {
	return *(*string)(unsafe.Pointer(&bs))
}

func getAndConvert(key string, record map[interface{}]interface{}, defaultVal interface{}) (interface{}, error) {
	val, ok := record[key]
	if !ok {
		if defaultVal == nil {
			return nil, fmt.Errorf("key %s: %w", key, ErrKeyMustExist)
		} else {
			return defaultVal, nil
		}
	}

	switch val.(type) {
	case float64:
		return uint64(int(val.(float64))), nil
	case uint64:
		return val.(uint64), nil
	case string:
		return val.(string), nil
	case []byte:
		return bs2str(val.([]byte)), nil
	case map[interface{}]interface{}:
		_, ok := defaultVal.(map[string]string)
		if ok {
			data := val.(map[interface{}]interface{})
			m := make(map[string]string, len(data))
			for k, _ := range data {
				tmp, _ := getAndConvert(k.(string), data, "")
				m[k.(string)] = tmp.(string)
			}
			return m, nil
		}
		return val, nil
	default:
		return nil, fmt.Errorf("uncaughted type <%T> with key<%s>: %w", val, key, ErrTypeInvalid)
	}
}

func getTime(record map[interface{}]interface{}) (time.Time, error) {
	timeStr, err := getAndConvert("time", record, "")
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, timeStr.(string))
	if err != nil {
		return time.Time{}, fmt.Errorf("parse time: %w", err)
	}

	return t, nil
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

func jsonRecord(record map[interface{}]interface{}) string {
	buf, _ := json.Marshal(record)
	return string(buf)
}
