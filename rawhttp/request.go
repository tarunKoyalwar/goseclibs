package rawhttp

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

/*
 reading raw http request and converting it to *http.Request

 Primary Goal is fuzzing

 Since this raw request does not intend to replace any
 blacklisted headers or test http request smuggling
 this will just be a wrapper . when time comes will use projectdiscovery/rawhttp
*/

// ForbiddenHeaders = Headers that are ignored
// these headers are usually added by go clinet no need to add manually
var ForbiddenHeaders map[string]bool = map[string]bool{
	"connection":        true,
	"content-length":    true,
	"transfer-encoding": true,
	"trailer":           true,
}

// RawHttpRequest : Raw HTTP request
type RawHttpRequest struct {
	RawURL         string            // Raw URL (Including Parameters)
	PredefinedHost string            // Useful When Fuzzzing Host Header
	Verb           string            // HTTP request Verb
	Path           string            // Relative Path of the request
	Params         url.Values        // Request Params
	Host           string            // Hostname / VHOST
	Headers        map[string]string // Headers
	Cookies        map[string]string // Cookies of a raw request
	ContentType    string            //Content-type of request
	Body           string            // Http request body
	HasBody        bool              // If request body is present
}

// getCookie : Construct Cookie From Data
func (r *RawHttpRequest) getCookie() string {
	rawcookie := ""
	if len(r.Cookies) == 0 {
		return rawcookie
	} else {
		for k, v := range r.Cookies {
			rawcookie += k + "=" + v + "; "
		}
	}

	return rawcookie

}

func (r *RawHttpRequest) GetRequest() *http.Request {
	var req *http.Request

	// Must construct URL Everytime to update changes
	//Construct request url
	var url *url.URL

	if r.PredefinedHost != "" {
		url, _ = url.Parse("https://" + r.PredefinedHost)
	} else {
		url, _ = url.Parse("https://" + r.Host)
	}

	z, _ := url.Parse(r.Path)
	z.RawQuery = r.Params.Encode()

	if r.HasBody {
		req, _ = http.NewRequest(r.Verb, z.String(), bytes.NewReader([]byte(r.Body)))
	} else {
		req, _ = http.NewRequest(r.Verb, z.String(), nil)
	}

	req.Host = r.Host

	cookie := r.getCookie()
	if cookie != "" {
		req.Header.Add("Cookie", cookie)
	}

	//Add remaining headers
	for k, v := range r.Headers {
		req.Header.Add(k, v)
	}

	if r.ContentType != "" {
		// Ovverrite Content-Type
		req.Header.Add("Content-Type", r.ContentType)
	}

	return req

}

// Parse : Parse request to struct
func (r *RawHttpRequest) Parse(dat string) error {
	r.Headers = map[string]string{}

	r.Cookies = map[string]string{}

	//Split request and body
	// remove all \r
	raw := strings.ReplaceAll(dat, "\r", "")

	// This will be a seperator for request and body
	arr := strings.Split(raw, "\n\n")

	// Everything except the request body
	var request string

	if len(arr) == 1 {
		// Body Missing
		request = arr[0]
	} else if len(arr) == 2 {
		request = arr[0]
		r.Body = strings.TrimSpace(arr[1])
		if len(r.Body) > 0 {
			r.HasBody = true
		}
	} else if len(arr) > 2 {
		//If by chance body also has \n\n
		request = arr[0]
		r.Body = strings.Join(arr[1:], "\n\n")
		r.Body = strings.TrimSpace(r.Body)
		if len(r.Body) > 0 {
			r.HasBody = true
		}
	}

	// Parse Upper body of request
	for k, v := range Split(request, '\n') {
		if k == 0 {
			if !strings.Contains(v, "HTTP") {
				return fmt.Errorf("this isn't a raw request no HTTP/1.1 found")
			}
			line := strings.Fields(v)
			if len(line) < 2 {
				return fmt.Errorf("this isn't a raw request split is lessthat 2 ")
			}
			r.Verb = line[0]
			r.Path = line[1]
			//ignore the protocol for now (default to HTTP/2)
		} else if k == 1 {
			v = strings.ToLower(v) // Just a percaution while writing raw request
			if !strings.Contains(v, "host") {
				return fmt.Errorf("this isn't a raw request no host header found")
			}
			r.Host = strings.TrimLeft(v, "host:")
			r.Host = strings.TrimSpace(r.Host)

			//Construct request url
			var url *url.URL

			if r.PredefinedHost != "" {
				url, _ = url.Parse("https://" + r.PredefinedHost)
			} else {
				url, _ = url.Parse("https://" + r.Host)
			}

			z, _ := url.Parse(r.Path)
			r.RawURL = z.String()

			r.Params = z.Query()

		} else {
			// No any Condition Now
			// Split at :
			x := Split(v, ':')
			key := strings.ToLower(x[0])
			if len(x) < 2 {
				return fmt.Errorf("malformed header & value %v", v)
			}
			if ForbiddenHeaders[key] {
				continue
			}

			val := strings.TrimSpace(x[1])

			if key == "cookie" {
				rawcookie := Split(val, ';')
				for _, b := range rawcookie {
					cookie := Split(b, '=')
					if len(cookie) == 2 {
						r.Cookies[strings.TrimSpace(cookie[0])] = strings.TrimSpace(cookie[1])
					}
				}
			} else {
				// other headers
				r.Headers[key] = val
			}

			if key == "content-type" {
				r.ContentType = val
			}
		}
	}

	return nil

}

// NewRawHttpRequest : New Raw Http Request From string
func NewRawHttpRequest(dat string) (*RawHttpRequest, error) {
	r := RawHttpRequest{}
	er := r.Parse(dat)

	return &r, er
}

// NewRawHttpRequestFromBytes : New Raw Http Request From bytes
func NewRawHttpRequestFromBytes(bin []byte) (*RawHttpRequest, error) {
	return NewRawHttpRequest(string(bin))
}

// SplitAtSpace : Similar to strings.Feilds but only considers ' '
func SplitAtSpace(s string) []string {

	return Split(s, ' ')

}

// Split : Similar to Strings.Feilds with Custom separator
func Split(s string, delim rune) []string {

	// Must trim the string first
	s = strings.TrimSpace(s)

	arr := []string{}

	var sb strings.Builder

	for _, v := range s {
		if v != delim {
			sb.WriteRune(v)
		} else {
			if sb.Len() != 0 {
				arr = append(arr, sb.String())
				sb.Reset()
			}
		}
	}

	if sb.Len() != 0 {
		arr = append(arr, sb.String())
		sb.Reset()
	}

	return arr
}
