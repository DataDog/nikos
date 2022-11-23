package types

type Checksum struct {
	Hash string `xml:",chardata"`
	Type string `xml:"type,attr"`
}
