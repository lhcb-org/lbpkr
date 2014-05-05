package yum

import (
	"reflect"
	"sort"
	"testing"
)

func getTestClient(t *testing.T) (*Client, error) {
	const siteroot = "testdata/mysiteroot"
	checkForUpdates := true
	manualConfig := true
	client, err := newClient(
		siteroot,
		[]string{"RepositoryXMLBackend"},
		checkForUpdates,
		manualConfig,
	)
	setupBackend := false
	repo, err := NewRepository("testrepo", "http://dummy-url.org", "testdata/cachedir.tmp",
		[]string{"RepositoryXMLBackend"},
		setupBackend,
		checkForUpdates,
	)
	if err != nil {
		return nil, err
	}

	backend, err := NewRepositoryXMLBackend(repo)
	if err != nil {
		return nil, err
	}
	backend.Primary = "testdata/repo.xml"

	repo.Backend = backend
	err = repo.Backend.LoadDB()
	if err != nil {
		return nil, err
	}

	client.repos[repo.Name] = repo
	client.configured = true
	return client, err
}

func TestPackageMatching(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	p := NewRequires("TestPackage", "1.0.0", "1", "", "EQ", "")
	pkg, err := yum.FindLatestMatchingRequire(p)
	if err != nil {
		t.Fatalf("could not find match: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find match: nil package\n")
	}

	if pkg.Version() != "1.0.0" {
		t.Fatalf("expected version=%q. got=%q\n", "1.0.0", pkg.Version())
	}
}

func TestPackageByNameWithRelease(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TP2", "1.2.5", "1")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "1.2.5"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "1"
	if pkg.Release() != exp {
		t.Fatalf("expected release=%q. got=%q\n", exp, pkg.Release())
	}
}

func TestPackageByNameWithoutRelease(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TP2", "1.2.5", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "1.2.5"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "2"
	if pkg.Release() != exp {
		t.Fatalf("expected release=%q. got=%q\n", exp, pkg.Release())
	}
}

func TestPackageByNameWithoutVersion(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TP2", "", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "1.2.5"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "2"
	if pkg.Release() != exp {
		t.Fatalf("expected release=%q. got=%q\n", exp, pkg.Release())
	}
}

func TestDependencyGreater(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TP2", "", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "1.2.5"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "2"
	if pkg.Release() != exp {
		t.Fatalf("expected release=1. got=%q\n", exp, pkg.Release())
	}

	deps, err := yum.PackageDeps(pkg)
	if err != nil {
		t.Fatalf("could not find package deps: %v\n", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected #deps=%d. got=%d\n", 1, len(deps))
	}

	dep := deps[0]
	exp = "TestPackage"
	if dep.Name() != exp {
		t.Fatalf("expected name=%q. got=%q\n", exp, dep.Name())
	}

	exp = "1.3.7"
	if dep.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, dep.Version())
	}
}

func TestDependencyEqual(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TP3", "", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "1.18.22"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "2"
	if pkg.Release() != exp {
		t.Fatalf("expected release=%q. got=%q\n", exp, pkg.Release())
	}

	deps, err := yum.PackageDeps(pkg)
	if err != nil {
		t.Fatalf("could not find package deps: %v\n", err)
	}

	if len(deps) != 1 {
		t.Fatalf("expected #deps=%d. got=%d\n", 1, len(deps))
	}

	dep := deps[0]
	exp = "TestPackage"
	if dep.Name() != exp {
		t.Fatalf("expected name=%q. got=%q\n", exp, dep.Name())
	}

	exp = "1.2.5"
	if dep.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, dep.Version())
	}
}

func TestCyclicDependency(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TCyclicDep", "", "")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "1.0.0"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "1"
	if pkg.Release() != exp {
		t.Fatalf("expected release=%q. got=%q\n", exp, pkg.Release())
	}

	deps, err := yum.PackageDeps(pkg)
	if err != nil {
		t.Fatalf("could not find package deps: %v\n", err)
	}

	if len(deps) != 2 {
		t.Fatalf("expected #deps=%d. got=%d\n", 1, len(deps))
	}
}

func TestFindReleaseUpdate(t *testing.T) {

	yum, err := getTestClient(t)
	if err != nil {
		t.Fatalf("could not create test repo: %v\n", err)
	}
	defer yum.Close()

	pkg, err := yum.FindLatestMatchingName("TPRel", "4.2.7", "1")
	if err != nil {
		t.Fatalf("could not find latest matching name: %v\n", err)
	}

	if pkg == nil {
		t.Fatalf("could not find latest matching name: nil package\n")
	}

	exp := "4.2.7"
	if pkg.Version() != exp {
		t.Fatalf("expected version=%q. got=%q\n", exp, pkg.Version())
	}

	exp = "1"
	if pkg.Release() != exp {
		t.Fatalf("expected release=%q. got=%q\n", exp, pkg.Release())
	}

	for _, table := range []struct {
		req *Requires
		ver string
		rel string
	}{
		{
			req: NewRequires(pkg.Name(), "", "", "", "EQ", ""),
			ver: "4.2.8",
			rel: "1",
		},
		{
			req: NewRequires(pkg.Name(), pkg.Version(), "", "", "EQ", ""),
			ver: "4.2.7",
			rel: "2",
		},
	} {
		n, err := yum.FindLatestMatchingRequire(table.req)
		if err != nil {
			t.Fatalf("could not find match: %v\n", err)
		}

		if n == nil {
			t.Fatalf("could not find match: nil package\n")
		}

		if n.Version() != table.ver {
			t.Fatalf("expected version=%q. got=%q\n", table.ver, n.Version())
		}

		if n.Release() != table.rel {
			t.Fatalf("expected release=%q. got=%q\n", table.rel, n.Release())
		}
	}

}

func TestLoadConfig(t *testing.T) {
	for _, table := range []struct {
		siteroot string
		backends []string
	}{
		{
			siteroot: "testdata/testconfig-xml",
			backends: []string{"RepositoryXMLBackend"},
		},
		// {
		// 	siteroot: "testdata/testconfig-sqlite",
		// 	backends: []string{
		// 		"RepositorySQLiteBackend",
		// 		"RepositoryXMLBackend",
		// 	},
		// },
	} {
		siteroot := table.siteroot
		checkForUpdates := false
		manualConfig := false
		yum, err := newClient(siteroot, table.backends, checkForUpdates, manualConfig)
		if err != nil {
			t.Fatalf("could not create yum.Client(siteroot=%q): %v\n", siteroot, err)
		}
		defer yum.Close()

		if len(yum.repos) != 3 {
			t.Fatalf("expected 3 repositories. got=%d (siteroot=%q)\n", len(yum.repos), siteroot)
		}

		brunels, err := yum.ListPackages("BRUNEL", "", "")
		if err != nil {
			t.Fatalf("could not list BRUNEL packages: %v (siteroot=%q)\n", err, siteroot)
		}
		if len(brunels) != 7 {
			t.Fatalf("expected 7 BRUNEL packages. got=%d (siteroot=%q)\n", len(brunels), siteroot)
		}

		pkg, err := yum.FindLatestMatchingName("ROOT_5.32.02_x86_64_slc5_gcc46_opt", "1.0.0", "1")
		if err != nil {
			allpkgs, _ := yum.ListPackages("ROOT", "", "")
			str := "["
			for _, pp := range allpkgs {
				str += rpmString(pp) + ", "
			}
			str += "]"
			t.Fatalf("could not find match: %v\namong packages: %v (siteroot=%q)\n", err, str, siteroot)
		}

		if pkg == nil {
			t.Fatalf("could not find match: nil package (siteroot=%q)\n", siteroot)
		}

		exp := "1.0.0"
		if pkg.Version() != exp {
			t.Fatalf("expected ROOT version=%q. got=%q (siteroot=%q)\n", exp, pkg.Version(), siteroot)
		}

		exp = "1"
		if pkg.Release() != exp {
			t.Fatalf("expected ROOT release=%q. got=%q (siteroot=%q)\n", exp, pkg.Release(), siteroot)
		}

		req := NewRequires(
			"BRUNEL_v43r1p1_x86_64_slc5_gcc43_opt",
			"1.0.0",
			"1",
			"", "EQ", "",
		)
		brunel, err := yum.FindLatestMatchingRequire(req)
		if err != nil {
			t.Fatalf("could not find match: %v (siteroot=%q)\n", err, siteroot)
		}

		if brunel == nil {
			t.Fatalf("could not find match: nil package (siteroot=%q)\n")
		}

		exp = "1.0.0"
		if brunel.Version() != exp {
			t.Fatalf("expected BRUNEL version=%q. got=%q (siteroot=%q)\n", exp, brunel.Version(), siteroot)
		}

		exp = "1"
		if brunel.Release() != exp {
			t.Fatalf("expected BRUNEL release=%q. got=%q (siteroot=%q)\n", exp, brunel.Release(), siteroot)
		}

		exprequired := []string{
			"AIDA_3.2.1_common",
			"BRUNEL_v43r1p1",
			"BRUNEL_v43r1p1_x86_64_slc5_gcc43_opt",
			"Boost_1.48.0_python2.6_x86_64_slc5_gcc43_opt",
			"CMT",
			"COMPAT",
			"COOL_COOL_2_8_14_common",
			"COOL_COOL_2_8_14_x86_64_slc5_gcc43_opt",
			"CORAL_CORAL_2_3_23_common",
			"CORAL_CORAL_2_3_23_x86_64_slc5_gcc43_opt",
			"CppUnit_1.12.1_p1_x86_64_slc5_gcc43_opt",
			"DBASE_AppConfig",
			"DBASE_Det_SQLDDDB",
			"DBASE_FieldMap",
			"DBASE_TCK_HltTCK",
			"DBASE_TCK_L0TCK",
			"GAUDI_v23r3",
			"GAUDI_v23r3_x86_64_slc5_gcc43_opt",
			"GSL_1.10_x86_64_slc5_gcc43_opt",
			"Grid_LFC_1.7.4_7sec_x86_64_slc5_gcc43_opt",
			"Grid_cgsi-gsoap_1.3.3_1_x86_64_slc5_gcc43_opt",
			"Grid_gfal_1.11.8_2_x86_64_slc5_gcc43_opt",
			"Grid_globus_4.0.7_VDT_1.10.1_x86_64_slc5_gcc43_opt",
			"Grid_lcg-dm-common_1.7.4_7sec_x86_64_slc5_gcc43_opt",
			"Grid_voms-api-c_1.9.17_1_x86_64_slc5_gcc43_opt",
			"Grid_voms-api-cpp_1.9.17_1_x86_64_slc5_gcc43_opt",
			"HepMC_2.06.05_x86_64_slc5_gcc43_opt",
			"HepPDT_2.06.01_x86_64_slc5_gcc43_opt",
			"LBCOM_v13r1p1",
			"LBCOM_v13r1p1_x86_64_slc5_gcc43_opt",
			"LBSCRIPTS",
			"LCGCMT_64_x86_64_slc5_gcc43_opt",
			"LCGCMT_LCGCMT_64_common",
			"LHCB_v35r1p1",
			"LHCB_v35r1p1_x86_64_slc5_gcc43_opt",
			"PARAM_ChargedProtoANNPIDParam",
			"PARAM_ParamFiles",
			"Python_2.6.5p2_x86_64_slc5_gcc43_opt",
			"QMtest_2.4.1_python2.6_x86_64_slc5_gcc43_opt",
			"REC_v14r1p1",
			"REC_v14r1p1_x86_64_slc5_gcc43_opt",
			"RELAX_RELAX_1_3_0h_x86_64_slc5_gcc43_opt",
			"ROOT_5.34.00_x86_64_slc5_gcc43_opt",
			"XercesC_3.1.1p1_x86_64_slc5_gcc43_opt",
			"blas_20110419_x86_64_slc5_gcc43_opt",
			"castor_2.1.9_9_x86_64_slc5_gcc43_opt",
			"cernlib_2006a_x86_64_slc5_gcc43_opt",
			"clhep_1.9.4.7_x86_64_slc5_gcc43_opt",
			"dcache_client_2.47.5_0_x86_64_slc5_gcc43_opt",
			"expat_2.0.1_x86_64_slc5_gcc43_opt",
			"fastjet_2.4.4_x86_64_slc5_gcc43_opt",
			"fftw3_3.1.2_x86_64_slc5_gcc43_opt",
			"frontier_client_2.8.5_x86_64_slc5_gcc43_opt",
			"gcc_4.3.5_x86_64_slc5",
			"gcc_4.3.5_x86_64_slc5_gcc43_opt",
			"gccxml_0.9.0_20110825_x86_64_slc5_gcc43_opt",
			"graphviz_2.28.0_x86_64_slc5_gcc43_opt",
			"lapack_3.4.0_x86_64_slc5_gcc43_opt",
			"libunwind_5c2cade_x86_64_slc5_gcc43_opt",
			"neurobayes_expert_3.7.0_x86_64_slc5_gcc43_opt",
			"oracle_11.2.0.1.0p3_x86_64_slc5_gcc43_opt",
			"pyanalysis_1.3_python2.6_x86_64_slc5_gcc43_opt",
			"pygraphics_1.2p1_python2.6_x86_64_slc5_gcc43_opt",
			"pytools_1.7_python2.6_x86_64_slc5_gcc43_opt",
			"qt_4.7.4_x86_64_slc5_gcc43_opt",
			"sqlite_3070900_x86_64_slc5_gcc43_opt",
			"tcmalloc_1.7p1_x86_64_slc5_gcc43_opt",
			"uuid_1.42_x86_64_slc5_gcc43_opt",
			"xqilla_2.2.4_x86_64_slc5_gcc43_opt",
			"xrootd_3.1.0p2_x86_64_slc5_gcc43_opt",
			"zlib_1.2.5_x86_64_slc5_gcc43_opt",
		}

		found, err := yum.RequiredPackages(brunel)
		if err != nil {
			t.Fatalf("could not retrieve list of required packages for BRUNEL: %v (siteroot=%q)\n", err, siteroot)
		}

		required := make([]string, 0, len(found))
		for _, p := range found {
			required = append(required, p.Name())
		}
		sort.Strings(exprequired)
		sort.Strings(required)

		if !reflect.DeepEqual(exprequired, required) {
			t.Fatalf("%s: lists of required packages differ\nexp=%v (len=%d)\ngot=%v (len=%d) (siteroot=%q)\n",
				siteroot,
				exprequired,
				len(exprequired),
				required,
				len(required),
				siteroot,
			)
		}
	}
}
