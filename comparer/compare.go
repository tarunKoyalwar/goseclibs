package comparer

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/tarunKoyalwar/goseclibs/rawhttp"
)

/*

Compare two responses
with given settings
including factors like
cookies
header values
and other commons like statuscode ,bodylength

*/

// Exclusions For any given type of Factor
// StatusCode, ContentType, ContentLength does not account for
// exclusions as it doesn't make any sense
// exclusions with empty arr will skip the factor entirely
var Exclusions map[Factor][]string = map[Factor][]string{
	Header: {"date"},
}

// DualResponseComparer : Compare any two responses and get changes
type DualResponseComparer struct {
	Old    *rawhttp.RawHttpResponse
	New    *rawhttp.RawHttpResponse
	Ignore map[Factor]bool /* These Factors are Ignored and are not calculated
	By default all Factors are considered except HeaderValue*/
}

// Compare : This Function return Changes (Empty array is returned if there are no differences/changes)
func (d *DualResponseComparer) Compare() ([]Change, error) {
	changes := []Change{}

	AllFactors := []Factor{StatusCode, ContentLength, ContentType, Header, HeaderValue, Cookie, Location}

	if d.Old == nil || d.New == nil {
		return changes, fmt.Errorf("missing Responses to Compare")
	}

	for _, v := range AllFactors {

		if d.Ignore[v] {
			// This Factor Is Ignored
			continue
		}

		w, ok := Exclusions[v]
		if ok {
			if w == nil || len(w) == 0 {
				continue
			}
		}

		switch v {
		case StatusCode:
			// Check For Changes in Status Code
			if d.Old.StatusCode != d.New.StatusCode {
				sc := Change{
					Type: StatusCode,
					Old:  strconv.Itoa(d.Old.StatusCode),
					New:  strconv.Itoa(d.New.StatusCode),
				}
				changes = append(changes, sc)
			}

		case ContentLength:
			// Check For Changes in ContentLength
			if d.Old.ContentLength != d.New.ContentLength {
				clc := Change{
					Type: ContentLength,
					Old:  strconv.Itoa(d.Old.ContentLength),
					New:  strconv.Itoa(d.New.ContentLength),
				}
				changes = append(changes, clc)
			}

		case ContentType:
			// Check For Changes in ContentType
			c := strings.Compare(d.Old.ContentType, d.New.ContentType)
			if c != 0 {
				ctc := Change{
					Type: ContentType,
					Old:  d.Old.ContentType,
					New:  d.New.ContentType,
				}
				changes = append(changes, ctc)
			}

		case Location:
			// Check if there is a change in location header and its value
			cmp := strings.Compare(d.Old.Location, d.New.Location)
			if cmp != 0 {
				ctc := Change{
					Type: Location,
					Old:  d.Old.Location,
					New:  d.New.Location,
				}
				changes = append(changes, ctc)
			}

		case Header:
			hc := d.compareHeaders()
			if hc != nil {
				// Found Something
				changes = append(changes, *hc)
			}

		case HeaderValue:
			hvc := d.compareHeaderValues()
			if hvc != nil {
				//found something
				changes = append(changes, *hvc)
			}

		case Cookie:
			cc := d.compareCookies()
			if cc != nil {
				changes = append(changes, *cc)
			}

		}

	}

	// After Comparison of All Factors

	return changes, nil

}

func (d *DualResponseComparer) compareHeaders() *Change {
	excluded := map[string]bool{}
	if w, ok := Exclusions[Header]; ok {
		for _, v := range w {
			excluded[v] = true
		}
	}
	// Change If any header was added /removed
	// Unique map of old headers
	oldunique := map[string]bool{}
	for k, _ := range d.Old.Headers {
		// check if this header is excluded
		if w, ok := excluded[k]; ok && w {
		} else {
			oldunique[k] = true
		}
	}
	found := ""

	newunique := map[string]bool{}
	for k, _ := range d.New.Headers {
		// check if this header is excluded
		if w, ok := excluded[k]; ok && w {
		} else {
			newunique[k] = true
			// Check if any new request contains any additional value
			if w, ok := oldunique[k]; !ok && !w {
				found += k + " // Extra Header\n"
			}
		}
	}
	//Check if any headers are missing
	for k, _ := range oldunique {
		// Check if any new request contains any additional value
		if w, ok := newunique[k]; !ok && !w {
			found += k + " // Missing Header\n"
		}
	}

	if found != "" {
		// Changes found
		hc := Change{
			Type: Header,
			Old:  "", // Doesnot make sense
			New:  found,
		}
		return &hc
	}

	return nil
}

func (d *DualResponseComparer) compareHeaderValues() *Change {
	// This comparison is unnecessary and only required in
	// rare cases and is blacklisted by default

	// Method of comparison is same only
	// values here are in `header:value` format
	excluded := map[string]bool{}
	if w, ok := Exclusions[HeaderValue]; ok {
		for _, v := range w {
			excluded[v] = true
		}
	}
	// Change If any header was added /removed
	// Unique map of old headers
	oldunique := map[string]bool{}
	for k, v := range d.Old.Headers {
		// check if this header is excluded
		if w, ok := excluded[k]; ok && w {
		} else {
			entry := k + ":" + strings.TrimSpace(v)
			oldunique[entry] = true
		}
	}
	found := ""

	newunique := map[string]bool{}
	for k, v := range d.New.Headers {
		// check if this header is excluded
		if w, ok := excluded[k]; ok && w {
		} else {
			entry := k + ":" + strings.TrimSpace(v)
			newunique[entry] = true
			// Check if any new request contains any additional value
			if w, ok := oldunique[entry]; !ok && !w {
				found += entry + " // Change In Header:Value\n"
			}
		}
	}
	//Check if any headers are missing
	for k, _ := range oldunique {
		// Check if any new request contains any additional value
		if w, ok := newunique[k]; !ok && !w {
			found += k + " // Change In Header:Value\n"
		}
	}

	if found != "" {
		// Changes found
		hc := Change{
			Type: HeaderValue,
			Old:  "", // Doesnot make sense
			New:  found,
		}
		return &hc
	}

	return nil
}

func (d *DualResponseComparer) compareCookies() *Change {
	// If any changes in cookie is observed
	// i.e New cookies are set in response
	excluded := map[string]bool{}
	if w, ok := Exclusions[Cookie]; ok {
		for _, v := range w {
			excluded[v] = true
		}
	}
	// Change If any cookie was added /removed
	// Unique map of old Cookies
	oldunique := map[string]bool{}
	for k, _ := range d.Old.Cookies {
		// check if this cookie is excluded
		if w, ok := excluded[k]; ok && w {
		} else {
			oldunique[k] = true
		}
	}
	found := ""

	newunique := map[string]bool{}
	for k, _ := range d.New.Cookies {
		// check if this cookie is excluded
		if w, ok := excluded[k]; ok && w {
		} else {
			newunique[k] = true
			// Check if any new request contains any additional value
			if w, ok := oldunique[k]; !ok && !w {
				found += k + " // Extra Cookie\n"
			}
		}
	}
	//Check if any Cookies are missing
	for k, _ := range oldunique {
		// Check if any new request contains any additional value
		if w, ok := newunique[k]; !ok && !w {
			found += k + " // Missing Cookie\n"
		}
	}

	if found != "" {
		// Changes found
		hc := Change{
			Type: Cookie,
			Old:  "", // Doesnot make sense
			New:  found,
		}
		return &hc
	}
	return nil
}

func NewDualResponseComparer(old *rawhttp.RawHttpResponse, new *rawhttp.RawHttpResponse) *DualResponseComparer {
	return &DualResponseComparer{
		Old:    old,
		New:    new,
		Ignore: map[Factor]bool{HeaderValue: true},
	}
}
