package main

import (
	"log"
	"net"
	"net/url"

	"github.com/gorilla/websocket"
)

////////////////////////////////////////////////////////////////////////////////
// httpClient
////////////////////////////////////////////////////////////////////////////////

// httpClient implements the Runner interface
type httpClient struct {
	connectWS *url.URL
	listenTCP string
}

// NewHTTPClient creates a new TCP server which connects tunnels to an HTTP server
func NewHTTPClient(listenTCP string, connectWS *url.URL) Runner {
	switch connectWS.Scheme {
	case "http":
		connectWS.Scheme = "ws"
	case "https":
		connectWS.Scheme = "wss"
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

	connectWSURL := h.connectWS.String()
	log.Printf("Listening to %s", h.listenTCP)
	for {
		tcpConn, err := tcpConnection.Accept()
		if err == nil {
			log.Printf("%s - Connecting to %s", tcpConn.RemoteAddr(), connectWSURL)
			httpConn, _, err := websocket.DefaultDialer.Dial(connectWSURL, nil)
			if err != nil {
				log.Printf("%s - Error while dialing %s: %s", tcpConn.RemoteAddr(), connectWSURL, err)
				tcpConn.Close()
				continue
			}

			b := NewBidirConnection(tcpConn, httpConn)
			go b.Run()
		} else {
			log.Printf("Error: could not accept the connection: %s", err)
		}
	}
}
