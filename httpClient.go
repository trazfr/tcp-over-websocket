package main

import (
	"crypto/tls"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

////////////////////////////////////////////////////////////////////////////////
// httpClient
////////////////////////////////////////////////////////////////////////////////

// httpClient implements the Runner interface
type httpClient struct {
	connectWS string
	listenTCP string
}

// NewHTTPClient creates a new TCP server which connects tunnels to an HTTP server
func NewHTTPClient(listenTCP, connectWS string, insecure bool) Runner {
	if insecure {
		websocket.DefaultDialer.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &httpClient{
		connectWS: connectWS,
		listenTCP: listenTCP,
	}
}

func (h *httpClient) Run() error {
	tcpConnection, err := net.Listen("tcp", h.listenTCP)
	if err != nil {
		return err
	}
	defer tcpConnection.Close()

	log.Printf("Listening to %s", h.listenTCP)
	for {
		tcpConn, err := tcpConnection.Accept()
		if err != nil {
			log.Printf("Error: could not accept the connection: %s", err)
			continue
		}

		wsConn, err := h.createWsConnection(tcpConn.RemoteAddr().String())
		if err != nil || wsConn == nil {
			log.Printf("%s - Error while dialing %s: %s", tcpConn.RemoteAddr(), h.connectWS, err)
			tcpConn.Close()
			continue
		}

		b := NewBidirConnection(tcpConn, wsConn, time.Second*10)
		go b.Run()
	}
}

func (h *httpClient) toWsURL(asString string) (string, error) {
	asURL, err := url.Parse(asString)
	if err != nil {
		return asString, err
	}

	switch asURL.Scheme {
	case "http":
		asURL.Scheme = "ws"
	case "https":
		asURL.Scheme = "wss"
	}
	return asURL.String(), nil
}

func (h *httpClient) createWsConnection(remoteAddr string) (wsConn *websocket.Conn, err error) {
	url := h.connectWS
	for {
		var wsURL string
		wsURL, err = h.toWsURL(url)
		if err != nil {
			return
		}
		log.Printf("%s - Connecting to %s", remoteAddr, wsURL)
		var httpResponse *http.Response
		wsConn, httpResponse, err = websocket.DefaultDialer.Dial(wsURL, nil)
		if httpResponse != nil {
			switch httpResponse.StatusCode {
			case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther, http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
				url = httpResponse.Header.Get("Location")
				log.Printf("%s - Redirect to %s", remoteAddr, url)
				continue
			}
		}
		return
	}
}
