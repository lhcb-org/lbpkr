package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"os/exec"
	"reflect"
	"regexp"
	"testing"
)

func TestLbpkrSelfBdist(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	cmd := newCommand("lbpkr", "self", "bdist")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running bdist: %v", err)
	}
}

func TestLbpkrSelfBdistRpm(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	if _, err := exec.LookPath("rpmbuild"); err != nil {
		t.Skip("no rpmbuild installed")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	cmd := newCommand("lbpkr", "self", "bdist-rpm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running bdist-rpm: %v", err)
	}
}

func TestLbpkrInstallLbpkr(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running install: %v", err)
	}
}

func TestLbpkrInstallWithUpdate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	// install an old version
	cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_dbg-1.0.0-12")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running install: %v", err)
	}

	// install a new version + a new package
	cmd = newCommand("lbpkr", "install", "-siteroot="+tmpdir, "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_dbg-1.0.0-72", "CASTOR-9ccc5_2.1.13_6_x86_64_slc6_gcc48_dbg-1.0.0-72")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running install: %v", err)
	}

	// make sure we have only 3 packages installed
	cmd = newCommand("lbpkr", "installed", "-siteroot="+tmpdir)
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = buf
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running installed: %v", err)
	}

	want := []byte(`AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_dbg-1.0.0-72
CASTOR-9ccc5_2.1.13_6_x86_64_slc6_gcc48_dbg-1.0.0-72
gcc_4.8.1_x86_64_slc6-1.0.0-1
`)
	if !reflect.DeepEqual(want, buf.Bytes()) {
		t.Fatalf("invalid number of packages.\nwant: %s\n got: %s\n", string(want), string(buf.Bytes()))
	}
}

func TestLbpkrInstallNoUpdateThenUpdate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	// install an old version
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr-0.1.20140701")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running install: %v", err)
		}
	}

	// (explicitly) install a newer version - should FAIL
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr-0.1.20141113")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err == nil {
			t.Fatalf("running install should have FAILED")
		}
	}

	// install _a_ newer version - should FAIL
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err == nil {
			t.Fatalf("running install should have FAILED")
		}
	}

	// update
	{
		cmd := newCommand("lbpkr", "update", "-siteroot="+tmpdir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running update: %v", err)
		}
	}

}

func TestLbpkrInstallThenUpdate(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	// install an old version
	cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_dbg-1.0.0-12", "lbpkr-0.1.20141210-0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running install: %v", err)
	}

	// update
	cmd = newCommand("lbpkr", "update", "-siteroot="+tmpdir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running install: %v", err)
	}

	// make sure we have only 3 packages installed
	cmd = newCommand("lbpkr", "installed", "-siteroot="+tmpdir)
	buf := new(bytes.Buffer)
	cmd.Stdout = buf
	cmd.Stderr = buf
	cmd.Dir = tmpdir

	err = cmd.Run()
	if err != nil {
		t.Fatalf("error running installed: %v", err)
	}

	want := regexp.MustCompile(`AIDA-3fe9f_3.2.1_x86_64_slc6_gcc48_dbg-1.0.0-72
gcc_.*?
lbpkr-.*?
`)
	if !want.Match(buf.Bytes()) {
		t.Fatalf("invalid number of packages.\nwant: %s\n got: %s\n", want.String(), string(buf.Bytes()))
	}

}

func TestLbpkrDryRun(t *testing.T) {

	t.Parallel()
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	tmpdir, err := ioutil.TempDir("", "test-lbpkr-")
	if err != nil {
		t.Fatalf("error creating temporary directory: %v", err)
	}
	defer os.RemoveAll(tmpdir)

	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "-dry-run", "lbpkr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running install: %v", err)
		}
	}

	{
		cmd := newCommand("lbpkr", "installed", "-siteroot="+tmpdir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running installed: %v", err)
		}
	}

	// install an old version
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr-0.1.20140701")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running install: %v", err)
		}
	}

	{
		cmd := newCommand("lbpkr", "installed", "-siteroot="+tmpdir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running installed: %v", err)
		}
	}

	// dry-run install a new(er) version
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "-dry-run", "lbpkr-0.1.20141113")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running install: %v", err)
		}
	}

	// install a new(er) version (that should fail)
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr-0.1.20141113")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err == nil {
			t.Fatalf("running install should have FAILED")
		}
	}

	// dry-run remove lbpkr
	{
		cmd := newCommand("lbpkr", "rm", "-siteroot="+tmpdir, "-dry-run", "lbpkr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running remove: %v", err)
		}
	}

	// install a new(er) version (that should STILL fail)
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr-0.1.20141113")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err == nil {
			t.Fatalf("running install should have FAILED")
		}
	}

	// remove lbpkr
	{
		cmd := newCommand("lbpkr", "rm", "-siteroot="+tmpdir, "lbpkr")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running remove: %v", err)
		}
	}

	// install a new(er) version (that should NOW succeed)
	{
		cmd := newCommand("lbpkr", "install", "-siteroot="+tmpdir, "lbpkr-0.1.20141113")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Dir = tmpdir

		err = cmd.Run()
		if err != nil {
			t.Fatalf("error running install: %v", err)
		}
	}
}

func TestRPMSplit(t *testing.T) {
	t.Parallel()
	for _, table := range []struct {
		rpm  string
		want [3]string
	}{
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-1.0.0-",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "1.0.0", ""},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-10.20.30-1",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "10.20.30", "1"},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-1.0.0-1",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "1.0.0", "1"},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-1.0.0-71",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "1.0.0", "71"},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-1.0-71",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "1.0", "71"},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-10.20.30",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "10.20.30", ""},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-1.0.0",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "1.0.0", ""},
		},
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "", ""},
		},
		{
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt-1.0.0-1",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "1.0.0", "1"},
		},
		{
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt-10.20.30-1",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "10.20.30", "1"},
		},
		{
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt-1.0.0-71",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "1.0.0", "71"},
		},
		{
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt-10.20.30",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "10.20.30", ""},
		},
		{
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt-1.0.0",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "1.0.0", ""},
		},
		{
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "", ""},
		},
		{
			rpm:  "BRUNEL_v45r1-1.0.0-21",
			want: [3]string{"BRUNEL_v45r1", "1.0.0", "21"},
		},
		{
			rpm:  "BRUNEL_v45r1-1.0.0-1",
			want: [3]string{"BRUNEL_v45r1", "1.0.0", "1"},
		},
		{
			rpm:  "BRUNEL_v45r1-1.0.0",
			want: [3]string{"BRUNEL_v45r1", "1.0.0", ""},
		},
		{
			rpm:  "BRUNEL_v45r1",
			want: [3]string{"BRUNEL_v45r1", "", ""},
		},
		{
			rpm:  "BRUNEL_v45r1_x86_64_slc6_gcc48_opt-1.0.0-21",
			want: [3]string{"BRUNEL_v45r1_x86_64_slc6_gcc48_opt", "1.0.0", "21"},
		},
		{
			rpm:  "BRUNEL_v45r1_x86_64_slc6_gcc48_opt-1.0.0-1",
			want: [3]string{"BRUNEL_v45r1_x86_64_slc6_gcc48_opt", "1.0.0", "1"},
		},
		{
			rpm:  "BRUNEL_v45r1_x86_64_slc6_gcc48_opt-1.0.0",
			want: [3]string{"BRUNEL_v45r1_x86_64_slc6_gcc48_opt", "1.0.0", ""},
		},
		{
			rpm:  "BRUNEL_v45r1_x86_64_slc6_gcc48_opt",
			want: [3]string{"BRUNEL_v45r1_x86_64_slc6_gcc48_opt", "", ""},
		},
	} {
		rpm := splitRPM(table.rpm)
		if rpm != table.want {
			t.Errorf(
				"%s: error.\nwant=[name=%q version=%q release=%q].\n got=[name=%q version=%q release=%q]\n",
				table.rpm,
				table.want[0], table.want[1], table.want[2],
				rpm[0], rpm[1], rpm[2],
			)
		}
	}
}
