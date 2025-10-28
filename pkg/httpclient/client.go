package httpclient

import "net/http"

func NewHttpClient() *http.Client {
	return &http.Client{}
}
