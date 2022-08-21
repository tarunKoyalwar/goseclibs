package burpsuite

/*
Load Items From Burpsuite Schema and
Structure them into schema similar to sitemap tree
*/
type SiteMap struct {
	AllItems []HTTPItem //contains request and responses
}

func (s *SiteMap) Parse(z Items) {
	if s.AllItems == nil {
		s.AllItems = []HTTPItem{}
	}

	for _, v := range z.Items {
		h := NewHTTPItem(v)
		s.AllItems = append(s.AllItems, h)
	}
}

func NewSiteMap(burp Items) *SiteMap {
	x := SiteMap{}

	x.Parse(burp)

	return &x
}
