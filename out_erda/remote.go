package outerda

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"golang.org/x/time/rate"
)

type remoteService struct {
	cfg        RemoteConfig
	client     *http.Client
	limiter    *rate.Limiter
	compressor *gzipper
}

type gzipper struct {
	buf    *bytes.Buffer
	writer *gzip.Writer
}

func newCollectorService(cfg RemoteConfig) *remoteService {
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
	cs := &remoteService{
		cfg:    cfg,
		client: client,
	}
	if cfg.GzipLevel > 0 {
		buf := bytes.NewBuffer(nil)
		gc, _ := gzip.NewWriterLevel(buf, cfg.GzipLevel)
		cs.compressor = &gzipper{
			buf:    buf,
			writer: gc,
		}
	}

	cs.BasicAuth()
	return cs
}

func (c *remoteService) GetURL() string {
	return c.cfg.URL
}

func (c *remoteService) SetURL(u string) {
	c.cfg.URL = u
}

func (c *remoteService) BasicAuth() {
	if c.cfg.BasicAuthPassword != "" && c.cfg.BasicAuthUsername != "" {
		token := basicAuth(c.cfg.BasicAuthUsername, c.cfg.BasicAuthPassword)
		c.cfg.Headers["Authorization"] = "Basic " + token
	}
}

func (c *remoteService) SendLogWithURL(data interface{}, u string) error {
	buf, err := c.serializer(data)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(buf))
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

func (c *remoteService) setHeaders(req *http.Request) {
	for k, v := range c.cfg.Headers {
		req.Header.Set(k, v)
	}
}

func (c *remoteService) serializer(data interface{}) ([]byte, error) {
	var buf []byte
	switch c.cfg.Format {
	case "", "json":
		c.cfg.Headers["Content-Type"] = "application/json; charset=UTF-8"
		tmp, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("json marshal: %w", err)
		}
		buf = tmp
	default:
		return nil, fmt.Errorf("unsported format: %s", c.cfg.Format)
	}

	if c.cfg.GzipLevel > 0 {
		c.cfg.Headers["Content-Encoding"] = "gzip"
		if c.compressor != nil {
			tmp, err := c.compress(buf)
			if err != nil {
				return nil, fmt.Errorf("compress failed: %w", err)
			}
			buf = tmp
		}
	}
	return buf, nil
}

func (c *remoteService) compress(data []byte) ([]byte, error) {
	defer func() {
		c.compressor.buf.Reset()
		c.compressor.writer.Reset(c.compressor.buf)
	}()
	if _, err := c.compressor.writer.Write(data); err != nil {
		return nil, fmt.Errorf("gizp write data: %w", err)
	}
	if err := c.compressor.writer.Flush(); err != nil {
		return nil, fmt.Errorf("gzip flush data: %w", err)
	}
	if err := c.compressor.writer.Close(); err != nil {
		return nil, fmt.Errorf("gzip close: %w", err)
	}
	buf := bytes.NewBuffer(nil) // todo init size?
	if _, err := io.Copy(buf, c.compressor.buf); err != nil {
		return nil, fmt.Errorf("gzip copy: %w", err)
	}
	return buf.Bytes(), nil
}

func hostJoinPath(host, path string) string {
	return strings.Join([]string{strings.Trim(host, "/"), strings.Trim(path, "/")}, "/")
}
