package main

import (
	"fmt"
	"time"
	"unsafe"

	"C"
	outerda "github.com/erda-project/erda-for-fluent-bit/out_erda"
	"github.com/sirupsen/logrus"

	"github.com/fluent/fluent-bit-go/output"
)

func init() {
	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

var outErdaInstance *outerda.Output

const (
	defaultEventLimit             = 50
	defaultTriggerTimeout         = time.Second * 5
	defaultNetWriteBytesPerSecond = 10 * 1024 * 1024 // 10MB/s
	defaultEventContentBytesLimit = 4 * 1024 * 1024  // 4MB
)

func defaultConfig() outerda.Config {
	return outerda.Config{
		RemoteConfig: outerda.RemoteConfig{
			JobPath:              "/collector/logs/job",
			ContainerPath:        "/collector/logs/container",
			RequestTimeout:       time.Second * 10,
			KeepAliveIdleTimeout: time.Second * 60,
		},
		CompressLevel:               3,
		DockerContainerRootPath:     "/var/lib/docker/containers",
		DockerConfigSyncInterval:    time.Second * 10,
		BatchEventLimit:             defaultEventLimit,
		BatchTriggerTimeout:         defaultTriggerTimeout,
		BatchNetWriteBytesPerSecond: defaultNetWriteBytesPerSecond,
		BatchEventContentLimitBytes: defaultEventContentBytesLimit,
	}
}

//export FLBPluginRegister
func FLBPluginRegister(def unsafe.Pointer) int {
	return output.FLBPluginRegister(def, "erda", "forward data to erda!")
}

//export FLBPluginInit
// (fluentbit will call this)
// plugin (context) pointer to fluentbit context (state/ c code)
func FLBPluginInit(plugin unsafe.Pointer) int {
	cfg := defaultConfig()
	err := outerda.LoadFromFLBPlugin(&cfg, func(key string) string {
		return output.FLBPluginConfigKey(plugin, key)
	})
	if err != nil {
		outerda.LogError("load error: %s", err)
		return output.FLB_ERROR
	}

	// todo debug
	logrus.Infof("cfg: %+v", cfg)

	outErdaInstance = outerda.NewOutput(cfg)
	if err := outErdaInstance.Start(); err != nil {
		outerda.LogError("start failed", err)
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	var count int
	var ret int
	var ts interface{}
	var record map[interface{}]interface{}

	// Create Fluent Bit decoder
	dec := output.NewDecoder(data, int(length))

	// Iterate Records
	for {
		// Extract Record
		ret, ts, record = output.GetRecord(dec)
		if ret != 0 {
			break
		}

		var timestamp time.Time
		switch t := ts.(type) {
		case output.FLBTime:
			timestamp = ts.(output.FLBTime).Time
		case uint64:
			timestamp = time.Unix(int64(t), 0)
		default:
			fmt.Println("time provided invalid, defaulting to now.")
			timestamp = time.Now()
		}

		if val := outErdaInstance.AddEvent(&outerda.Event{Record: record, Timestamp: timestamp}); val != output.FLB_OK {
			return val
		}

		count++
	}

	// Return options:
	//
	// output.FLB_OK    = data have been processed.
	// output.FLB_ERROR = unrecoverable error, do not try this again.
	// output.FLB_RETRY = retry to flush later.
	return output.FLB_OK
}

//export FLBPluginExit
func FLBPluginExit() int {
	if err := outErdaInstance.Close(); err != nil {
		outerda.LogError("close output failed", err)
	}
	return output.FLB_OK
}

func main() {
}
