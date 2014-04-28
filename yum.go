package main

type YumClient struct {
	siteroot string
}

func NewClient(siteroot string) *YumClient {
	return &YumClient{
		siteroot: siteroot,
	}
}

// EOF
