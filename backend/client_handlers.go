package backend

import (
	"fmt"
	"log"
	"strings"
	"time"

	"simpleWT/backend/bebop"
)

func (c *Client) setupHandlers() {
	c.AddHandler(OpCodeHeartbeat, c.HandlePing)
	c.AddHandler(OpCodeBConnect, c.HandleBConnect)
	c.AddHandler(OpCodeBPlayerMoved, c.HandleBPlayerMoved)
	c.AddHandler(OpCodeBChat, c.HandleBChat)
	c.AddHandler(OpCodeSGarbage, c.HandleGarbageRequest)
	c.AddHandler(OpCodeSPlayers, c.HandlePlayers)
	c.AddHandler(OpCodeSGarbageAck, c.HandleGarbageAck)
}

// HandlePing
// Utility OpCodeHeartbeat
func (c *Client) HandlePing(payload []byte) {
	// log.Println("Handling ping")
	var msg bebop.Heartbeat
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n == 0 {
		log.Printf("Client: Invalid ping")
		return
	}

	c.writer.mu.Lock()
	defer c.writer.mu.Unlock()

	msg.Unix = time.Now().UnixMilli()
	_, err = SendStream(c.writer, c.Stream, &msg, OpCodeHeartbeat)
	if err != nil {
		log.Printf("Client: Error sending heartbeat: %v\n", err)
	}
}

// HandleBConnect
// Broadcast OpCodeBConnect
func (c *Client) HandleBConnect(payload []byte) {
	// log.Println("Client: Handling player connected")
	var msg bebop.GameBroadcastConnect
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n != msg.SizeBebop() {
		log.Printf("Client: Error getting player: %v\n", err)
		return
	}

	// Commented this out for testing lots of clients.

	// log.Printf("Client: Player: %s %s. (ID: %s)\n", name, conn, id)
}

// HandleBPlayerMoved
// Broadcast OpCodeBPlayerMoved
func (c *Client) HandleBPlayerMoved(payload []byte) {
	// log.Println("Client: Handling player moved")
	var msg bebop.GameBroadcastPlayerMove
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n != msg.SizeBebop() {
		log.Printf("Client: Error getting who: %v\n", err)
		return
	}

	x := msg.Who.X
	y := msg.Who.Y
	xdir := ""
	ydir := ""
	if x > 0 {
		xdir = "East"
	} else if x < 0 {
		xdir = "West"
	}
	if y > 0 {
		ydir = "North"
	} else if y < 0 {
		ydir = "South"
	}
	out := fmt.Sprintf("%s %s", ydir, xdir)
	out = strings.TrimLeft(out, " ")
	// log.Printf("Client: Player (%s): Moved %s", uid, out)
}

// HandleBChat
// Broadcast OpCodeBChat
func (c *Client) HandleBChat(payload []byte) {
	// log.Println("Client: Handling chat")
	var msg bebop.GameClientChat
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n == 0 {
		log.Printf("Client: Error reading chat: %v\n", err)
		return
	}

	// Turned off for go clients
	// log.Printf("Client: %s: %s\n", name, chat)
}

func (c *Client) HandleGarbageRequest(payload []byte) {
	// log.Println("Client: Handling garbage request")
	var msg bebop.GameServerGarbage
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n != msg.SizeBebop() {
		log.Printf("Client: Error getting garbage: %v\n", err)
		return
	}
	if len(msg.Base) == 0 {
		log.Println("Client: No base garbage", c.Name)
		c.garbageTicker.Stop()
		return
	}

	if msg.Per == 0 || msg.Amount == 0 {
		log.Printf("Client: Invalid garbage per second.")
		c.garbageTicker.Stop()
		return
	}

	// log.Printf("Client %s: Garbage needed %d/%ds", c.Name, c.garbageAmount, msg.Per())
	ntime := time.Second / time.Duration(msg.Per)
	if c.garbageTicker == nil {
		c.garbageTicker = time.NewTicker(ntime)
	} else {
		c.garbageTicker.Reset(ntime)
	}
	c.garbageAmount = int(msg.Amount)
	c.garbageBase = msg.Base
	c.garbageWait.Store(false)
	// log.Printf("Client %s: Garbage needed %d/%ds", c.Name, c.garbageAmount, msg.Per())
}

func (c *Client) HandleGarbageAck(payload []byte) {
	var msg bebop.GameServerGarbageAck
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n != msg.SizeBebop() {
		log.Printf("Client: Error getting garbage ack: %v\n", err)
		return
	}
	c.garbageWait.Store(false)
}

func (c *Client) HandlePlayers(payload []byte) {
	var msg bebop.GameServerPlayers
	n, err := msg.UnmarshalBebop(payload)
	if err != nil || n == 0 {
		log.Printf("Client: Error getting players: %v\n", err)
		return
	}
	// Maybe print amount of players?
}
