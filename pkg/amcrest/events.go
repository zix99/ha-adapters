package amcrest

import (
	"bytes"
	"ha-adapters/pkg/xhttp"
	"mime"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

type Event struct {
	Err          error
	Code, Action string
	Index        int
	Data         string
}

func (s *AmcrestDevice) OpenReliableEventStream(maxSequentialRetries int) <-chan Event {
	c := make(chan Event, 10)
	go func() {
		defer close(c)
		for retries := 0; retries < maxSequentialRetries; retries++ {
			stream, err := s.OpenEventStream()
			if err != nil {
				logrus.Warnf("Error opening stream: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}

			retries = 0 // reset! Success!

			for event := range stream {
				c <- event
			}
		}
	}()
	return c
}

func (s *AmcrestDevice) OpenEventStream() (<-chan Event, error) {
	/*
		Stream is a long-open multipart HTTP stream with a data-like object that
		needs custom parsing
	*/
	logrus.Info("Opening event stream...")

	url := s.url + "/cgi-bin/eventManager.cgi?action=attach&codes=[All]"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	lhttp := s.buildLongHttp()

	resp, err := lhttp.Do(req)
	if err != nil {
		return nil, err
	}

	// Start read loop

	logrus.Info("Connection open, listening to stream...")
	c := make(chan Event, 10)

	_, contentParams, err := mime.ParseMediaType(resp.Header.Get("content-type"))
	if err != nil {
		resp.Body.Close()
		return nil, err
	}
	boundaryKeyword := contentParams["boundary"]

	go func() {
		defer resp.Body.Close()
		defer close(c)
		mp := multipart.NewReader(resp.Body, boundaryKeyword)

		for {
			part, err := mp.NextPart()
			if err != nil {
				break
			}

			datalen, err := strconv.Atoi(part.Header.Get("content-length"))
			if err != nil {
				logrus.Warnf("Error reading stream length: %v", err)
				part.Close()
				continue
			}

			data := make([]byte, datalen)
			_, err = part.Read(data)
			if err != nil {
				logrus.Warnf("Error reading stream data: %v", err)
				part.Close()
				continue
			}
			logrus.Debugf("Received %d bytes: %s", len(data), string(data))
			c <- payloadToEvent(data)
			part.Close()
		}

		logrus.Info("Closing event stream...")
	}()

	return c, nil
}

func (s *AmcrestDevice) buildLongHttp() xhttp.XHttp {
	var longhttp xhttp.XHttp
	longhttp = &http.Client{
		Timeout: 1 * time.Hour,
	}
	longhttp = xhttp.NewDigest(longhttp, s.username, s.password)
	return longhttp
}

func payloadToEvent(payload []byte) (ret Event) {
	bucket := parseStreamPayload(payload)

	index, _ := strconv.Atoi(bucket["index"])
	return Event{
		Code:   bucket["code"],
		Action: bucket["action"],
		Index:  index,
		Data:   bucket["data"],
	}
}

func parseStreamPayload(payload []byte) (ret map[string]string) {
	ret = make(map[string]string)

	for len(payload) > 0 {
		nextToken := bytes.IndexByte(payload, ';')
		var slice []byte
		if nextToken < 0 {
			slice = payload
		} else {
			slice = payload[:nextToken]
		}

		if delimIndex := bytes.IndexByte(slice, '='); delimIndex > 0 {
			key := strings.ToLower(string(slice[:delimIndex]))
			val := string(slice[delimIndex+1:])
			ret[key] = val
		}

		if nextToken < 0 {
			break
		}

		payload = payload[len(slice)+1:]
	}

	return
}
