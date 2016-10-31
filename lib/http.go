package lib

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/golang/glog"
)

// NewHTTPReqeust makes a http request
func NewHTTPReqeust(method, url string, queries, headers map[string]string, body []byte) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	if queries != nil {
		query := req.URL.Query()
		for key, val := range queries {
			query.Set(key, val)
		}
		req.URL.RawQuery = query.Encode()
	}

	if headers != nil {
		for key, val := range headers {
			req.Header.Set(key, val)
		}
	}

	if body != nil {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
	}

	return req, nil
}

type RequestConfig struct {
	Params  map[string]string
	Headers map[string]string
	Timeout uint16
	Body    []byte
	Client  *http.Client
}

func NewReqeustConfig(params, headers map[string]string, timeout uint16, body []byte, client *http.Client) *RequestConfig {
	if timeout <= 0 {
		timeout = 10
	}
	config := &RequestConfig{
		Params:  params,
		Headers: headers,
		Timeout: timeout,
		Body:    body,
		Client:  client,
	}

	return config
}

// HTTPGet returns http response body in []byte, timeout in second
func HTTPGet(url string, config *RequestConfig) ([]byte, error) {
	req, err := NewHTTPReqeust("GET", url, config.Params, config.Headers, nil)
	if err != nil {
		glog.Error("HttpGet needs request not to be nil, ", err)
		return nil, err
	}

	client := http.DefaultClient
	if config.Client != nil {
		client = config.Client
	}
	client.Timeout = time.Duration(config.Timeout) * time.Second

	res, err := client.Do(req)
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("resp code is %d", res.StatusCode)
	}

	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return b, nil
}

// HTTPGetFile store body in single file, return file and file's content type
func HTTPGetFile(url string, config *RequestConfig) (outFName, contentType string, contentLength int64, err error) {
	tmpFp, err := ioutil.TempFile("", "dl")
	if err != nil {
		glog.Error("Create tmp file error, ", err)
		return
	}
	defer tmpFp.Close()
	outFName = tmpFp.Name()

	req, err := NewHTTPReqeust("GET", url, config.Params, config.Headers, nil)
	if err != nil {
		glog.Error("HttpGet needs request not to be nil, ", err)
		return
	}

	client := http.DefaultClient
	if config.Client != nil {
		client = config.Client
	}
	client.Timeout = time.Duration(config.Timeout) * time.Second

	res, err := client.Do(req)
	if err != nil {
		glog.Error(err)
		return
	}
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("resp code is %d", res.StatusCode)
		return
	}

	contentType = res.Header.Get("Content-Type")
	contentLength, _ = strconv.ParseInt(res.Header.Get("Content-Length"), 10, 64)

	reader := bufio.NewReader(res.Body)
	defer res.Body.Close()

	_, err = reader.WriteTo(tmpFp)
	if err != nil {
		glog.Error(err)
		return
	}

	return
}

// HTTPPost do http post
func HTTPPost(url string, config *RequestConfig) ([]byte, error) {
	req, err := NewHTTPReqeust("POST", url, config.Params, config.Headers, config.Body)
	if err != nil {
		glog.Error("HttpPost needs request not to be nil, ", err)
		return nil, err
	}

	client := http.DefaultClient
	if config.Client != nil {
		client = config.Client
	}
	client.Timeout = time.Duration(config.Timeout) * time.Second

	res, err := client.Do(req)
	if err != nil {
		glog.Error(err)
		return nil, err
	}
	b, err := ioutil.ReadAll(res.Body)
	defer res.Body.Close()

	if err != nil {
		glog.Error(err)
		return nil, err
	}

	return b, nil
}

func MustLoadCertificates(privateKeyFile, certificateFile, caFile string) (tls.Certificate, *x509.CertPool) {
	mycert, err := tls.LoadX509KeyPair(certificateFile, privateKeyFile)
	if err != nil {
		panic(err)
	}
	pem, err := ioutil.ReadFile(caFile)
	if err != nil {
		panic(err)
	}

	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(pem) {
		panic("Failed appending certs")
	}

	return mycert, certPool
}

func MustGetTlsConfiguration(privateKeyFile, certificateFile, caFile string) *tls.Config {
	config := &tls.Config{}
	mycert, certPool := MustLoadCertificates(privateKeyFile, certificateFile, caFile)
	config.Certificates = make([]tls.Certificate, 1)
	config.Certificates[0] = mycert

	config.RootCAs = certPool
	config.ClientCAs = certPool
	config.InsecureSkipVerify = true

	//config.ClientAuth = tls.RequireAndVerifyClientCert

	//Optional stuff

	//Use only modern ciphers
	config.CipherSuites = []uint16{tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256}

	//Use only TLS v1.2
	config.MinVersion = tls.VersionTLS12

	//Don't allow session resumption
	// config.SessionTicketsDisabled = true
	return config
}
