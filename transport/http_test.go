package transport

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/achilleasa/usrv"
)

var localhostCert = []byte(`-----BEGIN CERTIFICATE-----
MIICEzCCAXygAwIBAgIQMIMChMLGrR+QvmQvpwAU6zANBgkqhkiG9w0BAQsFADAS
MRAwDgYDVQQKEwdBY21lIENvMCAXDTcwMDEwMTAwMDAwMFoYDzIwODQwMTI5MTYw
MDAwWjASMRAwDgYDVQQKEwdBY21lIENvMIGfMA0GCSqGSIb3DQEBAQUAA4GNADCB
iQKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9SjY1bIw4
iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZBl2+XsDul
rKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQABo2gwZjAO
BgNVHQ8BAf8EBAMCAqQwEwYDVR0lBAwwCgYIKwYBBQUHAwEwDwYDVR0TAQH/BAUw
AwEB/zAuBgNVHREEJzAlggtleGFtcGxlLmNvbYcEfwAAAYcQAAAAAAAAAAAAAAAA
AAAAATANBgkqhkiG9w0BAQsFAAOBgQCEcetwO59EWk7WiJsG4x8SY+UIAA+flUI9
tyC4lNhbcF2Idq9greZwbYCqTTTr2XiRNSMLCOjKyI7ukPoPjo16ocHj+P3vZGfs
h1fIw3cSS2OolhloGw/XM6RWPWtPAlGykKLciQrBru5NAPvCMsb/I1DAceTiotQM
fblo6RBxUQ==
-----END CERTIFICATE-----`)

// LocalhostKey is the private key for localhostCert.
var localhostKey = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIICXgIBAAKBgQDuLnQAI3mDgey3VBzWnB2L39JUU4txjeVE6myuDqkM/uGlfjb9
SjY1bIw4iA5sBBZzHi3z0h1YV8QPuxEbi4nW91IJm2gsvvZhIrCHS3l6afab4pZB
l2+XsDulrKBxKKtD1rGxlG4LjncdabFn9gvLZad2bSysqz/qTAUStTvqJQIDAQAB
AoGAGRzwwir7XvBOAy5tM/uV6e+Zf6anZzus1s1Y1ClbjbE6HXbnWWF/wbZGOpet
3Zm4vD6MXc7jpTLryzTQIvVdfQbRc6+MUVeLKwZatTXtdZrhu+Jk7hx0nTPy8Jcb
uJqFk541aEw+mMogY/xEcfbWd6IOkp+4xqjlFLBEDytgbIECQQDvH/E6nk+hgN4H
qzzVtxxr397vWrjrIgPbJpQvBsafG7b0dA4AFjwVbFLmQcj2PprIMmPcQrooz8vp
jy4SHEg1AkEA/v13/5M47K9vCxmb8QeD/asydfsgS5TeuNi8DoUBEmiSJwma7FXY
fFUtxuvL7XvjwjN5B30pNEbc6Iuyt7y4MQJBAIt21su4b3sjXNueLKH85Q+phy2U
fQtuUE9txblTu14q3N7gHRZB4ZMhFYyDy8CKrN2cPg/Fvyt0Xlp/DoCzjA0CQQDU
y2ptGsuSmgUtWj3NM9xuwYPm+Z/F84K6+ARYiZ6PYj013sovGKUFfYAqVXVlxtIX
qyUBnu3X9ps8ZfjLZO7BAkEAlT4R5Yl6cGhaJQYZHOde3JEMhNRcVFMO8dJDaFeo
f9Oeos0UUothgiDktdQHxdNEwLjQf7lJJBzV+5OtwswCWA==
-----END RSA PRIVATE KEY-----`)

func TestHttpTransport(t *testing.T) {
	tr := NewHttp()
	tr.Config(NewHttpConfig(8080))
	defer tr.Close()

	reqChan, err := tr.Bind("localhost:8080", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		select {
		case reqMsg := <-reqChan:
			propVal := reqMsg.Property().Get("foo")
			if propVal != "bar" {
				t.Fatalf("Expected property 'foo' to have value 'bar'; got %s", propVal)
			}

			resMsg := tr.ReplyTo(reqMsg)
			resMsg.SetContent([]byte("OK"), nil)
			tr.Send(resMsg, 0, false)
		}
	}()

	reqMsg := tr.MessageTo("test", "localhost:8080", "ep1")
	reqMsg.Property().Set("foo", "bar")
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != nil {
		t.Fatal(err)
	}
	exp := "OK"
	if !bytes.Equal([]byte(exp), content) {
		t.Fatalf("Expected response to be %s; got %s\n", exp, string(content))
	}
	if reqMsg.CorrelationId() != resMsg.CorrelationId() {
		t.Fatalf("Expected res msg corellation id to be %s; got %s", reqMsg.CorrelationId(), resMsg.CorrelationId())
	}
}

func TestHttpTransportTimeouts(t *testing.T) {
	tr := NewHttp()
	tr.Config(NewHttpConfig(8080))
	defer tr.Close()

	reqChan, err := tr.Bind("localhost:8080", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		select {
		case <-reqChan:
		}
	}()

	reqMsg := tr.MessageTo("test", "localhost:8080", "ep1")
	resChan := tr.Send(reqMsg, 1*time.Millisecond, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != usrv.ErrServiceUnavailable {
		t.Fatalf("Expected to get ErrServiceUnavailable; got %v", err)
	}
	if content != nil {
		t.Fatalf("Expected content to be nil; got %v", content)
	}
}

func TestHttpTransportError(t *testing.T) {
	tr := NewHttp()
	tr.Config(NewHttpConfig(8080))
	tr.SetLogger(usrv.NullLogger)
	defer tr.Close()

	reqChan, err := tr.Bind("localhost:8080", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	expErr := fmt.Errorf("An error")

	go func() {
		select {
		case msg := <-reqChan:
			res := tr.ReplyTo(msg)
			res.SetContent(nil, expErr)
			tr.Send(res, 0, false)
		}
	}()

	reqMsg := tr.MessageTo("test", "localhost:8080", "ep1")
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err.Error() != expErr.Error() {
		t.Fatalf("Expected to get error %v; got %v", expErr, err)
	}
	if content != nil {
		t.Fatalf("Expected content to be nil; got %v", content)
	}
}

func TestHttpTransportUnknownEndpoint(t *testing.T) {
	tr := NewHttp()
	tr.Config(NewHttpConfig(8080))
	defer tr.Close()

	// Bind at least one endpoint begin listening for http requests
	_, err := tr.Bind("localhost:8080", "ep2")
	if err != nil {
		t.Fatal(err)
	}

	reqMsg := tr.MessageTo("test", "localhost:8080", "ep1")
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != usrv.ErrServiceUnavailable {
		t.Fatalf("Expected to get ErrServiceUnavailable; got %v", err)
	}
	if content != nil {
		t.Fatalf("Expected content to be nil; got %v", content)
	}
}
func TestHttpsTransport(t *testing.T) {
	// Bypass certificate verification
	oldClient := httpClient
	defer func() {
		httpClient = oldClient
	}()
	dialer := &net.Dialer{Timeout: 1000 * time.Millisecond}
	httpClient = &http.Client{
		Transport: &http.Transport{
			Dial:            dialer.Dial,
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	certFile, err := ioutil.TempFile("", "cert")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(certFile.Name())
	_, err = certFile.Write(localhostCert)
	if err != nil {
		t.Fatal(err)
	}
	certFile.Close()

	certKeyFile, err := ioutil.TempFile("", "cert")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(certKeyFile.Name())
	_, err = certKeyFile.Write(localhostKey)
	if err != nil {
		t.Fatal(err)
	}
	certKeyFile.Close()

	tr := NewHttp()
	tr.Config(NewHttpsConfig(8081, certFile.Name(), certKeyFile.Name()))
	defer tr.Close()

	// Bind at least one endpoint begin listening for http requests
	reqChan, err := tr.Bind("localhost:8081", "ep1")
	if err != nil {
		t.Fatal(err)
	}

	go func() {
		select {
		case reqMsg := <-reqChan:
			propVal := reqMsg.Property().Get("foo")
			if propVal != "bar" {
				t.Fatalf("Expected property 'foo' to have value 'bar'; got %s", propVal)
			}

			resMsg := tr.ReplyTo(reqMsg)
			resMsg.SetContent([]byte("OK"), nil)
			tr.Send(resMsg, 0, false)
		}
	}()

	reqMsg := tr.MessageTo("test", "localhost:8081", "ep1")
	reqMsg.Property().Set("foo", "bar")
	reqMsg.SetContent([]byte("Hello"), nil)
	resChan := tr.Send(reqMsg, 0, true)

	resMsg := <-resChan
	content, err := resMsg.Content()
	if err != nil {
		t.Fatal(err)
	}
	exp := "OK"
	if !bytes.Equal([]byte(exp), content) {
		t.Fatalf("Expected response to be %s; got %s\n", exp, string(content))
	}
	if reqMsg.CorrelationId() != resMsg.CorrelationId() {
		t.Fatalf("Expected res msg corellation id to be %s; got %s", reqMsg.CorrelationId(), resMsg.CorrelationId())
	}
}
