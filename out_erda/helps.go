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

func getAndConvert(key string, record map[interface{}]interface{}) (interface{}, error) {
	val, ok := record[key]
	if !ok {
		return nil, fmt.Errorf("key %s: %w", key, ErrKeyMustExist)
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
	timeStr, err := getAndConvert("time", record)
	if err != nil {
		return time.Time{}, err
	}
	t, err := time.Parse(time.RFC3339Nano, bs2str(timeStr.([]byte)))
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
