package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"
)

// newCommand is like os/exec.Command but ensures the subprocess is part of a process-group
func newCommand(name string, args ...string) *exec.Cmd {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// killProcess kills a process by way of its process-group.
func killProcess(p *os.Process) error {
	pgid, err := syscall.Getpgid(p.Pid)
	if err != nil {
		return err
	}
	err = syscall.Kill(-pgid, syscall.SIGKILL) // note the minus sign
	return err
}

func path_exists(name string) bool {
	_, err := os.Stat(name)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

func handle_err(err error) {
	if err != nil {
		if g_ctx != nil {
			g_ctx.msg.Errorf("%v\n", err.Error())
		} else {
			fmt.Fprintf(os.Stderr, "**error** %v\n", err)
		}
		os.Exit(1)
	}
}

func bincp(dst, src string) error {
	fsrc, err := os.Open(src)
	if err != nil {
		return err
	}
	defer fsrc.Close()

	fdst, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer fdst.Close()

	fisrc, err := os.Stat(src)
	if err != nil {
		return err
	}

	err = fdst.Chmod(fisrc.Mode())
	if err != nil {
		return err
	}

	_, err = io.Copy(fdst, fsrc)
	return err
}

func _tar_gz(targ, workdir string) error {

	f, err := os.Create(targ)
	if err != nil {
		return err
	}
	zout := gzip.NewWriter(f)
	tw := tar.NewWriter(zout)

	err = filepath.Walk(workdir, func(path string, fi os.FileInfo, err error) error {
		//fmt.Printf("::> [%s]...\n", path)
		if !strings.HasPrefix(path, workdir) {
			err = fmt.Errorf("walked filename %q doesn't begin with workdir %q", path, workdir)
			return err

		}
		name := path[len(workdir):] //path

		// make name "relative"
		if strings.HasPrefix(name, "/") {
			name = name[1:]
		}
		target, _ := os.Readlink(path)
		if err != nil {
			return err
		}
		hdr, err := tar.FileInfoHeader(fi, target)
		if err != nil {
			return err
		}
		hdr.Name = name
		hdr.Uname = "root"
		hdr.Gname = "root"
		hdr.Uid = 0
		hdr.Gid = 0

		// Force permissions to 0755 for executables, 0644 for everything else.
		if fi.Mode().Perm()&0111 != 0 {
			hdr.Mode = hdr.Mode&^0777 | 0755
		} else {
			hdr.Mode = hdr.Mode&^0777 | 0644
		}

		err = tw.WriteHeader(hdr)
		if err != nil {
			return fmt.Errorf("Error writing file %q: %v", name, err)
		}
		// handle directories and symlinks
		if hdr.Size <= 0 {
			return nil
		}
		r, err := os.Open(path)
		if err != nil {
			return err
		}
		defer r.Close()
		_, err = io.Copy(tw, r)
		return err
	})
	if err != nil {
		return err
	}
	if err := tw.Close(); err != nil {
		return err
	}
	if err := zout.Close(); err != nil {
		return err
	}
	return f.Close()
}

func sanitizePathOrURL(path string) (string, error) {
	switch {
	case strings.Contains(path, "://"):
		// a url. hopefully a correctly formed one.
		return path, nil
	case strings.Contains(path, ":/"):
		// maybe a url. hopefully a correctly formed one.
		return path, nil
	}
	p, err := filepath.Abs(path)
	if err != nil {
		return path, err
	}

	// p, err = filepath.EvalSymlinks(p)
	// if err != nil {
	// 	return path, err
	// }

	p = filepath.Clean(p)

	return p, nil
}

func getRemoteData(rpath string) (io.ReadCloser, error) {
	url, err := url.Parse(rpath)
	if err != nil {
		return nil, err
	}

	switch url.Scheme {
	case "file":
		f, err := os.Open(url.Path)
		if err != nil {
			return nil, err
		}
		return f, nil

	default:
		client := &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		}
		resp, err := client.Get(rpath)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
}

var (
	rpmRe1 = regexp.MustCompile(`(.*?)-([\d\.]+)$`)
	rpmRe2 = regexp.MustCompile(`(.*?)-([\d\.]+)-(\d*)$`)
)

// splitRPM splits a RPM package name into name-version-release
func splitRPM(rpm string) [3]string {
	switch strings.Count(rpm, "-") {
	case 0:
		return [3]string{rpm, "", ""}
	case 1:
		re := rpmRe1.FindAllStringSubmatch(rpm, -1)
		if len(re) == 1 {
			m := re[0]
			switch len(m) {
			case 2:
				return [3]string{m[1], "", ""}
			case 3:
				return [3]string{m[1], m[2], ""}
			}
		}
	default:
		re := rpmRe2.FindAllStringSubmatch(rpm, -1)
		if len(re) == 1 {
			m := re[0]
			switch len(m) {
			case 2:
				return [3]string{m[1], "", ""}
			case 3:
				return [3]string{m[1], m[2], ""}
			case 4:
				return [3]string{m[1], m[2], m[3]}
			}
		}

		re = rpmRe1.FindAllStringSubmatch(rpm, -1)
		if len(re) == 1 {
			m := re[0]
			switch len(m) {
			case 2:
				return [3]string{m[1], m[2], ""}
			case 3:
				return [3]string{m[1], m[2], ""}
			}
		}
	}
	return [3]string{rpm, "", ""}
}

// EOF
