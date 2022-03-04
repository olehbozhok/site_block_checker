package proxy_util

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/pkg/errors"
	"golang.org/x/net/proxy"
)

type Client struct {
	*resty.Client
}

func GetClient(addr string, username, password string) (*Client, error) {
	addr = strings.TrimSpace(addr)
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	var proxyAuth *proxy.Auth
	if username != "" {
		proxyAuth = new(proxy.Auth)
		proxyAuth.User = username
		proxyAuth.Password = password
	}

	dialer, err := proxy.SOCKS5(
		"tcp",
		addr,
		proxyAuth,
		nil) // with net.Dialer dialer got sometimes unexpected error on call site

	if err != nil {
		return nil, errors.Wrap(err, "could not create socks5 dialer")
	}

	var transport http.RoundTripper = &http.Transport{
		Proxy:               nil,
		Dial:                dialer.Dial,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	httpClient := &http.Client{Transport: transport}
	restyClient := resty.NewWithClient(httpClient)
	
	return &Client{
		restyClient,
	}, nil
}
