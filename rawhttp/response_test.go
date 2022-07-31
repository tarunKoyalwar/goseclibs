package rawhttp_test

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"testing"

	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

func Test_raw_response(t *testing.T) {

	go responsetestserver(t)

	resp, err := http.Get("http://localhost:8998/")

	if err != nil {
		t.Errorf("Network Failure %v", err)
	}

	r := rawhttp.RawHttpResponse{}
	err2 := r.Parse(resp)
	if err2 != nil {
		t.Errorf("Failed to parse response %v", err2)
	}

	if r.StatusCode == 0 {
		t.Logf("Something Went Wrong Status Code is 0")
	}

}

func responsetestserver(t *testing.T) {

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "200 OK response from localhost")
	})

	t.Logf("%v", http.ListenAndServe(":8998", nil))

}

func Test_Response_From_Bytes(t *testing.T) {

	go func() {

		mux := http.NewServeMux()

		mux.HandleFunc("/503", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "2")
			http.Error(w, fmt.Sprintf("test_%d_body", 503), http.StatusServiceUnavailable)
		})

		log.Fatalf("%v", http.ListenAndServe(":8095", mux))
	}()

	resp, err := http.Get("http://localhost:8095/")

	if err != nil {
		t.Errorf("Network Failure %v", err)
	}

	bin, _ := httputil.DumpResponse(resp, true)

	r, err2 := rawhttp.NewRawHttpResponseFromBytes(bin)

	if err2 != nil {
		t.Errorf("Failed to parse response %v", err2)
	}

	if r.StatusCode == 0 {
		t.Logf("Something Went Wrong Status Code is 0")
	}
}
