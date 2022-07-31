package burpsuite

// Items : Root of XML Tree contains all Requests & Responses
type Items struct {
	BurpVersion string `xml:"burpVersion,attr"`
	ExportTime  string `xml:"exportTime,attr"`
	Items       []Item `xml:"item"`
}

// Item :  Each Item contains a request and response
type Item struct {
	Time           string   `xml:"time"`
	URL            string   `xml:"url"`
	Host           Host     `xml:"host"`
	Port           string   `xml:"port"`
	Protocol       string   `xml:"protocol"`
	Method         string   `xml:"method"`
	Path           string   `xml:"path"`
	Request        Request  `xml:"request"`
	Status         int      `xml:"status"`
	ResponseLength int      `xml:"responselength"`
	MimeType       string   `xml:"mimetype"`
	Response       Response `xml:"response"`
	Comment        string   `xml:"comment"`
}

//Request Information
type Request struct {
	IsBase64 bool   `xml:"base64,attr"`
	RawData  string `xml:",chardata"`
}

// Response Information
type Response struct {
	IsBase64 bool   `xml:"base64,attr"`
	RawData  string `xml:",chardata"`
}

//Host information
type Host struct {
	Name string `xml:",chardata"`
	IP   string `xml:"ip,attr"`
}
