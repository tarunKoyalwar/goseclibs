package burpsuite

import (
	"encoding/xml"
	"io/ioutil"
)

/*
Extracting and using
HTTP Requests & Responses From BurpSuite Exported File

TO Export Any Branch/ Request
1. Right Click -> Save Selected Items

Apart From parsing  normal XML
Data can be parsed into rawhttprequest or response

*/

// ParseSavedItemsXML : Parse Saved Items Burpsuite XML data
func ParseSavedItemsXML(xmldata []byte) (*Items, error) {

	var p Items

	err := xml.Unmarshal(xmldata, &p)
	if err != nil {
		return &p, err
	}

	return &p, nil
}

// ParseSavedItemsFile : Parse Saved Items Burpsuite XML File
func ParseSavedItemsFile(xmlfile string) (*Items, error) {
	bin, err1 := ioutil.ReadFile(xmlfile)
	if err1 != nil {
		return nil, nil
	}

	return ParseSavedItemsXML(bin)
}
