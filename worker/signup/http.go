package main

import (
	"bmw/lib"
	"net/http"
)

var (
	httpClient *http.Client
	keyFile    string
	crtFile    string
)

func init() {
	keyFile = "./client_key.pem"
	crtFile = "./client_crt.pem"

	tlsConfig := lib.MustGetTlsConfiguration(keyFile, crtFile, crtFile)
	transport := &http.Transport{TLSClientConfig: tlsConfig}
	httpClient = &http.Client{Transport: transport}
}

func HTTPPostToApp(url string, config *lib.RequestConfig) ([]byte, error) {
	config.Client = httpClient
	return lib.HTTPPost(url, config)
}

func HTTPGetToApp(url string, config *lib.RequestConfig) ([]byte, error) {
	config.Client = httpClient
	return lib.HTTPGet(url, config)
}
