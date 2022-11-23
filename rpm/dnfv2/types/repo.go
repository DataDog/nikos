package types

type Repomd struct {
	Data []RepomdData `xml:"data"`
}

type RepomdData struct {
	Type         string   `xml:"type,attr"`
	Size         int      `xml:"size"`
	OpenSize     int      `xml:"open-size"`
	Location     Location `xml:"location"`
	Checksum     Checksum `xml:"checksum"`
	OpenChecksum Checksum `xml:"open-checksum"`
}

type Location struct {
	Href string `xml:"href,attr"`
}

type MetaLink struct {
	Files MetaLinkFiles `xml:"files"`
}

type MetaLinkFiles struct {
	Files []MetaLinkFile `xml:"file"`
}

type MetaLinkFile struct {
	Name      string                `xml:"name,attr"`
	Resources MetaLinkFileResources `xml:"resources"`
}

type MetaLinkFileResources struct {
	Urls []MetaLinkFileResourceURL `xml:"url"`
}

type MetaLinkFileResourceURL struct {
	Protocol   string `xml:"protocol,attr"`
	Preference int    `xml:"preference,attr"`
	URL        string `xml:",chardata"`
}
