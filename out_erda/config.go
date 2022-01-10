package outerda

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	"github.com/sirupsen/logrus"
)

const compressRatio = 0.25

type Config struct {
	RemoteConfig RemoteConfig

	CompressLevel int `fluentbit:"compress_level"`
	// environment key list
	ContainerEnvInclude            []string      `fluentbit:"container_env_include"`
	DockerContainerMetadataEnable  bool          `fluentbit:"docker_container_metadata_enable"`
	DockerContainerRootPath        string        `fluentbit:"docker_container_root_path"`
	DockerConfigSyncInterval       time.Duration `fluentbit:"docker_config_sync_interval"`
	DockerConfigMaxExpiredDuration time.Duration `fluentbit:"docker_config_max_expired_duration"`
}

type RemoteConfig struct {
	RemoteType           string            `fluentbit:"remote_type"`
	Headers              map[string]string `fluentbit:"headers"`
	URL                  string            `fluentbit:"erda_ingest_url"`
	JobPath              string            `fluentbit:"job_path"`
	ContainerPath        string            `fluentbit:"container_path"`
	RequestTimeout       time.Duration     `fluentbit:"request_timeout"`
	KeepAliveIdleTimeout time.Duration     `fluentbit:"keep_alive_idle_timeout"`
	BasicAuthUsername    string            `fluentbit:"basic_auth_username"`
	BasicAuthPassword    string            `fluentbit:"basic_auth_password"`
}

func (cfg *Config) Init() {
	if collectorType(cfg.RemoteConfig.RemoteType) != logAnalysis && cfg.RemoteConfig.URL == "" {
		erdaURL := "http://" + os.Getenv("COLLECTOR_ADDR")
		if os.Getenv("DICE_IS_EDGE") == "true" {
			erdaURL = os.Getenv("COLLECTOR_PUBLIC_URL")
		}
		cfg.RemoteConfig.URL = erdaURL
	}

	cfg.RemoteConfig.Headers["Content-Type"] = "application/json; charset=UTF-8"
	if cfg.CompressLevel > 0 {
		cfg.RemoteConfig.Headers["Content-Encoding"] = "gzip"
	}

	if v := os.Getenv("LOG_LEVEL"); v != "" {
		if lv, err := logrus.ParseLevel(v); err == nil {
			logrus.SetLevel(lv)
		}
	}
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
			case bool:
				tmp, err := strconv.ParseBool(strings.ToLower(data))
				if err != nil {
					return fmt.Errorf("convert field %s failed: %w", val, err)
				}
				v.SetBool(tmp)
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
