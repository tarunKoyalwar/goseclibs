package comparer

/*
Factors are nothing but places where changes are observed
Whenever a Factor is Found its details are stored in
Change Struct . Which contains Factor type . It's old value and New value
Factor is enum datatype
*/

type Factor int

const (
	StatusCode    Factor = iota
	ContentLength        // Change in Content Length
	ContentType          // Change in Content Type
	Location             // Change in Location Header(Includes Value)
	Header               // Extra/Missing Header
	HeaderValue          // Header Value is changed
	Cookie               // Extra/Missing Cookie
)

// Change : Change Observed For that particular Factor
type Change struct {
	Type Factor // Type of Factor
	Old  string // Old Value of this Factor
	New  string // New Value of this Factor
}

func FactorString(z Factor) string {
	switch z {
	case StatusCode:
		return "StatusCode"
	case ContentLength:
		return "ContentLength"
	case ContentType:
		return "ContentType"
	case Location:
		return "Location"
	case Header:
		return "Header"
	case HeaderValue:
		return "HeaderValue"
	case Cookie:
		return "Cookie"
	default:
		return "Invalid"
	}
}
