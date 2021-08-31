module github.com/DataDog/nikos

go 1.14

require (
	cloud.google.com/go/storage v1.12.0
	github.com/AlekSi/pointer v1.1.0 // indirect
	github.com/DisposaBoy/JsonConfigReader v0.0.0-20171218180944-5ea4d0ddac55 // indirect
	github.com/aptly-dev/aptly v1.4.0
	github.com/arduino/go-apt-client v0.0.0-20190812130613-5613f843fdc8
	github.com/awalterschulze/gographviz v2.0.1+incompatible // indirect
	github.com/cheggaaa/pb v1.0.29 // indirect
	github.com/cobaugh/osrelease v0.0.0-20181218015638-a93a0a55a249
	github.com/h2non/filetype v1.1.0 // indirect
	github.com/jlaffaye/ftp v0.0.0-20200812143550-39e3779af0db // indirect
	github.com/mattn/go-isatty v0.0.12 // indirect
	github.com/mattn/go-runewidth v0.0.9 // indirect
	github.com/pborman/uuid v1.2.1 // indirect
	github.com/sassoftware/go-rpmutils v0.1.1
	github.com/sirupsen/logrus v1.7.0
	github.com/spf13/cobra v1.0.0
	github.com/ugorji/go v1.1.7 // indirect
	github.com/wille/osutil v0.0.0-20200805111424-0dc696c283a2
	github.com/xi2/xz v0.0.0-20171230120015-48954b6210f8
	github.com/xor-gate/ar v0.0.0-20170530204233-5c72ae81e2b7
	golang.org/x/sys v0.0.0-20200905004654-be1d3432aa8f
	google.golang.org/api v0.32.0
	gopkg.in/yaml.v2 v2.2.8 // indirect
)

replace github.com/aptly-dev/aptly => github.com/lebauce/aptly v0.7.2-0.20210723103859-345a32860f4d

replace github.com/wille/osutil => github.com/lebauce/osutil v0.0.0-20201027170515-5409e8e42a87
