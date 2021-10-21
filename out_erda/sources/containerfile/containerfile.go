package containerfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const configJson = "config.v2.json"

type DockerContainerID string

type DockerContainerInfo struct {
	ID     DockerContainerID
	Name   string
	EnvMap map[string]string
}

type dockerConfigV2 struct {
	ID     DockerContainerID `json:"ID"`
	Name   string            `json:"Name"`
	Config struct {
		Env []string `json:"Env"`
	} `json:"Config"`
}

type Config struct {
	RootPath       string
	EnvIncludeList []string
	SyncInterval   time.Duration
}

type ContainerInfoCenter struct {
	rootPath      string
	globPattern   string
	syncInterval  time.Duration
	envKeyInclude map[string]struct{}
	mu            sync.RWMutex
	// todo should not exported
	Data map[DockerContainerID]DockerContainerInfo
}

func NewContainerInfoCenter(cfg Config) *ContainerInfoCenter {
	return &ContainerInfoCenter{
		syncInterval:  cfg.SyncInterval,
		rootPath:      cfg.RootPath,
		globPattern:   filepath.Join(cfg.RootPath, "*", configJson),
		envKeyInclude: listToMap(cfg.EnvIncludeList),
		Data:          make(map[DockerContainerID]DockerContainerInfo),
	}
}

func (ci *ContainerInfoCenter) watchDirFile() {
	// todo
}

func (ci *ContainerInfoCenter) Init() error {
	return ci.scan()
}

func (ci *ContainerInfoCenter) Start() {
	go ci.syncWithInterval()
}

func (ci *ContainerInfoCenter) syncWithInterval() {
	ticker := time.NewTicker(ci.syncInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			err := ci.scan()
			if err != nil {
				logrus.Errorf("sync scan failed: %s", err)
			}
		}
	}
}

func (ci *ContainerInfoCenter) scan() error {
	files, err := filepath.Glob(ci.globPattern)
	if err != nil {
		return err
	}
	data := make(map[DockerContainerID]DockerContainerInfo, len(files))
	for _, f := range files {
		buf, err := os.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read file %s failed: %w", f, err)
		}
		tmp := dockerConfigV2{}
		err = json.Unmarshal(buf, &tmp)
		if err != nil {
			return fmt.Errorf("unmarshal filed %s fialed: %w", f, err)
		}
		data[tmp.ID] = ci.convert(tmp)
	}
	ci.mu.Lock()
	ci.Data = data
	ci.mu.Unlock()
	return nil
}

func (ci *ContainerInfoCenter) convert(src dockerConfigV2) DockerContainerInfo {
	envmap := make(map[string]string)
	for _, item := range src.Config.Env {
		idx := strings.Index(item, "=")
		key, val := item[:idx], item[idx+1:]
		if _, ok := ci.envKeyInclude[key]; ok {
			envmap[key] = val
		}
	}
	return DockerContainerInfo{
		ID:     src.ID,
		Name:   src.Name,
		EnvMap: envmap,
	}
}

func (ci *ContainerInfoCenter) GetInfoByContainerID(cid string) (DockerContainerInfo, bool) {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	res, ok := ci.Data[DockerContainerID(cid)]
	return res, ok
}

func listToMap(list []string) map[string]struct{} {
	res := make(map[string]struct{}, len(list))
	for _, item := range list {
		res[item] = struct{}{}
	}
	return res
}
