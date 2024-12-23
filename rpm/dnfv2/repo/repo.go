package repo

import (
	"bufio"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ProtonMail/go-crypto/openpgp"
	"gopkg.in/ini.v1"

	"github.com/DataDog/nikos/rpm/dnfv2/internal/utils"
	"github.com/DataDog/nikos/rpm/dnfv2/types"
	"github.com/DataDog/nikos/xmlite"
	"github.com/hashicorp/go-multierror"
	"github.com/sassoftware/go-rpmutils"
)

type Repo struct {
	SectionName   string
	Name          string
	BaseURL       string
	MirrorList    string
	MetaLink      string
	Type          string
	Enabled       bool
	GpgCheck      bool
	GpgKeys       []string
	SSLVerify     bool
	SSLClientKey  string
	SSLClientCert string
	SSLCaCert     string
}

func ReadFromDir(repoDir string) ([]Repo, error) {
	repoFiles, err := filepath.Glob(utils.HostEtcJoin(repoDir, "*.repo"))
	if err != nil {
		return nil, err
	}

	repos := make([]Repo, 0)
	for _, repoFile := range repoFiles {
		cfg, err := ini.Load(repoFile)
		if err != nil {
			return nil, err
		}

		for _, section := range cfg.Sections() {
			if section.Name() == "DEFAULT" {
				continue
			}

			repo := Repo{}
			repo.SectionName = section.Name()
			repo.Name = section.Key("name").String()
			repo.BaseURL = section.Key("baseurl").String()
			repo.MirrorList = section.Key("mirrorlist").String()
			repo.MetaLink = section.Key("metalink").String()
			repo.Type = section.Key("type").String()
			repo.Enabled = section.Key("enabled").MustBool()
			repo.GpgCheck = section.Key("gpgcheck").MustBool()
			repo.GpgKeys = strings.Split(section.Key("gpgkey").String(), ",")
			repo.SSLVerify = section.Key("sslverify").MustBool(true)
			repo.SSLClientKey = section.Key("sslclientkey").String()
			repo.SSLClientCert = section.Key("sslclientcert").String()
			repo.SSLCaCert = section.Key("sslcacert").String()

			// hack for yast2 repo support
			if repo.Type == "yast2" && repo.BaseURL != "" {
				repo.BaseURL += "suse/"
			}

			repos = append(repos, repo)
		}
	}
	return repos, nil
}

type PkgInfo struct {
	Header   PkgInfoHeader
	Location string
	Checksum *types.Checksum
}

type PkgInfoHeader struct {
	Name string
	types.Version
	Arch string
}

type PkgMatchFunc = func(*PkgInfoHeader) bool

func (r *Repo) createHTTPClient() (*utils.HttpClient, error) {
	var certs []tls.Certificate
	if r.SSLClientCert != "" || r.SSLClientKey != "" {
		cert, err := tls.LoadX509KeyPair(utils.HostEtcJoin(r.SSLClientCert), utils.HostEtcJoin(r.SSLClientKey))
		if err != nil {
			return nil, fmt.Errorf("failed to load SSL certificate: %w", err)
		}
		certs = append(certs, cert)
	}

	var certPool *x509.CertPool
	if r.SSLCaCert != "" {
		certPool = x509.NewCertPool()
		customPem, err := os.ReadFile(utils.HostEtcJoin(r.SSLCaCert))
		if err != nil {
			return nil, fmt.Errorf("failed to read custom CA cert")
		}
		if !certPool.AppendCertsFromPEM(customPem) {
			return nil, fmt.Errorf("failed to add custom CA cert to cert pool")
		}
	}

	inner := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: !r.SSLVerify,
				Certificates:       certs,
				RootCAs:            certPool,
			},
		},
	}

	return utils.NewHttpClientFromInner(inner), nil
}

func (r *Repo) FetchPackage(ctx context.Context, pkgMatcher PkgMatchFunc) (*PkgInfo, []byte, error) {
	httpClient, err := r.createHTTPClient()
	if err != nil {
		return nil, nil, err
	}

	repoMd, err := r.FetchRepoMD(ctx, httpClient)
	if err != nil {
		return nil, nil, err
	}

	fetchURL, err := r.FetchURL(ctx, httpClient)
	if err != nil {
		return nil, nil, err
	}

	var entityList openpgp.EntityList
	if r.GpgCheck {
		el, err := readGPGKeys(ctx, httpClient, r.GpgKeys)
		// if we found keys we can ignore the error
		if err != nil && len(el) == 0 {
			return nil, nil, fmt.Errorf("failed to read gpg key: %w", err)
		}
		entityList = el
	}

	pkgInfo, err := r.FetchPackageFromList(ctx, httpClient, repoMd, pkgMatcher)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find valid package from repo %s: %w", r.Name, err)
	}

	pkgUrl, err := utils.UrlJoinPath(fetchURL, pkgInfo.Location)
	if err != nil {
		return nil, nil, err
	}

	pkgRpm, err := httpClient.GetWithChecksum(ctx, pkgUrl, pkgInfo.Checksum)
	if err != nil {
		return nil, nil, err
	}

	if r.GpgCheck {
		rpmReader, err := pkgRpm.Reader()
		defer rpmReader.Close()

		if err != nil {
			return nil, nil, err
		}
		_, _, err = rpmutils.Verify(rpmReader, entityList)
		if err != nil {
			return nil, nil, err
		}
	}

	pkgRpmData, err := pkgRpm.Data()
	return pkgInfo, pkgRpmData, err
}

func readGPGKeys(ctx context.Context, httpClient *utils.HttpClient, gpgKeys []string) (openpgp.EntityList, *multierror.Error) {
	visited := make(map[string]bool, len(gpgKeys))

	var entities openpgp.EntityList
	var errors *multierror.Error

	for _, gpgKey := range gpgKeys {
		if visited[gpgKey] {
			// this key is already loaded
			continue
		}
		visited[gpgKey] = true

		gpgKeyUrl, err := url.Parse(gpgKey)
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}

		var publicKeyReader io.Reader
		if gpgKeyUrl.Scheme == "file" {
			publicKeyFile, err := os.Open(utils.HostEtcJoin(gpgKeyUrl.Path))
			if err != nil {
				errors = multierror.Append(errors, err)
				continue
			}
			defer publicKeyFile.Close()
			publicKeyReader = publicKeyFile
		} else if gpgKeyUrl.Scheme == "http" || gpgKeyUrl.Scheme == "https" {
			content, err := httpClient.Get(ctx, gpgKey)
			if err != nil {
				errors = multierror.Append(errors, err)
				continue
			}
			publicKeyDataReader, err := content.Reader()
			if err != nil {
				errors = multierror.Append(errors, err)
				continue
			}
			defer publicKeyDataReader.Close()
			publicKeyReader = publicKeyDataReader
		} else {
			err := fmt.Errorf("only file and http(s) scheme are supported for gpg key: %s", gpgKey)
			errors = multierror.Append(errors, err)
			continue
		}

		newEntities, err := openpgp.ReadArmoredKeyRing(publicKeyReader)
		if err != nil {
			errors = multierror.Append(errors, err)
			continue
		}
		entities = append(entities, newEntities...)
	}
	return entities, errors
}

const repomdSubpath = "repodata/repomd.xml"

func (r *Repo) FetchRepoMD(ctx context.Context, httpClient *utils.HttpClient) (*types.Repomd, error) {
	fetchURL, err := r.FetchURL(ctx, httpClient)
	if err != nil {
		return nil, err
	}

	repoMDUrl := fetchURL
	if !utils.UrlHasSuffix(repoMDUrl, "repomd.xml") {
		withFile, err := utils.UrlJoinPath(fetchURL, repomdSubpath)
		if err != nil {
			return nil, err
		}
		repoMDUrl = withFile
	}

	repoMd, err := utils.GetAndUnmarshalXML[types.Repomd](ctx, httpClient, repoMDUrl, nil)
	if err != nil {
		return nil, err
	}

	return repoMd, nil
}

func (r *Repo) FetchURL(ctx context.Context, httpClient *utils.HttpClient) (string, error) {
	if r.BaseURL != "" {
		return r.BaseURL, nil
	}

	if r.MirrorList != "" {
		baseURL, err := fetchURLFromMirrorList(ctx, httpClient, r.MirrorList)
		if err != nil {
			return "", err
		}
		r.BaseURL = baseURL
		return r.BaseURL, nil
	}

	if r.MetaLink != "" {
		url, err := fetchURLFromMetaLink(ctx, httpClient, r.MetaLink)
		if err != nil {
			return "", err
		}
		r.BaseURL = url
		return r.BaseURL, nil
	}

	return "", fmt.Errorf("unable to get a base URL for this repo `%s`", r.Name)
}

func fetchURLFromMirrorList(ctx context.Context, httpClient *utils.HttpClient, mirrorListURL string) (string, error) {
	mirrorList, err := httpClient.Get(ctx, mirrorListURL)
	if err != nil {
		return "", err
	}

	mirrorListReader, err := mirrorList.Reader()
	if err != nil {
		return "", err
	}

	mirrors := make([]string, 0)
	sc := bufio.NewScanner(mirrorListReader)
	for sc.Scan() {
		if sc.Err() != nil {
			return "", err
		}

		line := strings.TrimSpace(sc.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}

		mirrors = append(mirrors, sc.Text())
	}

	if len(mirrors) == 0 {
		return "", fmt.Errorf("no mirror available")
	}

	return mirrors[0], nil
}

func fetchURLFromMetaLink(ctx context.Context, httpClient *utils.HttpClient, metaLinkURL string) (string, error) {
	metalink, err := utils.GetAndUnmarshalXML[types.MetaLink](ctx, httpClient, metaLinkURL, nil)
	if err != nil {
		return "", err
	}

	for _, file := range metalink.Files.Files {
		if file.Name == "repomd.xml" {
			urls := make([]types.MetaLinkFileResourceURL, 0, len(file.Resources.Urls))
			for _, resUrl := range file.Resources.Urls {
				if resUrl.Protocol == "http" || resUrl.Protocol == "https" {
					urls = append(urls, resUrl)
				}
			}

			if len(urls) == 0 {
				return "", errors.New("no url for `repomd.xml` resource")
			}

			sort.Slice(urls, func(i, j int) bool {
				return urls[j].Preference < urls[i].Preference
			})

			repomdUrl := strings.TrimSuffix(urls[0].URL, repomdSubpath)
			return repomdUrl, nil
		}
	}

	return "", fmt.Errorf("failed to fetch base URL from meta link: %s", metaLinkURL)
}

func (r *Repo) FetchPackageFromList(ctx context.Context, httpClient *utils.HttpClient, repoMd *types.Repomd, pkgMatcher PkgMatchFunc) (*PkgInfo, error) {
	fetchURL, err := r.FetchURL(ctx, httpClient)
	if err != nil {
		return nil, err
	}

	for _, d := range repoMd.Data {
		if d.Type == "primary" {
			primaryURL, err := utils.UrlJoinPath(fetchURL, d.Location.Href)
			if err != nil {
				return nil, err
			}

			primaryContent, err := httpClient.GetWithChecksum(ctx, primaryURL, &d.OpenChecksum)
			if err != nil {
				return nil, err
			}

			var pkgInfo *PkgInfo
			for _, path := range []xmlPkgPath{fastPath, slowPath} {
				pkgInfo, err = func(path xmlPkgPath) (*PkgInfo, error) {
					primaryContentReader, err := primaryContent.Reader()
					if err != nil {
						return nil, err
					}
					defer primaryContentReader.Close()

					return path(primaryContentReader, pkgMatcher)
				}(path)

				if err != nil {
					continue
				}
				if pkgInfo != nil {
					return pkgInfo, nil
				}

				// if we found nothing but no error we don't run the slow path
				break
			}

			// if the slow path returns an error we fire it
			if err != nil {
				return nil, err
			}
		}
	}

	return nil, errors.New("no matching package found")
}

type xmlPkgPath = func(io.Reader, PkgMatchFunc) (*PkgInfo, error)

func fastPath(reader io.Reader, pkgMatcher PkgMatchFunc) (*PkgInfo, error) {
	handler := &PkgHandler{
		matcher: pkgMatcher,
	}
	decoder := xmlite.NewLiteDecoder(reader, handler)
	if err := decoder.Parse(); err != nil {
		return nil, err
	}

	return handler.winner, nil
}

type parseState int

const (
	Start parseState = iota
	InPackage
	InArch
	InLocation
	InFormat
	InProvides
	InEntry
	InChecksum
)

type PkgHandler struct {
	err     error
	matcher PkgMatchFunc
	winner  *PkgInfo
	state   parseState
	current *TempPkgInfo
}

type TempPkgInfo struct {
	arch      string
	location  string
	checksum  *types.Checksum
	currEntry *TempProvides
}

type TempProvides struct {
	name  string
	epoch string
	ver   string
	rel   string
}

func (ph *PkgHandler) StartTag(name []byte) {
	switch string(name) {
	case "package":
		ph.state = InPackage
		ph.current = &TempPkgInfo{}
	case "arch":
		if ph.state == InPackage {
			ph.state = InArch
		}
	case "location":
		if ph.state == InPackage {
			ph.state = InLocation
		}
	case "checksum":
		if ph.state == InPackage {
			ph.state = InChecksum
			if ph.current != nil {
				ph.current.checksum = &types.Checksum{}
			}
		}
	case "format":
		if ph.state == InPackage {
			ph.state = InFormat
		}
	case "rpm:provides":
		if ph.state == InFormat {
			ph.state = InProvides
		}
	case "rpm:entry":
		if ph.state == InProvides {
			ph.state = InEntry
			if ph.current != nil {
				ph.current.currEntry = &TempProvides{}
			}
		}
	}
}

func (ph *PkgHandler) EndTag(name []byte) {
	switch string(name) {
	case "package":
		ph.state = Start
		ph.current = nil
	case "arch":
		if ph.state == InArch {
			ph.state = InPackage
		}
	case "location":
		if ph.state == InLocation {
			ph.state = InPackage
		}
	case "checksum":
		if ph.state == InChecksum {
			ph.state = InPackage
		}
	case "format":
		if ph.state == InFormat {
			ph.state = InPackage
		}
	case "rpm:provides":
		if ph.state == InProvides {
			ph.state = InFormat
		}
	case "rpm:entry":
		if ph.state == InEntry {
			ph.state = InProvides
			if ph.current != nil && ph.current.currEntry != nil && !strings.Contains(ph.current.currEntry.name, "(") && ph.matcher != nil && ph.winner == nil {
				if ph.current.arch == "" {
					ph.err = errors.New("arch declared after entry, fast path impossible")
				}

				pkgInfo := &PkgInfo{
					Header: PkgInfoHeader{
						Name: ph.current.currEntry.name,
						Version: types.Version{
							Epoch: ph.current.currEntry.epoch,
							Ver:   ph.current.currEntry.ver,
							Rel:   ph.current.currEntry.rel,
						},
						Arch: ph.current.arch,
					},
					Location: ph.current.location,
					Checksum: ph.current.checksum,
				}

				if ph.matcher(&pkgInfo.Header) {
					ph.winner = pkgInfo
				}

				ph.current.currEntry = nil
			}
		}
	}
}

func (ph *PkgHandler) Attr(name, value []byte) {
	if ph.current == nil {
		return
	}

	if ph.state == InLocation && string(name) == "href" {
		ph.current.location = string(value)
	} else if ph.state == InEntry && ph.current.currEntry != nil {
		switch string(name) {
		case "name":
			ph.current.currEntry.name = string(value)
		case "epoch":
			ph.current.currEntry.epoch = string(value)
		case "ver":
			ph.current.currEntry.ver = string(value)
		case "rel":
			ph.current.currEntry.rel = string(value)
		}
	} else if ph.state == InChecksum && string(name) == "type" {
		if ph.current.checksum != nil {
			ph.current.checksum.Type = string(value)
		}
	}
}

func (ph *PkgHandler) CharData(value []byte) {
	if ph.current == nil {
		return
	}

	switch ph.state {
	case InArch:
		ph.current.arch = string(value)
	case InChecksum:
		if ph.current.checksum != nil {
			ph.current.checksum.Hash = string(value)
		}
	}
}

func slowPath(reader io.Reader, pkgMatcher PkgMatchFunc) (*PkgInfo, error) {
	d := xml.NewDecoder(reader)
	for {
		tok, err := d.Token()
		if tok == nil || err == io.EOF {
			break
		} else if err != nil {
			return nil, err
		}

		switch ty := tok.(type) {
		case xml.StartElement:
			if ty.Name.Local == "package" {
				var pkg types.Package
				if err = d.DecodeElement(&pkg, &ty); err != nil {
					return nil, err
				}

				for _, provides := range pkg.Provides {

					pkgInfo := &PkgInfo{
						Header: PkgInfoHeader{
							Name:    provides.Name,
							Version: provides.Version,
							Arch:    pkg.Arch,
						},
						Location: pkg.Location.Href,
						Checksum: &pkg.Checksum,
					}

					if pkgMatcher(&pkgInfo.Header) {
						return pkgInfo, nil
					}
				}
			}
		default:
		}
	}

	return nil, nil
}
