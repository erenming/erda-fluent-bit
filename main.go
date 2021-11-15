package main

import (
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
	defaultEventLimit             = 5000
	defaultNetLimitBytesPerSecond = 1 * 1024 * 1024 // 1MB/s
	defaultEventContentBytesLimit = 3 * 1024 * 1024 // 3MB * 25% = 0.75MB < 1MB/s
)

func defaultConfig() outerda.Config {
	return outerda.Config{
		RemoteConfig: outerda.RemoteConfig{
			Headers:                map[string]string{},
			JobPath:                "/collect/logs/job",
			ContainerPath:          "/collect/logs/container",
			RequestTimeout:         time.Second * 10,
			KeepAliveIdleTimeout:   time.Second * 60,
			NetLimitBytesPerSecond: defaultNetLimitBytesPerSecond,
		},
		CompressLevel:                  3,
		DockerContainerRootPath:        "/var/lib/docker/containers",
		DockerConfigSyncInterval:       10 * time.Minute,
		DockerConfigMaxExpiredDuration: time.Hour,
		// usual format: /var/lib/docker/containers/<id>/<id>-json.log
		DockerContainerIDIndex:      -2,
		BatchEventLimit:             defaultEventLimit,
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

	outErdaInstance = outerda.NewOutput(cfg)
	if err := outErdaInstance.Start(); err != nil {
		outerda.LogError("start failed", err)
		return output.FLB_ERROR
	}

	return output.FLB_OK
}

//export FLBPluginFlush
func FLBPluginFlush(data unsafe.Pointer, length C.int, tag *C.char) int {
	// var count int
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
			timestamp = time.Now()
		}

		// data, err := outerda.MarshalRecord(record)
		// if err != nil {
		// 	logrus.Error(err)
		// 	return output.FLB_RETRY
		// }
		// if _, ok := record["time"]; !ok {
		// 	fmt.Printf("[%d] %s [%s], \n", count, C.GoString(tag), timestamp.String())
		// 	fmt.Printf("\tdata: %s\n", string(data))
		// }

		if val := outErdaInstance.AddEvent(&outerda.Event{Record: record, Timestamp: timestamp}); val != output.FLB_OK {
			outErdaInstance.Reset()
			return val
		}

		// count++
	}
	err := outErdaInstance.Flush()
	if err != nil {
		outerda.LogError("Flush error", err)
		outErdaInstance.Reset()
		return output.FLB_RETRY
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
	if outErdaInstance == nil {
		return output.FLB_OK
	}
	if err := outErdaInstance.Close(); err != nil {
		outerda.LogError("close output failed", err)
	}
	return output.FLB_OK
}

func main() {
}
