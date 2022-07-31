package rawhttp_test

import (
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

func Test_SimpleClient(t *testing.T) {
	go func() {
		http.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusTemporaryRedirect)
			w.Header().Add("Location", "https://github.com")
		})

		log.Fatalf("%v", http.ListenAndServe(":8994", nil))
	}()

	// Test if Client acutally works
	c := rawhttp.SHTTPClient{
		FollowRedirect: false,
		DialTimeout:    3,
		TotalTimeout:   10,
	}

	c.Create()

	req, _ := http.NewRequest("GET", "http://localhost:8994/redirect", nil)

	resp, err := c.Do(req)
	if err != nil {
		t.Errorf("Error %v", err)
	}

	t.Logf("status code is %v\n", resp.StatusCode)

	if resp.StatusCode < 300 && resp.StatusCode >= 400 {
		t.Errorf("Something went wrong\nredirection failure got status code %v", resp.StatusCode)
	}
}

func Test_RetryAfter(t *testing.T) {

	go func() {

		http.HandleFunc("/503", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Retry-After", "2")
			http.Error(w, fmt.Sprintf("test_%d_body", 503), http.StatusServiceUnavailable)
		})

		log.Fatalf("%v", http.ListenAndServe(":8995", nil))
	}()

	c := rawhttp.SHTTPClient{
		RetryCount: 3,
	}

	c.Create()

	req, _ := http.NewRequest("GET", "http://localhost:8995/503", nil)

	then := time.Now()

	resp, err := c.Do(req)

	now := time.Now()

	diff := now.Sub(then)

	if diff.Seconds() < 5 || diff.Seconds() > 10 {
		t.Errorf("Something must have gone wrong time took was %v expected approx 6-8 sec", diff)
	} else {
		t.Logf("Request took approx %v\n to complete after  3 retries", diff.Seconds())
	}

	if err != nil {
		t.Errorf("this should not happend got error\n%v", err)
	}

	if resp.StatusCode != 503 {
		t.Errorf("status code is not 503 its %v", resp.StatusCode)
	}

}

func Test_UnavailableServer(t *testing.T) {

	// if the server doesnot exist
	c := rawhttp.SHTTPClient{
		DialTimeout: 3,
	}

	c.Create()

	req, _ := http.NewRequest("GET", "http://non.existing.com:8888", nil)

	then := time.Now()

	c.Do(req)

	now := time.Now()

	diff := now.Sub(then)

	if diff.Seconds() > 8 {
		t.Errorf("This should not be the case took %v seconds", diff.Seconds())
	} else {
		t.Logf("Total Time taken %v using dial timeout", diff.Seconds())
	}

}
