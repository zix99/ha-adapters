package xhttp

import (
	"errors"
	"net/http"
	"time"
)

var ErrorExceedsRetry = errors.New("exceeds retry count")

type AutoRetry struct {
	client      XHttp
	RetryCount  int
	Delay       time.Duration
	ExpectsCode []int
}

func NewAutoRetry(client XHttp, retryCount int) *AutoRetry {
	return &AutoRetry{
		client,
		retryCount,
		500 * time.Millisecond,
		[]int{200},
	}
}

func (s *AutoRetry) Do(req *http.Request) (*http.Response, error) {
	i := 0
	for {
		i++

		resp, err := s.client.Do(req)
		if err != nil {
			if i >= s.RetryCount {
				return nil, err
			}
			time.Sleep(s.Delay)
			continue
		}
		if containsInt(s.ExpectsCode, resp.StatusCode) {
			// success!
			return resp, err
		}

		// Didn't get the result we expected, cleanup and try again
		resp.Body.Close()

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
