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

type collectorType string

const (
	centralCollector collectorType = "central_collector"
	logAnalysis      collectorType = "log_analysis"
)

type remoteServiceInf interface {
	// SendLogWithURLString(data []byte, urlStr string) error
	SendLog(data []byte) error
	GetURL() string
	SetURL(u string)
	Type() collectorType
}

type collectorConfig struct {
	Headers              map[string]string
	URL                  string
	RequestTimeout       time.Duration
	KeepAliveIdleTimeout time.Duration
	BasicAuthUsername    string
	BasicAuthPassword    string

	// 流量限制
	NetLimitBytesPerSecond int

	collectorType collectorType
}

type collectorService struct {
	cfg     collectorConfig
	client  *http.Client
	limiter *rate.Limiter
}

func newCollectorService(cfg collectorConfig) *collectorService {
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

func (c *collectorService) GetURL() string {
	return c.cfg.URL
}

func (c *collectorService) SetURL(u string) {
	c.cfg.URL = u
}

func (c *collectorService) Type() collectorType {
	return c.cfg.collectorType
}

func (c *collectorService) BasicAuth() {
	if c.cfg.BasicAuthPassword != "" && c.cfg.BasicAuthUsername != "" {
		token := basicAuth(c.cfg.BasicAuthUsername, c.cfg.BasicAuthPassword)
		c.cfg.Headers["Authorization"] = "Basic " + token
	}
}

// func (c *collectorService) SendLogWithURLString(data []byte, urlStr string) error {
// 	u, err := url.Parse(urlStr)
// 	if err != nil {
// 		return fmt.Errorf("invalide url: %s, err: %w", urlStr, err)
// 	}
// 	return c.sendLogWithURL(data, u.String())
// }

func (c *collectorService) SendLog(data []byte) error {
	return c.sendLogWithURL(data, c.cfg.URL)
}

func (c *collectorService) sendLogWithURL(data []byte, u string) error {
	// block until ok
	if c.limiter != nil {
		r := c.limiter.ReserveN(time.Now(), len(data))
		if !r.OK() {
			newBurst := c.limiter.Burst() << 1
			c.limiter.SetBurst(newBurst)
			return fmt.Errorf("double of burst to %d", newBurst)
		}
		time.Sleep(r.Delay())
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(data))
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
