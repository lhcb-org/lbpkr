package yum

import (
	"io"
	"net/http"
	"net/url"
	"os"
)

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

func str_in_slice(str string, slice []string) bool {
	for _, v := range slice {
		if str == v {
			return true
		}
	}
	return false
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
		resp, err := http.Get(rpath)
		if err != nil {
			return nil, err
		}
		return resp.Body, nil
	}
}
