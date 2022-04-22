package websocket

import (
	"net/http"

	"github.com/AdRoll/baker"
	log "github.com/sirupsen/logrus"
	ws "golang.org/x/net/websocket"
)

// Server represents the websocket server.
type Server struct {
	clients   map[int]*client
	addCh     chan *client
	delCh     chan *client
	sendAllCh chan []string
	doneCh    chan bool
	errCh     chan error
	cfg       Conf
}

// Conf holds required configurations that are passed to NewServer to configure the Server
type Conf struct {
	Fields      []baker.FieldIndex
	FieldByName func(string) (baker.FieldIndex, bool)
}

// NewServer creates new chat server.
func NewServer(c Conf) *Server {
	return &Server{
		clients:   make(map[int]*client),
		addCh:     make(chan *client),
		delCh:     make(chan *client),
		sendAllCh: make(chan []string),
		doneCh:    make(chan bool),
		errCh:     make(chan error),
		cfg:       c,
	}
}

func (s *Server) add(c *client) {
	s.addCh <- c
}

func (s *Server) del(c *client) {
	s.delCh <- c
}

// SendAll sends a message to all connected clients. Used by the output component to broadcast
// the message to the clients
func (s *Server) SendAll(msg []string) {
	s.sendAllCh <- msg
}

func (s *Server) err(err error) {
	s.errCh <- err
}

func (s *Server) sendAll(msg []string) {
	for _, c := range s.clients {
		c.Write(msg)
	}
}

// Listen and serve.
// It serves client connection and broadcast request.
func (s *Server) Listen() {

	log.Info("Listening ws server...")

	// websocket handler
	onConnected := func(ws *ws.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		log.WithFields(log.Fields{"query": ws.Request().URL.Query()}).Info("Received args")
		client := newClient(ws, s)
		s.add(client)
		client.Listen()
	}

	http.Handle("/subscribe", ws.Handler(onConnected))

	log.Info("Created handler")

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			s.clients[c.id] = c
			log.WithField("# clients", len(s.clients)).Info("New Connection")

		// del a client
		case c := <-s.delCh:
			log.WithField("client", c).Info("Delete client")
			delete(s.clients, c.id)

		// broadcast Record for all clients
		case msg := <-s.sendAllCh:
			s.sendAll(msg)

		case err := <-s.errCh:
			log.WithError(err).Info("Error")

		case <-s.doneCh:
			return
		}
	}
}
