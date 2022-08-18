package rawhttp

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"go.uber.org/ratelimit"
)

var (
	RespectRetryAfterHeader = true
	MaxDialTimeout          = 10 // Should not be changed (unless explicitly required)
	MaxHTTPTimeout          = 60 // Should not be changed (unless explicitly required)
	Retryon502              = false
)

/*
SHTTPClient = Smart Http Client.

A Multi Purpose Client made by wrapping  http.Client Which handles
lame issues and provides goto client for bug hunter/security professional

1. Rate Limit
2. Retry (with Count)
3. Proxy
4. Dial Duration
5. Timeout
6. Insecure TLS Certificate
7. Max Idle Connections (& more)

Note : Client Will only be created when Create() Method is Called

It only makes sense to retry only if error is a network error (i.e timeout) or
if it is a server error 502,503 etc

If Timeout is detected
Timeout's are incrementented after each unsuccessful retry
With
Max DialTimeout = 10 sec
Max Timeout = 60 sec

If a 50x status code  error occurs. it retries with given retry count
*/
type SHTTPClient struct {
	ValidateCertificate bool // Validate TLS Certificate (Default: false)
	FollowRedirect      bool // Follow Redirect (Default: true)
	RetryCount          int  // (Default: 3)

	MaxConnections        int // Per Host MaxIdle Connections (Default: 100)
	IdleConnectionTimeout int // Idle Connection Timeout (Default: 10)

	DialTimeout  int // Timeout to make a tcp/ip connection (Default: 5)
	TotalTimeout int // Total Timeout of a connection (includes all) (Default: 30)

	RLPerSec    int    // Rate Limit Per Second (Default : Unlimited)
	RLPerMinute int    // Rate Limit Per Minute (Default : Unlimited)
	ProxyURL    string // Proxy URL

	client  *http.Client      //  Acutal Client
	t       *http.Transport   //  InternalUse Only
	limiter ratelimit.Limiter // InternalUse Only RateLimiter Client
	// Note
	// Retries does not follow rate-limit
}

// Create : Use Settings and Create Client
func (c *SHTTPClient) Create() {

	var Proxy func(*http.Request) (*url.URL, error) = nil

	// Set Proxy if given
	if c.ProxyURL != "" {
		proxyurl, err := url.Parse(c.ProxyURL)
		if err == nil {
			Proxy = http.ProxyURL(proxyurl)
		}
	}

	// TLS Connection Config
	tlsconfig := tls.Config{
		InsecureSkipVerify: !c.ValidateCertificate,
	}

	if c.DialTimeout == 0 {
		// Default Dial Timeout is 5 sec
		c.DialTimeout = 5
	}

	// Dialer used for tcp/ip connections
	dialer := &net.Dialer{
		Timeout: time.Duration(c.DialTimeout) * time.Second,
	}

	// if c.t == nil {

	if c.MaxConnections == 0 {
		c.MaxConnections = 100
	}

	if c.IdleConnectionTimeout == 0 {
		c.IdleConnectionTimeout = 10
	}

	c.t = &http.Transport{
		MaxIdleConnsPerHost: c.MaxConnections,
		MaxIdleConns:        c.MaxConnections,
		IdleConnTimeout:     time.Duration(c.IdleConnectionTimeout) * time.Second,
		TLSClientConfig:     &tlsconfig,
		Proxy:               Proxy,
		DialContext:         dialer.DialContext,
		ForceAttemptHTTP2:   true,
	}
	// }

	if c.TotalTimeout == 0 {
		c.TotalTimeout = 30
	}

	var RedirectFunction func(req *http.Request, via []*http.Request) error = nil

	if !c.FollowRedirect {
		RedirectFunction = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// New HTTP Client
	c.client = &http.Client{
		Transport:     c.t,
		Timeout:       time.Duration(c.TotalTimeout) * time.Second,
		CheckRedirect: RedirectFunction,
	}

	//Configure rate limits
	if c.RLPerMinute == 0 && c.RLPerSec == 0 {
		c.limiter = ratelimit.NewUnlimited()
	} else if c.RLPerMinute != 0 {
		c.limiter = ratelimit.New(c.RLPerMinute, ratelimit.Per(60*time.Second))
	} else if c.RLPerSec != 0 {
		c.limiter = ratelimit.New(c.RLPerSec)
	}

}

// CreateUsingTransport : Optional Method to Override defaults+more control using given transport struct
func (c *SHTTPClient) CreateUsingTransport(t *http.Transport) {
	if c.TotalTimeout == 0 {
		c.TotalTimeout = 30
	}

	var RedirectFunction func(req *http.Request, via []*http.Request) error = nil

	if !c.FollowRedirect {
		RedirectFunction = func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	// New HTTP Client
	c.client = &http.Client{
		Transport:     t,
		Timeout:       time.Duration(c.TotalTimeout) * time.Second,
		CheckRedirect: RedirectFunction,
	}

	//Configure rate limits
	if c.RLPerMinute == 0 && c.RLPerSec == 0 {
		c.limiter = ratelimit.NewUnlimited()
	} else if c.RLPerMinute != 0 {
		c.limiter = ratelimit.New(c.RLPerMinute, ratelimit.Per(60*time.Second))
	} else if c.RLPerSec != 0 {
		c.limiter = ratelimit.New(c.RLPerSec)
	}
}

// Get : Send HTTP Get Request
func (c *SHTTPClient) Get(url string) (*http.Response, error) {
	req, er1 := http.NewRequest("GET", url, nil)
	if er1 != nil {
		return nil, er1
	}

	return c.Do(req)

}

// POST : Send HTTP Post Request
func (c *SHTTPClient) Post(url string, ContentType string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Content-Type", ContentType)

	return c.Do(req)
}

// Do : Send HTTP Request
func (c *SHTTPClient) Do(req *http.Request) (*http.Response, error) {

	c.limiter.Take()
	resp, err := c.client.Do(req)

	if c.RetryCount == 0 {
		return resp, err
	}

	if err == nil {
		if resp != nil && resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented {
			// retry in this case && check retry-after header
			//parse retry after settings if given

			if Retryon502 {
				// if enabled retry
				return c.retry(req, 1)
			}

			if RespectRetryAfterHeader {
				if w, ok := resp.Header["Retry-After"]; ok {
					if sleep, err := strconv.ParseInt(w[0], 10, 64); err == nil {
						//sleep here
						time.Sleep(time.Duration(sleep) * time.Second)
						return c.retry(req, 1)
					}
				}

			}

		}

		return resp, err

	} else {

		// Check type of error
		if err, ok := err.(net.Error); ok && err.Timeout() {
			// A timeout error occurred
			// Handle it Using timeout retry
			return c.timeoutretry(req, 1)
		}

		// If its any other err retrying it has no point
		return resp, err
	}

}

func (c *SHTTPClient) timeoutretry(req *http.Request, retrycount int) (*http.Response, error) {

	resp, err := c.client.Do(req)

	if c.RetryCount == retrycount {
		return resp, err
	}

	if err != nil {
		if err, ok := err.(net.Error); ok && err.Timeout() {
			// Again If it's a timeout error
			if c.DialTimeout < MaxDialTimeout {
				c.DialTimeout += 3 //  Increase Dial Timeout
			}
			if c.TotalTimeout < MaxHTTPTimeout {
				c.TotalTimeout += 10 // Increase Total Timeout
			}
			c.Create() // regenerate

			return c.timeoutretry(req, retrycount+1)

		} else {
			// If it's not a timeout error
			// Retrying it does not make any sense
			return resp, err
		}
	} else {

		//If there is no error but status code is 50x and retry header is given then
		if resp != nil && resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented {
			// retry in this case && check retry-after header
			//parse retry after settings if given

			if Retryon502 {
				// if enabled retry
				return c.retry(req, retrycount)
			}

			if RespectRetryAfterHeader {
				if w, ok := resp.Header["Retry-After"]; ok {
					if sleep, err := strconv.ParseInt(w[0], 10, 64); err == nil {
						//sleep here
						time.Sleep(time.Duration(sleep) * time.Second)
						return c.retry(req, retrycount)
					}
				}

			}

		}

		return resp, err
	}

}

// retry : Retry this request
func (c *SHTTPClient) retry(req *http.Request, retrycount int) (*http.Response, error) {

	resp, err := c.client.Do(req)
	if err == nil && resp != nil {
		if resp.StatusCode >= 500 && resp.StatusCode != http.StatusNotImplemented {

			// If retry count is up do not retry
			if c.RetryCount == retrycount {
				return resp, err
			} else {
				if Retryon502 {
					// if enabled retry
					return c.retry(req, retrycount+1)
				}

				// Check if Retry-After Header is present
				if RespectRetryAfterHeader {
					if w, ok := resp.Header["Retry-After"]; ok {
						if sleep, err := strconv.ParseInt(w[0], 10, 64); err == nil {
							//sleep here
							time.Sleep(time.Duration(sleep) * time.Second)
							return c.retry(req, retrycount+1)
						}
					}
				}

			}
		}
	} else {
		// Check type of error
		if err, ok := err.(net.Error); ok && err.Timeout() {
			// A timeout error occurred
			// Handle it Using timeout retry
			return c.timeoutretry(req, retrycount)
		}

		// If its any other err retrying it has no point
		return resp, err
	}

	return resp, err
}
