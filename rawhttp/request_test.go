package rawhttp_test

import (
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

func Test_raw_request(t *testing.T) {

	z, _ := http.NewRequest("GET", "https://github.com/", nil)

	bin, er := httputil.DumpRequestOut(z, false)

	if er != nil {
		t.Errorf("Failed to dump request %v", er)
	}

	req, er2 := rawhttp.NewRawHttpRequestFromBytes(bin)

	if er2 != nil {
		t.Errorf("failed to parse request %v", er2)
	}

	request := req.GetRequest()

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Errorf("failed to send response %v", err)
	}

	if resp.StatusCode == 400 {
		t.Errorf("Responded with Bad request Something went wrong")
	}

	t.Logf("Got response %v", resp.StatusCode)

}
