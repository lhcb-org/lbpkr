package yum

import (
	"compress/bzip2"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"

	"github.com/gonuts/logger"
	_ "github.com/mattn/go-sqlite3"
)

// RepositorySQLiteBackend is Backend querying YUM SQLite repositories
type RepositorySQLiteBackend struct {
	Name         string
	Packages     map[string][]*Package
	Provides     map[string][]*Provides
	DBNameCompr  string
	DBName       string
	PrimaryCompr string
	Primary      string
	Repository   *Repository
	db           *sql.DB
	msg          *logger.Logger
}

func NewRepositorySQLiteBackend(repo *Repository) (*RepositorySQLiteBackend, error) {
	const comprdbname = "primary.sqlite.bz2"
	const dbname = "primary.sqlite"
	return &RepositorySQLiteBackend{
		Name:         "RepositorySQLiteBackend",
		Packages:     make(map[string][]*Package),
		Provides:     make(map[string][]*Provides),
		DBNameCompr:  comprdbname,
		DBName:       dbname,
		PrimaryCompr: filepath.Join(repo.CacheDir, comprdbname),
		Primary:      filepath.Join(repo.CacheDir, dbname),
		Repository:   repo,
		msg:          repo.msg,
	}, nil
}

// YumDataType returns the ID for the data type as used in the repomd.xml file
func (repo *RepositorySQLiteBackend) YumDataType() string {
	return "primary_db"
}

// Download the DB from server
func (repo *RepositorySQLiteBackend) GetLatestDB(url string) error {
	var err error
	repo.msg.Debugf("downloading latest version of SQLite DB\n")
	out, err := os.Create(repo.PrimaryCompr)
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
	if err != nil {
		return err
	}

	repo.msg.Debugf("decompressing latest version of SQLite DB\n")
	dbfile, err := os.Create(repo.Primary)
	if err != nil {
		return err
	}
	defer dbfile.Close()

	err = out.Sync()
	if err != nil {
		return err
	}
	_, err = out.Seek(0, 0)
	if err != nil {
		return err
	}

	return repo.decompress(dbfile, out)
}

// Check whether the DB is there
func (repo *RepositorySQLiteBackend) HasDB() bool {
	return path_exists(repo.PrimaryCompr)
}

// Load loads the DB
func (repo *RepositorySQLiteBackend) LoadDB() error {
	var err error
	if !path_exists(repo.Primary) {
		err = repo.decompress2(repo.Primary, repo.PrimaryCompr)
		if err != nil {
			return err
		}
	}

	db, err := sql.Open("sqlite3", repo.Primary)
	if err != nil {
		return err
	}
	repo.db = db
	return err
}

// FindLatestMatchingName locats a package by name, returns the latest available version.
func (repo *RepositorySQLiteBackend) FindLatestMatchingName(name, version string, release int) (*Package, error) {
	var pkg *Package
	var err error

	pkgs, err := repo.loadPackagesByName(name, version)
	if err != nil {
		return nil, err
	}
	matching := make(RPMSlice, 0, len(pkgs))
	req := NewRequires(name, version, release, 0, "EQ", "")
	for _, pkg := range pkgs {
		if req.ProvideMatches(pkg) {
			matching = append(matching, pkg)
		}
	}

	if len(matching) <= 0 {
		err = fmt.Errorf("no such package %q", name)
		return nil, err
	}

	sort.Sort(matching)
	pkg = matching[len(matching)-1].(*Package)
	return pkg, nil
}

// FindLatestMatchingRequire locates a package providing a given functionality.
func (repo *RepositorySQLiteBackend) FindLatestMatchingRequire(requirement *Requires) (*Package, error) {
	var pkg *Package
	var err error

	// list of all Provides with the same name
	provides, err := repo.findProvidesByName(requirement.Name())
	if err != nil {
		return nil, err
	}

	matching := make(RPMSlice, 0, len(provides))
	for _, pr := range provides {
		if requirement.ProvideMatches(pr) {
			matching = append(matching, pr)
		}
	}

	if len(matching) <= 0 {
		err = fmt.Errorf("no Provides for %q", requirement.Name())
		return nil, err
	}

	// now look-up the matching package
	sort.Sort(matching)
	prov := matching[len(matching)-1].(*Provides)
	pkgs, err := repo.loadPackagesProviding(prov)
	if err != nil {
		return nil, err
	}

	if len(pkgs) <= 0 {
		err = fmt.Errorf("no such package %q", requirement.Name())
		return nil, err
	}

	matching = matching[:0]
	for _, p := range pkgs {
		matching = append(matching, p)
	}

	sort.Sort(matching)
	pkg = matching[len(matching)-1].(*Package)
	return pkg, err
}

// GetPackages returns all the packages known by a YUM repository
func (repo *RepositorySQLiteBackend) GetPackages() []*Package {
	query := "select pkgKey, name, version, release, epoch, rpm_group, arch, location_href from packages"
	stmt, err := repo.db.Prepare(query)
	if err != nil {
		repo.msg.Errorf("db-error: %v\n", err)
		return nil
	}
	defer stmt.Close()

	rows, err := stmt.Query()
	if err != nil {
		repo.msg.Errorf("db-error: %v\n", err)
		return nil
	}
	defer rows.Close()

	pkgs := make([]*Package, 0)
	for rows.Next() {
		pkg, err := repo.newPackageFromScan(rows)
		if err != nil {
			repo.msg.Errorf("db-error: %v\n", err)
			repo.msg.Errorf("query: %q\n", query)
			panic(err)
			return nil
		}
		pkgs = append(pkgs, pkg)
	}
	err = rows.Err()
	if err != nil {
		repo.msg.Errorf("db-error-err: %v\n", err)
		panic(err)
		return nil
	}

	err = rows.Close()
	if err != nil {
		repo.msg.Errorf("db-error-close-row: %v\n", err)
		panic(err)
		return nil
	}

	err = stmt.Close()
	if err != nil {
		repo.msg.Errorf("db-error-close-stmt: %v\n", err)
		panic(err)
		return nil
	}

	return pkgs
}

func (repo *RepositorySQLiteBackend) newPackageFromScan(rows *sql.Rows) (*Package, error) {
	var pkg Package
	pkg.repository = repo.Repository
	pkg.requires = make([]*Requires, 0)
	pkg.provides = make([]*Provides, 0)
	var pkgkey sql.NullInt64
	var name []byte
	var version []byte
	var rel_str []byte
	var epoch_str []byte
	var group []byte
	var arch []byte
	var location []byte
	err := rows.Scan(
		&pkgkey,
		&name,
		&version,
		&rel_str,
		&epoch_str,
		&group,
		&arch,
		&location,
	)
	if err != nil {
		repo.msg.Errorf("scan error: %v\n", err)
		return nil, err
	}

	pkg.rpmBase.name = string(name)
	pkg.rpmBase.version = string(version)

	rel, err := strconv.Atoi(string(rel_str))
	if err != nil {
		return nil, err
	}
	pkg.rpmBase.release = rel

	epoch, err := strconv.Atoi(string(epoch_str))
	if err != nil {
		return nil, err
	}
	pkg.rpmBase.epoch = epoch

	pkg.group = string(group)
	pkg.arch = string(arch)
	pkg.location = string(location)

	err = repo.loadRequires(int(pkgkey.Int64), &pkg)
	if err != nil {
		repo.msg.Errorf("load-requires error: %v\n", err)
		return nil, err
	}

	err = repo.loadProvides(int(pkgkey.Int64), &pkg)
	if err != nil {
		repo.msg.Errorf("load-provides error: %v\n", err)
		return nil, err
	}

	return &pkg, nil
}

func (repo *RepositorySQLiteBackend) loadProvides(pkgkey int, pkg *Package) error {
	var err error
	stmt, err := repo.db.Prepare(
		"select name, version, release, epoch, flags from provides where pkgKey=?",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.Query(pkgkey)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var p Provides
		var name []byte
		var version []byte
		var release []byte
		var epoch []byte
		var flags []byte
		err = rows.Scan(
			&name, &version, &release,
			&epoch, &flags,
		)
		if err != nil {
			return err
		}

		p.rpmBase.name = string(name)
		p.rpmBase.version = string(version)
		rel, err := strconv.Atoi(string(release))
		if err != nil {
			return err
		}
		p.rpmBase.release = rel

		epo, err := strconv.Atoi(string(epoch))
		if err != nil {
			return err
		}
		p.rpmBase.epoch = epo

		p.rpmBase.flags = string(flags)
		p.Package = pkg
		pkg.provides = append(pkg.provides, &p)
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	err = rows.Close()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return err
}

func (repo *RepositorySQLiteBackend) loadRequires(pkgkey int, pkg *Package) error {
	var err error
	stmt, err := repo.db.Prepare(
		"select name, version, release, epoch, flags, pre from requires where pkgKey=?",
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	rows, err := stmt.Query(pkgkey)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var req Requires
		var name []byte
		var version []byte
		var release []byte
		var epoch []byte
		var flags []byte
		var pre []byte
		err = rows.Scan(
			&name, &version, &release,
			&epoch, &flags,
			&pre,
		)
		if err != nil {
			return err
		}

		req.rpmBase.name = string(name)
		req.rpmBase.version = string(version)
		rel, err := strconv.Atoi(string(release))
		if err != nil {
			return err
		}
		req.rpmBase.release = rel

		epo, err := strconv.Atoi(string(epoch))
		if err != nil {
			return err
		}
		req.rpmBase.epoch = epo

		req.rpmBase.flags = string(flags)
		req.pre = string(pre)

		if err != nil {
			return err
		}

		if req.rpmBase.flags == "" {
			req.rpmBase.flags = "EQ"
		}
	}
	err = rows.Err()
	if err != nil {
		return err
	}

	err = rows.Close()
	if err != nil {
		return err
	}

	err = stmt.Close()
	if err != nil {
		return err
	}

	return err
}

func (repo *RepositorySQLiteBackend) loadPackagesByName(name, version string) ([]*Package, error) {
	var err error
	pkgs := make([]*Package, 0)
	args := []interface{}{name}
	query := "select pkgKey, name, version, release, epoch, rpm_group, arch, location_href" +
		"from packages where name = ?"
	if version != "" {
		query += " and version = ?"
		args = append(args, version)
	}

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		pkg, err := repo.newPackageFromScan(rows)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}

	err = rows.Close()
	if err != nil {
		return nil, err
	}

	err = stmt.Close()
	if err != nil {
		return nil, err
	}

	return pkgs, err
}

func (repo *RepositorySQLiteBackend) findProvidesByName(name string) ([]*Provides, error) {
	var err error
	provides := make([]*Provides, 0)
	query := "select pkgKey, name, version, release, epoch, flags from provides where name=?"
	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Provides
		err = rows.Scan(
			&p.rpmBase.name, &p.rpmBase.version, &p.rpmBase.release,
			&p.rpmBase.epoch,
			&p.rpmBase.flags,
		)
		if err != nil {
			return nil, err
		}
		p.Package = nil
		provides = append(provides, &p)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	err = rows.Close()
	if err != nil {
		return nil, err
	}

	err = stmt.Close()
	if err != nil {
		return nil, err
	}

	return provides, err
}

func (repo *RepositorySQLiteBackend) loadPackagesProviding(prov *Provides) ([]*Package, error) {
	pkgs := make([]*Package, 0)
	var err error

	args := []interface{}{
		prov.Name(),
		prov.Version(),
	}
	query := `select p.pkgKey, p.name, p.version, p.release, p.epoch, p.rpm_group, p.arch, p.location_href
             from packages p, provides r
             where p.pkgKey = r.pkgKey
             and r.name = ?
             and r.version = ?`
	if prov.Release() > 0 {
		query += " and r.release = ?"
		args = append(args, prov.Release())
	}

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		pkg, err := repo.newPackageFromScan(rows)
		if err != nil {
			return nil, err
		}
		pkgs = append(pkgs, pkg)
	}

	err = rows.Err()
	if err != nil {
		return nil, err
	}

	err = rows.Close()
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	return pkgs, err
}

// decompress decompresses src into dst
func (repo *RepositorySQLiteBackend) decompress(dst io.Writer, src io.Reader) error {
	var err error
	r := bzip2.NewReader(src)
	_, err = io.Copy(dst, r)
	return err
}

// decompress2 decompresses src into dst
func (repo *RepositorySQLiteBackend) decompress2(dst string, src string) error {
	fdst, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fdst.Close()

	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	err = repo.decompress(fdst, fsrc)
	if err != nil {
		return err
	}

	err = fdst.Sync()
	if err != nil {
		return err
	}

	return err
}

func init() {
	g_backends["RepositorySQLiteBackend"] = func(repo *Repository) (Backend, error) {
		return NewRepositorySQLiteBackend(repo)
	}
}

// EOF
