package yum

import (
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
