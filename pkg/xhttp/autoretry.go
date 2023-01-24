package xhttp

import (
	"errors"
	"net/http"
)

var ErrorExceedsRetry = errors.New("exceeds retry count")

type AutoRetry struct {
	client      XHttp
	RetryCount  int
	ExpectsCode []int
}

func NewAutoRetry(client XHttp, retryCount int) *AutoRetry {
	return &AutoRetry{
		client,
		retryCount,
		[]int{200},
	}
}

func (s *AutoRetry) Do(req *http.Request) (*http.Response, error) {
	i := 0
	for {
		// FIXME: This code isn't so good, doesn't handle body.close()
		i++

		resp, err := s.client.Do(req)
		if err != nil {
			if i >= s.RetryCount {
				return nil, err
			}
			continue
		}
		if containsInt(s.ExpectsCode, resp.StatusCode) {
			// success!
			return resp, err
		}

		if i >= s.RetryCount {
			return nil, ErrorExceedsRetry
		}
	}
}

func containsInt(arr []int, item int) bool {
	for _, ele := range arr {
		if ele == item {
			return true
		}
	}
	return false
}
