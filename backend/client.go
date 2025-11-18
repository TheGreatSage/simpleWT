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

	"github.com/quic-go/quic-go"
	"github.com/quic-go/webtransport-go"

	"simpleWT/backend/bebop"
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
	garbageBuffer [254]bebop.GarbageData

	lastRec  atomic.Int64
	lastSent atomic.Int64
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
	d.QUICConfig = &quic.Config{
		EnableDatagrams: true,
	}
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
		snt := time.Since(time.Unix(0, c.lastSent.Load())).String()
		rcv := time.Since(time.Unix(0, c.lastRec.Load())).String()
		log.Printf("Client stream: %v (Sent last: %s, Recv Last: %s)\n", err, snt, rcv)
	}
	c.Stream = nil
	c.Close()
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
			c.lastRec.Store(time.Now().UnixNano())
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
	if c.Closing != nil {
		close(c.Closing)
		c.Closing = nil
	}
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
			c.lastSent.Store(time.Now().UnixNano())
		}
	}
}

func (c *Client) sendGarbage() bool {
	c.writer.mu.Lock()
	defer c.writer.mu.Unlock()

	// Create message
	msg := &bebop.GameClientGarbage{}

	msg.Hashes = c.garbageBuffer[:c.garbageAmount]

	for i := range int32(c.garbageAmount) {
		// Probably wrong way to do hash
		sh := sha1.Sum([]byte(fmt.Sprintf("%s%d", c.garbageBase, i)))

		msg.Hashes[i] = bebop.GarbageData{Data: sh[:]}
	}

	// Write
	_, err := SendStream(c.writer, c.Stream, msg, OpCodeCGarbage)
	if err != nil {
		return false
	}

	// Wait for ack
	c.garbageWait.Store(true)
	return true
}
