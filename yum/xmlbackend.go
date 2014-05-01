package yum

import (
	"compress/gzip"
	"encoding/xml"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gonuts/logger"
)

// RepositoryXMLBackend is a Backend querying YUM XML repositories
type RepositoryXMLBackend struct {
	Name       string
	Packages   map[string][]*Package
	Provides   map[string][]*Provides
	DBName     string
	Primary    string
	Repository *Repository
	msg        *logger.Logger
}

func NewRepositoryXMLBackend(repo *Repository) (Backend, error) {
	const dbname = "primary.xml.gz"
	return &RepositoryXMLBackend{
		Name:       "RepositoryXMLBackend",
		Packages:   make(map[string][]*Package),
		Provides:   make(map[string][]*Provides),
		DBName:     dbname,
		Primary:    filepath.Join(repo.CacheDir, dbname),
		Repository: repo,
		msg:        repo.msg,
	}, nil
}

// YumDataType returns the ID for the data type as used in the repomd.xml file
func (repo *RepositoryXMLBackend) YumDataType() string {
	return "primary"
}

// Download the DB from server
func (repo *RepositoryXMLBackend) GetLatestDB(url string) error {
	var err error
	out, err := os.Create(repo.Primary)
	if err != nil {
		return err
	}
	defer out.Close()

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, err = io.Copy(out, resp.Body)
	return err
}

// Check whether the DB is there
func (repo *RepositoryXMLBackend) HasDB() bool {
	return path_exists(repo.Primary)
}

// Load loads the DB
func (repo *RepositoryXMLBackend) LoadDB() error {
	var err error

	repo.msg.Infof("start parsing metadata XML file... (%s)\n", repo.Primary)
	type xmlTree struct {
		XMLName  xml.Name `xml:"metadata"`
		Packages []struct {
			Type string `xml:"type,attr"`
			Name string `xml:"name"`
			Arch string `xml:"arch"`

			Version struct {
				Epoch   int    `xml:"epoch,attr"`
				Version string `xml:"ver,attr"`
				Release int    `xml:"rel,attr"`
			} `xml:"version"`

			Checksum struct {
				Value string `xml:",innerxml"`
				Type  string `xml:"type,attr"`
				PkgId string `xml:"pkgid,attr"`
			} `xml:"checksum"`

			Summary  string `xml:"summary"`
			Descr    string `xml:"description"`
			Packager string `xml:"packager"`
			Url      string `xml:"url"`

			Time struct {
				File  string `xml:"file,attr"`
				Build string `xml:"build,attr"`
			} `xml:"time"`

			Size struct {
				Package   int64 `xml:"package,attr"`
				Installed int64 `xml:"installed,attr"`
				Archive   int64 `xml:"archive,attr"`
			} `xml:"size"`

			Location struct {
				Href string `xml:"href,attr"`
			} `xml:"location"`

			Format struct {
				License   string `xml:"rpm:license"`
				Vendor    string `xml:"rpm:vendor"`
				Group     string `xml:"rpm:group"`
				BuildHost string `xml:"rpm:buildhost"`
				SourceRpm string `xml:"rpm:sourcerpm"`

				HeaderRange struct {
					Beg int64 `xml:"start,attr"`
					End int64 `xml:"end,attr"`
				} `xml:"rpm:header-range"`

				Provides []struct {
					Name    string `xml:"name,attr"`
					Flags   string `xml:"flags,attr"`
					Epoch   int    `xml:"epoch,attr"`
					Version string `xml:"ver,attr"`
					Release int    `xml:"rel,attr"`
				} `xml:"rpm-provides"`

				Requires []struct {
					Name    string `xml:"name,attr"`
					Flags   string `xml:"flags,attr"`
					Epoch   int    `xml:"epoch,attr"`
					Version string `xml:"ver,attr"`
					Release int    `xml:"rel,attr"`
					Pre     string `xml:"pre,attr"`
				} `xml:"rpm-requires"`

				Files []string `xml:"file"`
			} `xml:"format"`
		} `xml:"package"`
	}

	// load the yum XML package list
	f, err := os.Open(repo.Primary)
	if err != nil {
		return err
	}
	defer f.Close()

	var r io.Reader
	if rr, err := gzip.NewReader(f); err != nil {
		if err != gzip.ErrHeader {
			return err
		}
		// perhaps not a compressed file after all...
		_, err = f.Seek(0, 0)
		if err != nil {
			return err
		}
		r = f
	} else {
		r = rr
		defer rr.Close()
	}

	var tree xmlTree
	err = xml.NewDecoder(r).Decode(&tree)
	if err != nil {
		return err
	}

	for _, xml := range tree.Packages {
		pkg := NewPackage(xml.Name, xml.Version.Version, xml.Version.Release, xml.Version.Epoch)
		pkg.arch = xml.Arch
		pkg.group = xml.Format.Group
		pkg.location = xml.Location.Href
		for _, v := range xml.Format.Provides {
			prov := NewProvides(
				v.Name,
				v.Version,
				v.Release,
				v.Epoch,
				v.Flags,
				pkg,
			)
			pkg.provides = append(pkg.provides, prov)

			if !str_in_slice(prov.Name(), g_IGNORED_PACKAGES) {
				repo.Provides[prov.Name()] = append(repo.Provides[prov.Name()], prov)
			}
		}

		for _, v := range xml.Format.Requires {
			req := NewRequires(
				v.Name,
				v.Version,
				v.Release,
				v.Epoch,
				v.Flags,
				v.Pre,
			)
			pkg.requires = append(pkg.requires, req)
		}
		pkg.repository = repo.Repository

		// add package to repository
		repo.Packages[pkg.Name()] = append(repo.Packages[pkg.Name()], pkg)
	}

	repo.msg.Infof("start parsing metadata XML file... (%s) [done]\n", repo.Primary)
	return err
}

// FindLatestMatchingName locats a package by name, returns the latest available version.
func (repo *RepositoryXMLBackend) FindLatestMatchingName(name, version, release string) (*Package, error) {
	var pkg *Package
	var err error

	return pkg, err
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (repo *RepositoryXMLBackend) FindLatestMatchingRequire(requirement RPM) (*Package, error) {
	var pkg *Package
	var err error

	return pkg, err
}

// GetPackages returns all the packages known by a YUM repository
func (repo *RepositoryXMLBackend) GetPackages() []*Package {
	pkgs := make([]*Package, 0, len(repo.Packages))
	for _, pkg := range repo.Packages {
		pkgs = append(pkgs, pkg...)
	}
	return pkgs
}

func init() {
	g_backends["RepositoryXMLBackend"] = NewRepositoryXMLBackend
}
