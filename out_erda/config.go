package outerda

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
)

type Config struct {
	RemoteConfig  RemoteConfig
	CompressLevel int `fluentbit:"compress_level"`
	// environment key list
	ContainerEnvInclude      []string      `fluentbit:"container_env_include"`
	DockerContainerRootPath  string        `fluentbit:"docker_container_root_path"`
	DockerConfigSyncInterval time.Duration `fluentbit:"docker_config_sync_interval"`

	// 日志事件的最大个数限制
	BatchEventLimit int `fluentbit:"batch_event_limit"`
	// 日志内容大小总和阈值
	BatchEventContentLimitBytes int `fluentbit:"batch_event_content_limit_bytes"`
	// TODO 最大每秒网络输出阈值
	BatchNetWriteBytesPerSecond int `fluentbit:"batch_net_write_bytes_per_second"`
}

func LoadFromFLBPlugin(source interface{}, finder func(key string) string) error {
	return setValue(reflect.ValueOf(source), finder)
}

func setValue(dst reflect.Value, finder func(key string) string) error {
	dst = reflect.Indirect(dst)
	typeDst := dst.Type()

	for i := 0; i < dst.NumField(); i++ {
		t := typeDst.Field(i)
		v := dst.Field(i)

		if v.Kind() == reflect.Struct {
			if err := setValue(v, finder); err != nil {
				return err
			}
		}

		if val, ok := t.Tag.Lookup("fluentbit"); ok {
			data := finder(val)
			if data == "" {
				continue
			}
			switch v.Interface().(type) {
			case int:
				tmp, err := strconv.Atoi(data)
				if err != nil {
					return fmt.Errorf("convert field %s failed: %w ", val, err)
				}
				v.SetInt(int64(tmp))
			case time.Duration:
				tmp, err := time.ParseDuration(data)
				if err != nil {
					return fmt.Errorf("convert field %s failed: %w ", val, err)
				}
				v.SetInt(int64(tmp))

			case string:
				v.SetString(data)
			case []string:
				v.Set(reflect.ValueOf(strings.Split(data, ",")))
			case map[string]string:
				tmp := reflect.MakeMap(v.Type())
				for _, item := range strings.Split(data, ",") {
					idx := strings.Index(item, "=")
					if idx != -1 {
						tmp.SetMapIndex(reflect.ValueOf(item[:idx]), reflect.ValueOf(item[idx+1:]))
					}
				}
				v.Set(tmp)
			default:
				return fmt.Errorf("unsupported field %s", val)
			}
		}
	}
	return nil
}

func (cfg *Config) SetConfigValue(plugin unsafe.Pointer, key string, setter func(value string) error) error {
	val := output.FLBPluginConfigKey(plugin, key)
	if val == "" {
		return nil
	}
	return setter(val)
}
