package websocket

import (
	"net/http"

	"github.com/AdRoll/baker"
	"github.com/AdRoll/baker/logger"
	ws "golang.org/x/net/websocket"
)

// Chat server.
type Server struct {
	clients   map[int]*client
	addCh     chan *client
	delCh     chan *client
	sendAllCh chan []string
	doneCh    chan bool
	errCh     chan error
	cfg       Conf
}

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

func (s *Server) SendAll(msg []string) {
	s.sendAllCh <- msg
}

func (s *Server) done() {
	close(s.doneCh)
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

	logger.Log.Info("Listening server...")

	// websocket handler
	onConnected := func(ws *ws.Conn) {
		defer func() {
			err := ws.Close()
			if err != nil {
				s.errCh <- err
			}
		}()

		logger.Log.Info("Received args. query=%v", ws.Request().URL.Query())
		client := newClient(ws, s)
		s.add(client)
		client.Listen()
	}

	http.Handle("/subscribe", ws.Handler(onConnected))

	logger.Log.Info("Created handler")

	for {
		select {

		// Add new a client
		case c := <-s.addCh:
			s.clients[c.id] = c
			logger.Log.Info("New Connection, # clients=", len(s.clients))

		// del a client
		case c := <-s.delCh:
			logger.Log.Info("Delete client. client=", c)
			delete(s.clients, c.id)

		// broadcast LogLine for all clients
		case msg := <-s.sendAllCh:
			s.sendAll(msg)

		case err := <-s.errCh:
			logger.Log.Info(err)

		case <-s.doneCh:
			return
		}
	}
}
