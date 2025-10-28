package httpclient

import (
	"fmt"
	"net/http"
	"runtime"

	"github.com/docker/cagent/pkg/version"
)

type userAgentTransport struct {
	agent string
	rt    http.RoundTripper
}

func (u *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r2 := req.Clone(req.Context())
	r2.Header.Set("User-Agent", u.agent)
	return u.rt.RoundTrip(r2)
}

func NewHttpClient() *http.Client {
	return &http.Client{
		Transport: &userAgentTransport{
			agent: fmt.Sprintf("Cagent/%s (%s; %s)", version.Version, runtime.GOOS, runtime.GOARCH),
			rt:    http.DefaultTransport,
		},
	}
}
