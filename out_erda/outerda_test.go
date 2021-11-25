package outerda

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/erda-project/erda-for-fluent-bit/out_erda/sources/containerfile"
	"github.com/stretchr/testify/assert"
)

func BenchmarkOutput_Process(b *testing.B) {
	o := &Output{
		cfg:   mockCfg,
		cache: mockCache(false),
	}
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = o.Process(mockTimestamp, mockRecord())
	}
}

func TestOutput_Process(t *testing.T) {
	type fields struct {
		output *Output
	}
	type args struct {
		timestamp time.Time
		record    map[interface{}]interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *LogEvent
		wantErr bool
	}{
		{
			name: "normal container",
			fields: fields{
				output: &Output{
					cfg:   mockCfg,
					cache: mockCache(false),
				}},
			args: args{
				timestamp: mockTimestamp,
				record:    mockRecord(),
			},
			want: &LogEvent{
				Source:    "container",
				ID:        "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
				Stream:    "stderr",
				Content:   "time=\"2021-10-12 16:00:14.130242184\" level=info msg=\"finish to run the task: executor K8S/MARATHONFORTERMINUSDEV (id: 1120384ca1, action: 5)\"",
				Timestamp: 1634025614130323755,
				Tags: map[string]string{
					"pod_name":              "scheduler-3feb156fc4-cf6b45b89-cwh5s",
					"pod_namespace":         "project-387-dev",
					"pod_id":                "ad05d65a-b8b0-4b7c-84f3-88a2abc11bde",
					"pod_ip":                "10.0.46.1",
					"container_id":          "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
					"dice_cluster_name":     "terminus-dev",
					"dice_application_name": "scheduler",
					"msp_env_id":            "abc111",
					"cluster_name":          "terminus-dev",
					"application_name":      "scheduler",
					"container_name":        "scheduler",
				},
				Labels: map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "normal job",
			fields: fields{
				output: &Output{
					cfg:   mockCfg,
					cache: mockCache(true),
				}},
			args: args{
				timestamp: mockTimestamp,
				record:    mockRecord(),
			},
			want: &LogEvent{
				Source:    "job",
				ID:        "pipeline-task-1024",
				Stream:    "stderr",
				Content:   "time=\"2021-10-12 16:00:14.130242184\" level=info msg=\"finish to run the task: executor K8S/MARATHONFORTERMINUSDEV (id: 1120384ca1, action: 5)\"",
				Timestamp: 1634025614130323755,
				Tags: map[string]string{
					"pod_name":              "scheduler-3feb156fc4-cf6b45b89-cwh5s",
					"pod_namespace":         "project-387-dev",
					"pod_id":                "ad05d65a-b8b0-4b7c-84f3-88a2abc11bde",
					"pod_ip":                "10.0.46.1",
					"container_id":          "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
					"dice_cluster_name":     "terminus-dev",
					"dice_application_name": "scheduler",
					"msp_env_id":            "abc111",
					"cluster_name":          "terminus-dev",
					"application_name":      "scheduler",
					"terminus_define_tag":   "pipeline-task-1024",
					"container_name":        "scheduler",
				},
				Labels: map[string]string{},
			},
			wantErr: false,
		},
		{
			name: "invalid Record",
			fields: fields{
				output: &Output{
					cfg:   mockCfg,
					cache: mockCache(false),
				}},
			args: args{
				timestamp: mockTimestamp,
				record: map[interface{}]interface{}{
					"hello": []byte("world"),
				},
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "no offset",
			fields: fields{
				output: &Output{
					cfg:   mockCfg,
					cache: mockCache(false),
				}},
			args: args{
				timestamp: mockTimestamp,
				record: map[interface{}]interface{}{
					"log":      []byte("time=\"2021-10-12 16:00:14.130242184\" level=info msg=\"finish to run the task: executor K8S/MARATHONFORTERMINUSDEV (id: 1120384ca1, action: 5)\"\n"),
					"stream":   []byte("stderr"),
					"log_path": []byte("/testdata/containers/b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd/b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd-json.log"),
					"attrs": map[interface{}]interface{}{
						"TERMINUS_APP": []byte("scheduler"),
						"TERMINUS_KEY": []byte("z179399ebf5ab436c937479640aec4dfa"),
					},
					"time": []byte("2021-10-12T08:00:14.130323755Z"),
					"kubernetes": map[interface{}]interface{}{
						"pod_name":       []byte("scheduler-3feb156fc4-cf6b45b89-cwh5s"),
						"namespace_name": []byte("project-387-dev"),
						"pod_id":         []byte("ad05d65a-b8b0-4b7c-84f3-88a2abc11bde"),
						"labels": map[interface{}]interface{}{
							"app":               []byte("scheduler"),
							"pod-template-hash": []byte("cf6b45b89"),
							"servicegroup-id":   []byte("3feb156fc4"),
						},
						"annotations": map[interface{}]interface{}{
							"cni.projectcalico.org/podIP":  []byte("10.112.5.71/32"),
							"cni.projectcalico.org/podIPs": []byte("10.112.5.71/32"),
							"sidecar.istio.io/inject":      []byte("false"),
						},
						"host":            []byte("node-010000006221"),
						"container_name":  []byte("scheduler"),
						"docker_id":       []byte("b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd"),
						"container_hash":  []byte("registry.cn-hangzhou.aliyuncs.com/dice/scheduler@sha256:60cf74c6690ad427e1f4a98122e038fb6c2d834556f28c0c85a00d03b1277922"),
						"container_image": []byte("registry.cn-hangzhou.aliyuncs.com/dice/scheduler:1.4.0-alpha-20211012114917-eb15d2a3"),
					},
				},
			},
			want: &LogEvent{
				Source:    "container",
				ID:        "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
				Stream:    "stderr",
				Content:   "time=\"2021-10-12 16:00:14.130242184\" level=info msg=\"finish to run the task: executor K8S/MARATHONFORTERMINUSDEV (id: 1120384ca1, action: 5)\"",
				Timestamp: 1634025614130323755,
				Tags: map[string]string{
					"pod_name":              "scheduler-3feb156fc4-cf6b45b89-cwh5s",
					"pod_namespace":         "project-387-dev",
					"pod_id":                "ad05d65a-b8b0-4b7c-84f3-88a2abc11bde",
					"pod_ip":                "10.0.46.1",
					"container_id":          "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
					"dice_cluster_name":     "terminus-dev",
					"dice_application_name": "scheduler",
					"msp_env_id":            "abc111",
					"cluster_name":          "terminus-dev",
					"application_name":      "scheduler",
					"container_name":        "scheduler",
				},
				Labels: map[string]string{},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.output.Process(tt.args.timestamp, tt.args.record)
			if (err != nil) != tt.wantErr {
				t.Errorf("Process() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

// mock
var (
	mockTimestamp = time.Date(2021, 10, 12, 8, 0, 15, 0, time.UTC)
	mockCfg       = Config{
		ContainerEnvInclude:    strings.Split("TERMINUS_DEFINE_TAG,TERMINUS_KEY,MESOS_TASK_ID,DICE_ORG_ID,DICE_ORG_NAME,DICE_PROJECT_ID,DICE_PROJECT_NAME,DICE_APPLICATION_ID,DICE_APPLICATION_NAME,DICE_RUNTIME_ID,DICE_RUNTIME_NAME,DICE_SERVICE_NAME,DICE_WORKSPACE,DICE_COMPONENT,TERMINUS_LOG_KEY,MONITOR_LOG_KEY,DICE_CLUSTER_NAME,MSP_ENV_ID,MSP_LOG_ATTACH,POD_IP", ","),
		DockerContainerIDIndex: -2,
	}
)

func mockCache(isJob bool) *metadataCache {
	res := &metadataCache{
		dockerConfig: &containerfile.ContainerInfoCenter{
			Data: map[containerfile.DockerContainerID]containerfile.DockerContainerInfo{
				"b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd": {
					ID:   "b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd",
					Name: "scheduler",
					EnvMap: map[string]string{
						"DICE_CLUSTER_NAME":     "terminus-dev",
						"DICE_APPLICATION_NAME": "scheduler",
						"MSP_ENV_ID":            "abc111",
						"POD_IP":                "10.0.46.1",
					},
					Labels: map[string]string{
						"io.kubernetes.container.name": "scheduler",
						"io.kubernetes.docker.type":    "container",
						"io.kubernetes.pod.name":       "scheduler-3feb156fc4-cf6b45b89-cwh5s",
						"io.kubernetes.pod.namespace":  "project-387-dev",
						"io.kubernetes.pod.uid":        "ad05d65a-b8b0-4b7c-84f3-88a2abc11bde",
					},
				},
			},
		},
	}
	if isJob {
		res.dockerConfig.Data["b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd"].EnvMap["TERMINUS_DEFINE_TAG"] = "pipeline-task-1024"
	}
	return res
}

func mockRecord() map[interface{}]interface{} {
	return map[interface{}]interface{}{
		"log":      []byte("time=\"2021-10-12 16:00:14.130242184\" level=info msg=\"finish to run the task: executor K8S/MARATHONFORTERMINUSDEV (id: 1120384ca1, action: 5)\"\n"),
		"stream":   []byte("stderr"),
		"log_path": []byte("/testdata/containers/b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd/b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd-json.log"),
		"attrs": map[interface{}]interface{}{
			"TERMINUS_APP": []byte("scheduler"),
			"TERMINUS_KEY": []byte("z179399ebf5ab436c937479640aec4dfa"),
		},
		"time": []byte("2021-10-12T08:00:14.130323755Z"),
		"kubernetes": map[interface{}]interface{}{
			"pod_name":       []byte("scheduler-3feb156fc4-cf6b45b89-cwh5s"),
			"namespace_name": []byte("project-387-dev"),
			"pod_id":         []byte("ad05d65a-b8b0-4b7c-84f3-88a2abc11bde"),
			"labels": map[interface{}]interface{}{
				"app":               []byte("scheduler"),
				"pod-template-hash": []byte("cf6b45b89"),
				"servicegroup-id":   []byte("3feb156fc4"),
			},
			"annotations": map[interface{}]interface{}{
				"cni.projectcalico.org/podIP":  []byte("10.112.5.71/32"),
				"cni.projectcalico.org/podIPs": []byte("10.112.5.71/32"),
				"sidecar.istio.io/inject":      []byte("false"),
			},
			"host":            []byte("node-010000006221"),
			"container_name":  []byte("scheduler"),
			"docker_id":       []byte("b2a9cb046a8275c57307cad907ef0a5553a78d6f4c1da7186566555d1a5383dd"),
			"container_hash":  []byte("registry.cn-hangzhou.aliyuncs.com/dice/scheduler@sha256:60cf74c6690ad427e1f4a98122e038fb6c2d834556f28c0c85a00d03b1277922"),
			"container_image": []byte("registry.cn-hangzhou.aliyuncs.com/dice/scheduler:1.4.0-alpha-20211012114917-eb15d2a3"),
		},
	}
}

func Test_stripNewLine(t *testing.T) {
	type args struct {
		data []byte
	}
	tests := []struct {
		name string
		args args
		want []byte
	}{
		{
			name: "",
			args: args{data: []byte("hello \n")},
			want: []byte("hello "),
		},
		{
			name: "",
			args: args{data: []byte("hello ")},
			want: []byte("hello "),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripNewLine(tt.args.data); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripNewLine() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOutput_enrichWithErdaMetadata(t *testing.T) {
	type args struct {
		lg     *LogEvent
		record map[interface{}]interface{}
	}
	tests := []struct {
		name string
		args args
		want *LogEvent
	}{
		{
			name: "",
			args: args{
				lg: &LogEvent{
					Tags: map[string]string{},
				},
				record: map[interface{}]interface{}{
					"meta_erda_level":      "INFO",
					"meta_erda_request_id": "abc",
				},
			},
			want: &LogEvent{
				Tags: map[string]string{
					"level":      "INFO",
					"request_id": "abc",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Output{}
			o.enrichWithErdaMetadata(tt.args.lg, tt.args.record)
			assert.Equal(t, tt.want, tt.args.lg)
		})
	}
}
