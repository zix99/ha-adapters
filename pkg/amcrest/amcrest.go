package amcrest

import (
	"errors"
	"fmt"
	"ha-adapters/pkg/xhttp"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

// Useful for reverse engineering
// https://github.com/tchellomello/python-amcrest/tree/4d0c15af5684edf70383ba5a597e27ff48a0e0d3/src/amcrest

type AmcrestDevice struct {
	url                string
	username, password string
	digestClient       xhttp.XHttp

	SerialNumber    string
	DeviceType      string
	SoftwareVersion string
}

func ConnectAmcrest(url string, username, password string) (*AmcrestDevice, error) {
	var httpClient xhttp.XHttp
	httpClient = &http.Client{
		Timeout: 5 * time.Second,
	}
	httpClient = xhttp.NewDigest(httpClient, username, password)
	httpClient = xhttp.NewAutoRetry(httpClient, 5)

	s := &AmcrestDevice{
		url:          url,
		digestClient: httpClient,
		username:     username,
		password:     password,
	}

	// Static metdata
	var err error
	s.SerialNumber, err = s.magicBox("getSerialNo")
	if err != nil {
		return nil, err
	}

	s.DeviceType, err = s.magicBox("getDeviceType")
	if err != nil {
		return nil, err
	}

	s.SoftwareVersion, err = s.magicBox("getSoftwareVersion")
	if err != nil {
		return nil, err
	}

	// Some validation
	if s.DeviceType != "AD410" {
		return nil, errors.New("expecting ad410")
	}

	return s, nil
}

func (s *AmcrestDevice) GetStorageInfo() (string, error) {
	// Todo: Some better interpretation
	return s.request("/cgi-bin/storageDevice.cgi?action=getDeviceAllInfo")
}

func (s *AmcrestDevice) magicBox(action string) (string, error) {
	ret, err := s.request("/cgi-bin/magicBox.cgi?action=" + action)
	if err != nil {
		return ret, err
	}
	_, val := extractKV(ret)
	return val, nil
}

func (s *AmcrestDevice) request(uri string) (string, error) {
	fullUrl := s.url + uri

	req, err := http.NewRequest(http.MethodGet, fullUrl, nil)
	if err != nil {
		return "", err
	}

	logrus.Debugf("Request %s %s", req.Method, req.URL)

	resp, err := s.digestClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("http error %d", resp.StatusCode)
	}

	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func extractKV(text string) (string, string) {
	text = strings.TrimSpace(text)
	idx := strings.IndexByte(text, '=')
	return text[:idx], text[idx+1:]
}
