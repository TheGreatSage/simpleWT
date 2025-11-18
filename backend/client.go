package backend

import (
	"context"
	"crypto/sha1"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/quic-go/webtransport-go"

	"simpleWT/backend/cpnp"
)

// ClientPacketHandlerFunc
// This is a bad way to do this.
// See SessionPacketHandlerFunc
type ClientPacketHandlerFunc func([]byte)

type Client struct {
	Name string

	Sess   *webtransport.Session
	Stream *webtransport.Stream

	handlers map[uint16]ClientPacketHandlerFunc
	incoming chan Packet

	writer *PacketWriter
	reader *PacketReader

	// Close chan
	Closing chan struct{}

	garbageWait   atomic.Bool
	garbageTicker *time.Ticker
	garbageAmount int
	garbageBase   []byte
}

// ClientConnection
// Dummy struct to pass IP and port for connections
// Name is non-nil.
// IP, HTTPPort, WTPort can be null.
//
// Defaults to localhost:8770 and localhost:8771
type ClientConnection struct {
	Name     string
	IP       string
	HTTPPort string
	WTPort   string
}

// ClientConnect
// Creates and connects a client
func ClientConnect(cc ClientConnection) (*Client, error) {
	if cc.Name == "" {
		return nil, errors.New("no client name")
	}
	if cc.HTTPPort == "" {
		cc.HTTPPort = "8770"
	}
	if cc.WTPort == "" {
		cc.WTPort = "8771"
	}
	if cc.IP == "" {
		cc.IP = "127.0.0.1"
	}
	conS := fmt.Sprintf("http://%s:%s/login?name=%s", cc.IP, cc.HTTPPort, url.QueryEscape(cc.Name))
	loginRes, err := http.Get(conS)
	if err != nil {
		return nil, err
	}
	login, err := io.ReadAll(loginRes.Body)
	_ = loginRes.Body.Close()
	if err != nil {
		return nil, err
	}

	if loginRes.StatusCode != http.StatusOK {
		log.Printf("Client Login error: %s", loginRes.Status)
		return nil, errors.New(loginRes.Status)
	}

	// log.Println("Code: ", string(login))

	var headers http.Header
	var d webtransport.Dialer
	// d.QUICConfig.EnableDatagrams = true
	d.TLSClientConfig = &tls.Config{
		InsecureSkipVerify: true,
	}
	conS = fmt.Sprintf("http://%s:%s/wt?code=%s", cc.IP, cc.WTPort, string(login))
	rsp, ses, err := d.Dial(context.Background(), conS, headers)
	if err != nil {
		if rsp != nil {
			_ = rsp.Body.Close()
		}
		log.Fatal(err)
	}
	// log.Println("Status", rsp.StatusCode)
	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("login error: %v", loginRes.Status)
	}

	gtick := time.NewTicker(time.Second)
	gtick.Stop()
	client := &Client{
		Name:          cc.Name,
		Sess:          ses,
		handlers:      make(map[uint16]ClientPacketHandlerFunc),
		writer:        NewPacketWriter(),
		reader:        NewPacketReader(),
		incoming:      make(chan Packet, 1024),
		garbageTicker: gtick,
		Closing:       make(chan struct{}),
	}

	client.garbageWait.Store(false)

	client.setupHandlers()

	go client.HandleStream()
	go client.Run()

	return client, nil
}

// HandleStream
// Accepts a stream from the server and starts reading from it.
func (c *Client) HandleStream() {
	stream, err := c.Sess.AcceptStream(context.Background())
	if err != nil {
		log.Printf("Error accepting stream: %v\n", err)
		return
	}
	if stream == nil {
		log.Printf("Stream is nil\n")
		return
	}
	c.Stream = stream

	err = HandleStream(stream, c.incoming, c.Closing)
	if err != nil {
		log.Printf("Client stream: %v\n", err)
	}
	c.Stream = nil
}

func (c *Client) AddHandler(opcode uint16, handler ClientPacketHandlerFunc) {
	c.handlers[opcode] = handler
}

func (c *Client) Run() {
	go c.runGarbage()
	for {
		select {
		case <-c.Closing:
			return
		case packet := <-c.incoming:
			fun, ok := c.handlers[packet.Header.OpCode]
			if !ok {
				return
			}
			// Not sure if the mutex is needed.
			c.reader.mu.Lock()
			fun(packet.Payload)
			c.reader.mu.Unlock()
		}
	}
}

func (c *Client) Close() {
	close(c.Closing)
}

func (c *Client) runGarbage() {
	for {
	cRunGarbage:
		select {
		case <-c.Closing:
			return
		case <-c.garbageTicker.C:
			// Bad way to do acks
			// you miss a whole tick
			if c.garbageWait.Load() {
				goto cRunGarbage
			}
			// False is an error
			if !c.sendGarbage() {
				c.garbageTicker.Stop()
				goto cRunGarbage
			}
		}
	}
}

func (c *Client) sendGarbage() bool {
	c.writer.mu.Lock()
	defer c.writer.mu.Unlock()

	// Create message
	// msg, err := NewMessage(c.writer, cpnp.NewRootGameClientGarbage)
	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return false
	}
	msg, err := cpnp.NewRootGameClientGarbage(seg)
	if err != nil || !msg.IsValid() || c.garbageAmount == 0 {
		return false
	}

	// Create hashes
	hashes, err := msg.NewHash(int32(c.garbageAmount))
	if err != nil || !hashes.IsValid() {
		return false
	}

	for i := range hashes.Len() {
		garb := hashes.At(i)

		if !garb.IsValid() {
			return false
		}

		// Probably wrong way to do hash
		sh := sha1.Sum([]byte(fmt.Sprintf("%s%d", c.garbageBase, i)))

		// Why does this fail?
		err = garb.SetData(sh[:])
		if err != nil {
			return false
		}

		// _ = hashes.Set(i, garb)
	}

	// Set hashes back
	err = msg.SetHash(hashes)
	if err != nil {
		return false
	}

	// Write
	_, err = SendStream(c.writer, c.Stream, msg.Message(), OpCodeCGarbage)
	if err != nil {
		return false
	}

	// Wait for ack
	c.garbageWait.Store(true)
	return true
}
