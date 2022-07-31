package comparer_test

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"testing"
	"time"

	"github.com/tarunKoyalwar/goseclibs/comparer"
	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

func Test_ManyComparer(t *testing.T) {
	// test server
	go func() {

		mux := http.NewServeMux()

		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "Nothing to compare plain request")
		})

		mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Location", "https://github.com")
			http.Redirect(w, r, "https://github.com", http.StatusFound)
		})

		mux.HandleFunc("/success", func(w http.ResponseWriter, r *http.Request) {
			http.SetCookie(w, &http.Cookie{
				Name:  "access_token",
				Value: "xxxxxblaxxxxblaxxxxbla",
			})
			w.Header().Add("Location", "/secretpanel")
			http.Redirect(w, r, "/secretpanel", http.StatusFound)
			fmt.Fprintf(w, "You are now Admin")
		})

		log.Fatal(http.ListenAndServe("localhost:8900", mux))
	}()

	client := rawhttp.SHTTPClient{FollowRedirect: false}
	client.Create()

	resp1, err1 := client.Get("http://localhost:8900/")
	HandleError(t, err1)

	response1, _ := rawhttp.NewRawHttpResponse(resp1)

	resp2, err2 := client.Get("http://localhost:8900/redirect")
	HandleError(t, err2)

	response2, _ := rawhttp.NewRawHttpResponse(resp2)

	resp3, err3 := client.Get("http://localhost:8900/success")
	HandleError(t, err3)

	response3, _ := rawhttp.NewRawHttpResponse(resp3)

	//total 2 cases
	/*
		1 2
		1 3
	*/

	// Must obtain 2 responses since all are different

	m := comparer.NewOne2ManyResponseComparer(response1, response2, response3)
	m.Ignore = map[comparer.Factor]bool{
		comparer.HeaderValue: true,
	}

	ctx := context.Background()

	then := time.Now()
	res := m.Compare(ctx)
	now := time.Now()

	if len(res) == 2 {
		t.Logf("test successful\n")
		t.Logf("took %v seconds\n", now.Sub(then).Seconds())
	} else {
		t.Errorf("Combinations missed only got %v  responses", len(res))
	}
}
