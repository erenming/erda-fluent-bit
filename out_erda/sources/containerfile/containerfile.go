package containerfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sirupsen/logrus"
)

const configJson = "config.v2.json"

type DockerContainerID string

type DockerContainerInfo struct {
	ID     DockerContainerID
	Name   string
	EnvMap map[string]string

	// for debug
	configFilePath string
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
	done          chan struct{}
	watcher       *fsnotify.Watcher
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
		done:          make(chan struct{}),
	}
}

func (ci *ContainerInfoCenter) Init() error {
	err := ci.initWatcher()
	if err != nil {
		return fmt.Errorf("init watcher: %w", err)
	}
	err = ci.scan()
	if err != nil {
		return fmt.Errorf("init scan: %w", err)
	}
	return nil
}

func (ci *ContainerInfoCenter) Start() {
	go ci.syncWithInterval()
	go ci.watchFileChange()
}

func (ci *ContainerInfoCenter) Close() error {
	close(ci.done)
	return ci.watcher.Close()
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
		case <-ci.done:
			return
		}
	}
}

func (ci *ContainerInfoCenter) watchFileChange() {
	for {
		select {
		case event, ok := <-ci.watcher.Events:
			if !ok {
				return
			}
			if (event.Op & fsnotify.Create) == fsnotify.Create {
				f := filepath.Join(event.Name, configJson)
				time.Sleep(2 * time.Second) // in case flushing
				dinfo, err := ci.readConfigFile(f)
				if err != nil {
					logrus.Errorf("readConfigFile event<%s> fialed: %s", event.Name, err)
					continue
				}
				ci.mu.Lock()
				ci.Data[dinfo.ID] = dinfo
				ci.mu.Unlock()
				logrus.Infof("inotify: event<%s> created. load file: %s success!", event.Name, f)
			}
		case event, ok := <-ci.watcher.Errors:
			if !ok {
				return
			}
			logrus.Errorf("error event received: %s", event.Error())
		case <-ci.done:
			return
		}
	}
}

func (ci *ContainerInfoCenter) initWatcher() error {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("create fs watcher failed")
	}
	err = w.Add(ci.rootPath)
	if err != nil {
		return fmt.Errorf("add dir: %w", err)
	}
	ci.watcher = w
	return nil
}

func (ci *ContainerInfoCenter) scan() error {
	files, err := filepath.Glob(ci.globPattern)
	if err != nil {
		return err
	}
	data := make(map[DockerContainerID]DockerContainerInfo, len(files))
	for _, f := range files {
		dinfo, err := ci.readConfigFile(f)
		if err != nil {
			return err
		}
		data[dinfo.ID] = dinfo
	}
	ci.mu.Lock()
	ci.Data = data
	ci.mu.Unlock()
	return nil
}

func (ci *ContainerInfoCenter) readConfigFile(f string) (DockerContainerInfo, error) {
	buf, err := os.ReadFile(f)
	if err != nil {
		return DockerContainerInfo{}, fmt.Errorf("read file %s failed: %w", f, err)
	}
	tmp := dockerConfigV2{}
	err = json.Unmarshal(buf, &tmp)
	if err != nil {
		return DockerContainerInfo{}, fmt.Errorf("unmarshal filed %s fialed: %w", f, err)
	}
	di := ci.convert(tmp)
	di.configFilePath = f
	return di, nil
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
