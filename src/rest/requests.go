package rest

import (
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"strings"
	"time"
)

func SendGet(url string) (string, error) {

	transport := &*http.DefaultTransport.(*http.Transport)
	transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	client := &http.Client{
		Transport: transport,
		Timeout:   50 * time.Second,
	}

	res, err := client.Get(url)
	if err != nil {
		return "", errors.New("error sending get: " + err.Error())
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.New("error parsing response body: " + err.Error())
	}

	return string(body), nil
}

func GetPeersNames(endpoint string) ([]string, error) {

	res, err := SendGet(endpoint + "/peers/")
	if err != nil {
		return nil, err
	}

	return strings.Split(res, "\n"), nil
}

func GetPeerAddresses(endpoint string, peerName string) ([]string, error) {
	res, err := SendGet(endpoint + "/peers/" + peerName + "/addresses")
	if err != nil {
		return nil, err
	}

	return strings.Split(res, "\n"), nil
}
