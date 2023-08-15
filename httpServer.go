package main

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	promSubSystem = "http_to_tcp"
)

var (
	promTotalConnections = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: promSubSystem,
		Name:      "connections",
		Help:      "The total number of connections open",
	}, []string{"type"})
	promActiveConnections = promauto.NewGauge(prometheus.GaugeOpts{
		Subsystem: promSubSystem,
		Name:      "active_connections",
		Help:      "The total number of active connections",
	})
	promErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Subsystem: promSubSystem,
		Name:      "error",
		Help:      "The total number of errors",
	}, []string{"type"})
)

////////////////////////////////////////////////////////////////////////////////
// httpServer
////////////////////////////////////////////////////////////////////////////////

// httpServer implements the Runner interface
type httpServer struct {
	wsHandler  wsHandler
	listenHTTP string
	httpMux    *http.ServeMux
}

// NewHTTPServer creates a new websocket server which will wait for clients and open TCP connections
func NewHTTPServer(listenHTTP, connectTCP string, realIPHeader string) Runner {
	result := &httpServer{
		wsHandler: wsHandler{
			connectTCP: connectTCP,
			wsUpgrader: websocket.Upgrader{
				ReadBufferSize:  BufferSize,
				WriteBufferSize: BufferSize,
				CheckOrigin:     func(r *http.Request) bool { return true },
			},
			realIPHeader: realIPHeader,
		},
		listenHTTP: listenHTTP,
		httpMux:    &http.ServeMux{},
	}

	result.httpMux.Handle("/", &result.wsHandler)
	result.httpMux.Handle("/metrics", promhttp.Handler())
	return result
}

func (h *httpServer) Run() error {
	log.Printf("Listening to %s", h.listenHTTP)
	return http.ListenAndServe(h.listenHTTP, h.httpMux)
}

////////////////////////////////////////////////////////////////////////////////
// wsHandler
////////////////////////////////////////////////////////////////////////////////

// wsHandler implements the http.Handler interface
type wsHandler struct {
	connectTCP   string
	wsUpgrader   websocket.Upgrader
	realIPHeader string
}

func (ws *wsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	promActiveConnections.Inc()
	defer promActiveConnections.Dec()
	remoteAddr := r.RemoteAddr
	if len(ws.realIPHeader) > 0 {
		data := r.Header.Get(ws.realIPHeader)
		if parsedIP := net.ParseIP(data); parsedIP != nil {
			remotePort := ""
			if portData := strings.Split(r.RemoteAddr, ":"); len(portData) == 2 {
				remotePort = portData[1]
			} else {
				log.Printf("failed to parse remote port in \"%s\"", r.RemoteAddr)
			}
			remoteAddr = fmt.Sprintf("%s:%s", data, remotePort)
		} else {
			log.Printf("failed to parse data \"%s\" in real ip header %s", data, ws.realIPHeader)
		}
	}

	httpConn, err := ws.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		promErrors.WithLabelValues("upgrade").Inc()
		log.Printf("%s - Error while upgrading: %s", remoteAddr, err)
		return
	}
	promTotalConnections.WithLabelValues("http").Inc()
	log.Printf("%s - Client connected", remoteAddr)

	tcpConn, err := net.Dial("tcp", ws.connectTCP)
	if err != nil {
		promErrors.WithLabelValues("dial_tcp").Inc()
		httpConn.Close()
		log.Printf("%s - Error while dialing %s: %s", remoteAddr, ws.connectTCP, err)
		return
	}

	promTotalConnections.WithLabelValues("tcp").Inc()
	log.Printf("%s - Connected to TCP: %s", remoteAddr, ws.connectTCP)
	NewBidirConnection(tcpConn, httpConn, 0).Run()
	log.Printf("%s - Client disconnected", remoteAddr)
}
