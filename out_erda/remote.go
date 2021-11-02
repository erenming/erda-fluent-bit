package outerda

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type remoteServiceInf interface {
	SendContainerLog(data []byte) error
	SendJobLog(data []byte) error // todo should be removed
}

type RemoteConfig struct {
	Headers              map[string]string `fluentbit:"headers"`
	URL                  string            `fluentbit:"erda_ingest_url"`
	JobPath              string            `fluentbit:"job_path"`
	ContainerPath        string            `fluentbit:"container_path"`
	RequestTimeout       time.Duration     `fluentbit:"request_timeout"`
	KeepAliveIdleTimeout time.Duration     `fluentbit:"keep_alive_idle_timeout"`
	BasicAuthUsername    string            `fluentbit:"basic_auth_username"`
	BasicAuthPassword    string            `fluentbit:"basic_auth_password"`

	// 流量限制
	NetLimitBytesPerSecond int `fluentbit:"net_limit_bytes_per_second"`
}

type collectorService struct {
	cfg     RemoteConfig
	client  *http.Client
	limiter *rate.Limiter
}

func newCollectorService(cfg RemoteConfig) *collectorService {
	client := &http.Client{
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   cfg.RequestTimeout,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			// ForceAttemptHTTP2:     true,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   50,
			IdleConnTimeout:       cfg.KeepAliveIdleTimeout,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
		Timeout: cfg.RequestTimeout,
	}
	cs := &collectorService{
		cfg:     cfg,
		client:  client,
		limiter: rate.NewLimiter(rate.Limit(cfg.NetLimitBytesPerSecond), cfg.NetLimitBytesPerSecond),
	}

	cs.BasicAuth()
	return cs
}

func (c *collectorService) SendContainerLog(data []byte) error {
	return c.sendWithPath(data, c.cfg.ContainerPath)
}

func (c *collectorService) SendJobLog(data []byte) error {
	return c.sendWithPath(data, c.cfg.JobPath)
}

func (c *collectorService) BasicAuth() {
	if c.cfg.BasicAuthPassword != "" && c.cfg.BasicAuthUsername != "" {
		token := basicAuth(c.cfg.BasicAuthUsername, c.cfg.BasicAuthPassword)
		c.cfg.Headers["Authorization"] = "Basic " + token
	}
}

func (c *collectorService) sendWithPath(data []byte, path string) error {
	// block until ok
	r := c.limiter.ReserveN(time.Now(), len(data))
	if !r.OK() {
		newBurst := c.limiter.Burst() << 1
		c.limiter.SetBurst(newBurst)
		return fmt.Errorf("double of burst to %d", newBurst)
	}
	time.Sleep(r.Delay())

	req, err := http.NewRequest(http.MethodPost, hostJoinPath(c.cfg.URL, path), bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("new request failed: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("do request failed: %w", err)
	}
	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("copy resp.Body: %w", err)
	}
	defer func() {
		err := resp.Body.Close()
		if err != nil {
			logrus.Infof("close body failed: %s", err)
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("response status code %v is not success", resp.StatusCode)
	}
	return nil
}

func (c *collectorService) setHeaders(req *http.Request) {
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
}

func hostJoinPath(host, path string) string {
	return strings.Join([]string{strings.Trim(host, "/"), strings.Trim(path, "/")}, "/")
}
