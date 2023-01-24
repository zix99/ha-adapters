package xhttp

import "net/http"

type XHttp interface {
	Do(req *http.Request) (resp *http.Response, err error)
}
