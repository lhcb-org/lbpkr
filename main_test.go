package main

import (
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
)

func TestLbpkrSelfBdist(t *testing.T) {
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

func TestRPMSplit(t *testing.T) {
	for _, table := range []struct {
		rpm  string
		want [3]string
	}{
		{
			rpm:  "AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt-1.0.0-",
			want: [3]string{"AIDA-3fe9f_3.2.1_x86_64_slc6_gcc49_opt", "1.0.0", ""},
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
			rpm:  "LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt-1.0.0-71",
			want: [3]string{"LCG_67_AIDA_3.2.1_x86_64_slc6_gcc47_opt", "1.0.0", "71"},
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
