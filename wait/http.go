package wait

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// HTTPStrategy waits for an HTTP endpoint to return a matching response.
type HTTPStrategy struct {
	Path              string
	Port              string
	UseTLS            bool
	TLSConfig         *tls.Config
	User              string
	Password          string
	Method            string
	StatusCodeMatcher func(int) bool
	ResponseMatcher   func(io.Reader) bool
	startupTimeout    time.Duration
	PollInterval      time.Duration
}

// Timeout returns the custom timeout for this strategy.
func (s *HTTPStrategy) Timeout() time.Duration {
	return s.startupTimeout
}

// WithPort sets the target port.
func (s *HTTPStrategy) WithPort(port string) *HTTPStrategy {
	s.Port = port
	return s
}

// WithTLS enables HTTPS check.
func (s *HTTPStrategy) WithTLS() *HTTPStrategy {
	s.UseTLS = true
	return s
}

// WithBasicAuth sets username and password for HTTP Basic Auth.
func (s *HTTPStrategy) WithBasicAuth(user, password string) *HTTPStrategy {
	s.User = user
	s.Password = password
	return s
}

// WithMethod sets the HTTP method.
func (s *HTTPStrategy) WithMethod(method string) *HTTPStrategy {
	s.Method = method
	return s
}

// WithStatusCodeMatcher sets a custom matcher for response status codes.
func (s *HTTPStrategy) WithStatusCodeMatcher(matcher func(int) bool) *HTTPStrategy {
	s.StatusCodeMatcher = matcher
	return s
}

// WithResponseMatcher sets a custom matcher for response body.
func (s *HTTPStrategy) WithResponseMatcher(matcher func(io.Reader) bool) *HTTPStrategy {
	s.ResponseMatcher = matcher
	return s
}

// WithStartupTimeout sets the custom startup timeout.
func (s *HTTPStrategy) WithStartupTimeout(d time.Duration) *HTTPStrategy {
	s.startupTimeout = d
	return s
}

// WithPollInterval sets the polling interval.
func (s *HTTPStrategy) WithPollInterval(d time.Duration) *HTTPStrategy {
	s.PollInterval = d
	return s
}

// ForHTTP creates an HTTP wait strategy for the given path.
func ForHTTP(path string) *HTTPStrategy {
	return &HTTPStrategy{
		Path:         path,
		Port:         "80",
		Method:       http.MethodGet,
		PollInterval: 100 * time.Millisecond,
	}
}

// WaitUntilReady queries the HTTP endpoint until it returns a matching status and body, or times out.
func (s *HTTPStrategy) WaitUntilReady(ctx context.Context, target StrategyTarget) error {
	ticker := time.NewTicker(s.PollInterval)
	defer ticker.Stop()

	// By default, match 200 OK
	statusMatcher := s.StatusCodeMatcher
	if statusMatcher == nil {
		statusMatcher = func(code int) bool {
			return code == http.StatusOK
		}
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if s.TLSConfig != nil {
		transport.TLSClientConfig = s.TLSConfig
	}
	client := &http.Client{
		Transport: transport,
		Timeout:   2 * time.Second,
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			host, err := target.Host(ctx)
			if err != nil {
				continue
			}

			mappedPort, err := target.MappedPort(ctx, s.Port)
			if err != nil {
				continue
			}

			scheme := "http"
			if s.UseTLS {
				scheme = "https"
			}

			u := url.URL{
				Scheme: scheme,
				Host:   net.JoinHostPort(host, strconv.Itoa(mappedPort)),
				Path:   s.Path,
			}

			req, err := http.NewRequestWithContext(ctx, s.Method, u.String(), nil)
			if err != nil {
				continue
			}

			if s.User != "" || s.Password != "" {
				req.SetBasicAuth(s.User, s.Password)
			}

			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			ok := statusMatcher(resp.StatusCode)
			if ok && s.ResponseMatcher != nil {
				ok = s.ResponseMatcher(resp.Body)
			}
			_ = resp.Body.Close()

			if ok {
				return nil
			}
		}
	}
}
