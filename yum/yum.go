package yum

type Client struct {
	siteroot string
}

func New(siteroot string) *Client {
	return &Client{
		siteroot: siteroot,
	}
}

// FindLatestMatchingName locates a package by name and returns the latest available version
func (yum *Client) FindLatestMatchingName(name, version, release string) (string, error) {
	var err error
	if version == "" {
		version = "0.0.1"
	}
	if release == "" {
		release = "1"
	}
	pkg := name + "-" + version + "-" + release
	return pkg, err
}

// EOF
