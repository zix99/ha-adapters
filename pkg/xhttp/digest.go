package xhttp

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync/atomic"
)

// https://stackoverflow.com/questions/39474284/how-do-you-do-a-http-post-with-digest-authentication-in-golang

type HttpDigestSession struct {
	client             XHttp
	authHash           string // aka "ha1"
	username, password string
	nonceCount         uint32
}

func NewDigest(client XHttp, username, password string) *HttpDigestSession {
	return &HttpDigestSession{
		client,
		"",
		username, password,
		0,
	}
}

func (s *HttpDigestSession) Do(req *http.Request) (*http.Response, error) {
	// Initial digest request, expecting 401
	resp0, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp0.Body.Close()

	if resp0.StatusCode != http.StatusUnauthorized {
		return nil, fmt.Errorf("expecting unauthorized, got %d", resp0.StatusCode)
	}

	// Modify request with digest auth
	digest := parseDigest(resp0)
	reqUri := req.URL.RequestURI()

	if s.authHash == "" {
		s.authHash = genMd5(s.username + ":" + digest["realm"] + ":" + s.password)
	}
	ha2 := genMd5(req.Method + ":" + reqUri)
	cnonce := genCnonce()
	nc := atomic.AddUint32(&s.nonceCount, 1)
	ha3 := genMd5(fmt.Sprintf("%s:%s:%08x:%s:%s:%s", s.authHash, digest["nonce"], nc, cnonce, digest["qop"], ha2))
	digestHeader := fmt.Sprintf(`Digest username="%s", realm="%s", nonce="%s", uri="%s", cnonce="%s", nc="%08x", qop="%s", response="%s"`,
		s.username, digest["realm"], digest["nonce"], reqUri, cnonce, nc, digest["qop"], ha3)

	req.Header.Set("Authorization", digestHeader)

	// Make another request with digest auth
	resp1, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	return resp1, nil
}

func parseDigest(resp *http.Response) map[string]string {
	// I dislike this function strongly, but it works
	result := map[string]string{}
	if len(resp.Header["Www-Authenticate"]) > 0 {
		wantedHeaders := []string{"nonce", "realm", "qop"}
		responseHeaders := strings.Split(resp.Header["Www-Authenticate"][0], ",")
		for _, r := range responseHeaders {
			for _, w := range wantedHeaders {
				if strings.Contains(r, w) {
					result[w] = strings.Split(r, `"`)[1]
				}
			}
		}
	}
	return result
}

func genMd5(text string) string {
	hash := md5.Sum([]byte(text))
	return hex.EncodeToString(hash[:])
}

func genCnonce() string {
	var b [8]byte
	rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
