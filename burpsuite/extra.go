package burpsuite

import (
	"encoding/base64"

	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

/*
Unlike Burpsuite Item these are verbose and has many extra methods
*/
type HTTPItem struct {
	DomainName string                   // domain name
	URL        string                   // URL
	Time       string                   // time
	Comment    string                   // BurpSuite Comment
	Request    *rawhttp.RawHttpRequest  //request struct
	Response   *rawhttp.RawHttpResponse // response struct

}

func NewHTTPItem(x Item) HTTPItem {
	h := HTTPItem{
		URL:        x.URL,
		Time:       x.Time,
		Comment:    x.Comment,
		DomainName: x.Host.Name,
	}

	var reqbytes []byte

	if x.Request.IsBase64 {
		reqbytes, _ = base64.StdEncoding.DecodeString(x.Request.RawData)
	} else {
		reqbytes = []byte(x.Request.RawData)
	}

	req, _ := rawhttp.NewRawHttpRequestFromBytes(reqbytes)

	h.Request = req

	var respbytes []byte

	if x.Response.IsBase64 {
		respbytes, _ = base64.RawStdEncoding.DecodeString(x.Response.RawData)
	} else {
		respbytes = []byte(x.Response.RawData)
	}

	resp, _ := rawhttp.NewRawHttpResponseFromBytes(respbytes)

	h.Response = resp

	return h

}
