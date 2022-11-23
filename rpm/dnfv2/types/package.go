package types

type Metadata struct {
	Packages []*Package `xml:"package"`
}

type Package struct {
	Type     string   `xml:"type,attr"`
	Name     string   `xml:"name"`
	Arch     string   `xml:"arch"`
	Version  Version  `xml:"version"`
	Checksum Checksum `xml:"checksum"`
	Location Location `xml:"location"`
}

type Version struct {
	Epoch string `xml:"epoch,attr"`
	Ver   string `xml:"ver,attr"`
	Rel   string `xml:"rel,attr"`
}
