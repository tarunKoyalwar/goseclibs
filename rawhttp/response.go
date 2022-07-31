package rawhttp

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

/*
A Wrapper Around http.Response
Goals
1. To read raw responses(ex: Burpsuites)
2. Auto Decode gzip encoded data
3. Response Comparison


Lot of servers send 404 page with 200 statuscode
in such cases `BlackListContentLength` can be used to convert statuscode
from 200 to 404 depending on blacklisted content length

//Format and do other things

*/

var (
	StoreResponse             = true // store pointer to original response
	StoreResponseBody         = true
	BlackListContentLength    = false // Will Modify response from 200 to 404 if response body size matches given blacklisted length
	BlackListContentLenVal    = 0     // Value of Content Length to blacklist
	BlackListContentLenValMin = 0     // Starting  Value of Content Length to blacklist
	BlackListContentLenValMax = 0     // Terminal Value of Content Length to blacklist
)

type RawHttpResponse struct {
	Response      *http.Response //http response(Acutal)
	StatusCode    int
	ContentLength int
	ContentType   string
	Location      string // If response was 302
	Headers       map[string]string
	Cookies       map[string]string
	Body          []byte
}

func NewRawHttpResponse(res *http.Response) (*RawHttpResponse, error) {
	rx := &RawHttpResponse{}

	err := rx.Parse(res)

	return rx, err
}

func NewRawHttpResponseFromBytes(bin []byte) (*RawHttpResponse, error) {
	rx := &RawHttpResponse{}

	err := rx.ParseFromBytes(bin)

	return rx, err
}

func (r *RawHttpResponse) Parse(resp *http.Response) error {

	r.Headers = map[string]string{}
	r.Cookies = map[string]string{}

	if StoreResponse {
		r.Response = resp // Just a reference
	}

	r.StatusCode = resp.StatusCode

	setcookies := resp.Cookies()

	if len(setcookies) > 0 {
		for _, v := range setcookies {
			r.Cookies[v.Name] = v.Value
		}
	}

	//Read response body length instead of content-length header
	defer resp.Body.Close()

	if StoreResponseBody {
		bin, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		r.ContentLength = len(bin)
		r.Body = bin
	} else {
		// By doing this golang will reuse connections
		len, _ := io.Copy(io.Discard, resp.Body)
		r.ContentLength = int(len)
	}

	// Parse Headers
	lc, err := resp.Location()
	if err == nil {
		// If err  is nil then location header is present
		r.Location = lc.String()
	}

	for k, v := range resp.Header {
		// Ignore Cookie Headers
		if k == "Set-Cookie" || k == "Location" {
			continue
		} else {
			r.Headers[k] = strings.Join(v, " ")
		}

		if k == "Content-Type" {
			r.ContentType = strings.Join(v, " ")
			r.ContentType = strings.ToLower(r.ContentType)
		}

		// If content-encoding is gzip
		if k == "Content-Encoding" && StoreResponseBody {

			val := strings.Join(v, " ")

			// Only Supports gzip for now
			if strings.Contains(val, "gzip") {
				rdr, gerr := gzip.NewReader(bytes.NewReader(r.Body))
				if gerr == nil {
					// if successfully decoded
					bin, derr := ioutil.ReadAll(rdr)
					if derr == nil {
						r.Body = bin
					}
				}
			}

		}
	}

	// Prettify JSON Body if it is json
	if StoreResponseBody && strings.Contains(r.ContentType, "json") {
		r.Body = PrettyJSON(r.Body)
	}

	r.modifystatuscode()

	return nil
}

func (r *RawHttpResponse) modifystatuscode() {
	if !BlackListContentLength {
		return
	}

	if BlackListContentLenVal != 0 {
		if r.ContentLength == BlackListContentLenVal {
			r.StatusCode = 404
		}
	} else if BlackListContentLenValMin != 0 && BlackListContentLenValMax != 0 {
		if BlackListContentLenValMin <= r.StatusCode && BlackListContentLenValMax >= r.StatusCode {
			r.StatusCode = 404
		}
	}
}

// ParseFromBytes : Parse response from bytes (ex: burp response)
func (r *RawHttpResponse) ParseFromBytes(bin []byte) error {

	//Initialize maps
	r.Cookies = map[string]string{}
	r.Headers = map[string]string{}

	// Remove all \r
	temp := bytes.ReplaceAll(bin, []byte{'\r'}, []byte{})

	// split data at \n\n i.e response headers and body
	data := bytes.SplitN(temp, []byte{'\n', '\n'}, 2)

	if len(data) == 1 {
		// Response does not have any body
		r.ContentLength = 0
		r.Body = []byte{}
	} else {
		r.Body = data[1]
		r.ContentLength = len(data[1])
	}

	// raw contains upper body of raw response
	// which contains all headers,cookies , status code etc
	raw := data[0]

	for k, v := range Split(string(raw), '\n') {
		if k == 0 {
			//First line extract status code
			line := strings.SplitN(v, " ", 3)
			if len(line) != 3 {
				return fmt.Errorf("malformed response received %v", v)
			}
			val, err := strconv.Atoi(line[1])
			if err != nil {
				return fmt.Errorf("failed to parse status code %v", line[1])
			}
			r.StatusCode = val
		} else {
			// All remaining items are headers
			line := strings.SplitN(v, ":", 2)
			if len(line) != 2 {
				//malformed skip this
				continue
			} else {
				key := line[0]
				value := strings.TrimSpace(line[1])

				switch key {
				case "Content-Length":
					continue
				case "Content-Type":
					r.ContentType = value
				case "Location":
					r.Location = value
				case "Set-Cookie":
					//only set cookie and its value
					tarr := strings.Split(value, ";")
					if len(tarr) > 0 {
						d := strings.SplitN(tarr[0], "=", 2)
						if len(d) == 2 {
							cookiename := strings.TrimSpace(d[0])
							r.Cookies[cookiename] = d[1]
						}
					}
				default:
					//treat as header
					r.Headers[key] = value
				}
			}

		}
	}

	if strings.Contains(r.ContentType, "json") && len(r.Body) > 2 {
		r.Body = PrettyJSON(r.Body)
	}

	r.modifystatuscode()

	return nil

}

// PrettyJSON : Format/Indent JSON
func PrettyJSON(bin []byte) []byte {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, bin, "", "	"); err != nil {
		return bin
	}
	return prettyJSON.Bytes()
}
