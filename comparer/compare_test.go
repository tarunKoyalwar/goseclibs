package comparer_test

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"testing"

	"github.com/tarunKoyalwar/goseclibs/comparer"
	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

func Test_DualResponseComparer(t *testing.T) {
	// Compare Two responses

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

		log.Fatal(http.ListenAndServe("localhost:8989", mux))
	}()

	client := rawhttp.SHTTPClient{FollowRedirect: false}
	client.Create()

	resp1, err1 := client.Get("http://localhost:8989/")
	HandleError(t, err1)

	response1, _ := rawhttp.NewRawHttpResponse(resp1)

	resp2, err2 := client.Get("http://localhost:8989/redirect")
	HandleError(t, err2)

	response2, _ := rawhttp.NewRawHttpResponse(resp2)
	t.Logf("[Info] Content Length of redirect request is %v\n", response2.ContentLength)

	resp3, err3 := client.Get("http://localhost:8989/success")
	HandleError(t, err3)

	response3, _ := rawhttp.NewRawHttpResponse(resp3)
	t.Logf("[Info] Content Length of success request is %v\n", response3.ContentLength)
	t.Logf("[Info] Cookies of success request are %v\n", response3.Cookies)

	c1 := comparer.NewDualResponseComparer(response1, response2)

	results1, _ := c1.Compare()

	if len(results1) != 4 {
		t.Errorf("[Error]Must be 4 differences status code,content lenght,new header,location but got %v", results1)
	} else {
		// These are the changes to look for
		statuschange := false
		locationchange := false
		newheader := false
		for _, v := range results1 {
			if v.Type == comparer.StatusCode {
				if v.New != strconv.Itoa(http.StatusFound) {
					t.Errorf("[Error]Must be a 302 response but got %v", v)
				} else {
					t.Logf("[Info]302 response received")
					statuschange = true
				}
			} else if v.Type == comparer.Location {
				if v.New != "https://github.com" {
					t.Errorf("Must be a change in location header %v", v)
				} else {
					t.Logf("[Info]Found Location")
					locationchange = true
				}
			} else {
				if v.Type == comparer.ContentLength {
					t.Logf("[Info]Observed content length change in 302")
				} else if v.Type == comparer.ContentType {
					t.Logf("[Info] Observed New Header Content-Type")
					newheader = true
				} else {
					t.Errorf("[Error]Extra Change Found %v", v)
				}
			}
		}
		if statuschange && locationchange && newheader {
			t.Logf("[Info]Status Code & Location Factors Verified")
		} else {
			t.Errorf("[Error]Status Code & Location Factors Verification Failed")
		}
	}

	c2 := comparer.NewDualResponseComparer(response2, response3)
	results2, _ := c2.Compare()

	if len(results2) != 3 {
		t.Errorf("must be 3 changes location ,cookie, content length change %v", results2)
	} else {
		// These are changes to look for
		cookiechange := false
		locationchange := false
		contentlengthchange := false
		for _, v := range results2 {
			if v.Type == comparer.Cookie {
				t.Logf("[Info] got new cookie %v\n", v.New)
				cookiechange = true

			} else if v.Type == comparer.Location {
				if v.New != "http://localhost:8989/secretpanel" {
					t.Errorf("[Error]Must be a change in location header %v", v)
				} else {
					t.Logf("[Info]Found Location")
					locationchange = true
				}
			} else if v.Type == comparer.ContentLength {
				t.Logf("[Info]got new content length")
				contentlengthchange = true
			} else {
				t.Errorf("[Error] Extra Change Found %v", v)
			}
		}
		if cookiechange && locationchange && contentlengthchange {
			t.Logf("[Info] location ,cookie, content length  Factors Verified")
		} else {
			t.Errorf("[Error] location ,cookie, content length Factors Verification Failed")
		}
	}

}

func HandleError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("got an error while sending request %v", err)
	}
}
