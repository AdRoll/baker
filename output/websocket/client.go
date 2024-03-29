package websocket

import (
	"errors"
	"io"

	log "github.com/sirupsen/logrus"
	ws "golang.org/x/net/websocket"
)

const channelBufSize = 100

var maxId int = 0

// Websocket client.
type client struct {
	id      int
	ws      *ws.Conn
	server  *Server
	filters map[int][]string
	ch      chan []string
	doneCh  chan bool
}

func contains(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

func getFieldIdInReturn(name string, cfg Conf) (int, error) {
	idx, _ := cfg.FieldByName(name)
	for i, v := range cfg.Fields {
		if idx == v {
			return i, nil
		}
	}
	return 0, errors.New("no Field")
}

// Create new ws client.
func newClient(ws *ws.Conn, server *Server) *client {
	maxId++
	ch := make(chan []string, channelBufSize)
	doneCh := make(chan bool)

	filters := make(map[int][]string)
	for k, v := range ws.Request().URL.Query() {
		if k == "fields" {
			// This is to select specific fields
			// For now skip
			continue
		}
		fieldId, err := getFieldIdInReturn(k, server.cfg)
		if err != nil {
			continue
		}
		filters[fieldId] = v
	}

	return &client{
		id:      maxId,
		ws:      ws,
		server:  server,
		filters: filters,
		ch:      ch,
		doneCh:  doneCh,
	}
}

func (c *client) shouldSend(msg []string) bool {
	if len(c.filters) == 0 {
		return true
	}
	for k, vs := range c.filters {
		// choice of implementation here is that it's all an OR
		// so any matching filter will pass the line, in reality
		// you might want different setups, for example ORs in the
		// same field is the only thing that makes sense of course
		// but among different ones you might want an AND.
		if contains(vs, msg[k]) {
			return true
		}
	}
	return false

}

func (c *client) Conn() *ws.Conn {
	return c.ws
}

func (c *client) Write(msg []string) {
	select {
	case c.ch <- msg:
	default:
	}
}

func (c *client) Done() {
	close(c.doneCh)
}

// Listen Write and Read request via channel
func (c *client) Listen() {
	go c.listenWrite()
	c.listenRead()
}

// Listen write request via chanel
func (c *client) listenWrite() {
	ctx := log.WithFields(log.Fields{"client": c, "fn": "listenWrite"})

	ctx.Info("Listening write to client")
	for {
		select {

		// send message to the client
		case msg := <-c.ch:
			if c.shouldSend(msg) {
				err := ws.JSON.Send(c.ws, msg)
				if err == io.EOF {
					ctx.Info("Terminating")
					c.Done()
					return
				}
			}

		// receive done request
		case <-c.doneCh:
			ctx.Info("Terminating")
			c.server.del(c)
			return
		}
	}
}

// Listen read request via chanel
func (c *client) listenRead() {
	ctx := log.WithFields(log.Fields{"client": c, "fn": "listenRead"})
	ctx.Info("Listening read from client")
	for {
		select {

		// receive done request
		case <-c.doneCh:
			ctx.Info("Terminating")
			c.server.del(c)
			return

		// read data from websocket connection
		default:
			var msg map[string]string
			err := ws.JSON.Receive(c.ws, &msg)
			if err == io.EOF {
				c.Done()
			} else if err != nil {
				c.server.err(err)
			} else {
				ctx.WithField("msg", msg).Info("Received")
			}
		}
	}
}
