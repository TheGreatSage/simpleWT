package backend

import (
	"fmt"
	"log"
	"strings"
	"time"

	"capnproto.org/go/capnp/v3"

	"simpleWT/backend/cpnp"
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
	_, ok := DeserializeValid(c.reader, payload, cpnp.ReadRootHeartbeat)
	if !ok {
		log.Printf("Client: Invalid ping")
	}
	c.writer.mu.Lock()
	defer c.writer.mu.Unlock()

	_, seg, err := capnp.NewMessage(capnp.SingleSegment(nil))
	if err != nil {
		return
	}
	msg, err := cpnp.NewRootHeartbeat(seg)
	// msg, err := NewMessage(c.writer, cpnp.NewRootHeartbeat)
	if err != nil {
		log.Printf("Client: Error creating heartbeat: %v\n", err)
		return
	}
	msg.SetUnix(time.Now().UnixMilli())

	_, err = SendStream(c.writer, c.Stream, msg.Message(), OpCodeHeartbeat)
	if err != nil {
		log.Printf("Client: Error sending heartbeat: %v\n", err)
	}
}

// HandleBConnect
// Broadcast OpCodeBConnect
func (c *Client) HandleBConnect(payload []byte) {
	// log.Println("Client: Handling player connected")
	msg, valid := DeserializeValid(c.reader, payload, cpnp.ReadRootGameBroadcastConnect)
	if !valid {
		log.Println("Client: Invalid connect message")
		return
	}
	if !msg.HasPlayer() {
		log.Println("Client: No player in connect message")
		return
	}
	who, err := msg.Player()
	if err != nil {
		log.Printf("Client: Error getting player: %v\n", err)
		return
	}
	if !who.HasId() {
		log.Println("Client: No ID in connect message")
		return
	}
	if !who.HasName() {
		log.Println("Client: No Name in connect message")
		return
	}

	// Commented this out for testing lots of clients.

	// name, err := who.Name()
	// if err != nil {
	// 	log.Printf("Client: Error getting name: %v\n", err)
	// 	return
	// }
	// id, err := who.Id()
	// if err != nil {
	// 	log.Printf("Client: Error getting id: %v\n", err)
	// 	return
	// }
	// conn := "connected"
	// if !msg.Connected() {
	// 	conn = "disconnected"
	// }
	// log.Printf("Client: Player: %s %s. (ID: %s)\n", name, conn, id)
}

// HandleBPlayerMoved
// Broadcast OpCodeBPlayerMoved
func (c *Client) HandleBPlayerMoved(payload []byte) {
	// log.Println("Client: Handling player moved")
	msg, valid := DeserializeValid(c.reader, payload, cpnp.ReadRootGameBroadcastPlayerMove)
	if !valid {
		log.Printf("Client: Invalid message. Len %d\n", len(payload))
		return
	}
	if !msg.HasWho() {
		return
	}
	who, err := msg.Who()
	if err != nil {
		log.Printf("Client: Error getting who: %v\n", err)
		return
	}
	if !who.HasId() {
		return
	}
	// uid, err := who.Id()
	_, err = who.Id()
	if err != nil {
		log.Printf("Client: Error getting id: %v\n", err)
		return
	}
	x := who.X()
	y := who.Y()
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
	msg, valid := DeserializeValid(c.reader, payload, cpnp.ReadRootGameBroadcastChat)
	if !valid {
		log.Printf("Client: Invalid message. Len %d\n", len(payload))
		return
	}
	if !msg.HasName() || !msg.HasText() {
		return
	}

	// name, err := msg.Name()
	// if err != nil {
	// 	log.Printf("Client: Error getting name: %v\n", err)
	// 	return
	// }
	// chat, err := msg.Text()
	// if err != nil {
	// 	log.Printf("Client: Error getting chat: %v\n", err)
	// 	return
	// }

	// Turned off for go clients
	// log.Printf("Client: %s: %s\n", name, chat)
}

func (c *Client) HandleGarbageRequest(payload []byte) {
	// log.Println("Client: Handling garbage request")
	msg, valid := DeserializeValid(c.reader, payload, cpnp.ReadRootGameServerGarbage)
	if !valid {
		log.Printf("Client %s: Invalid garbage. Len %d\n", c.Name, len(payload))
		return
	}
	if !msg.HasBase() {
		log.Println("Client: No base garbage", c.Name)
		c.garbageTicker.Stop()
		return
	}
	base, err := msg.Base()
	if err != nil {
		log.Printf("Client: Error getting base: %v\n", err)
		c.garbageTicker.Stop()
		return
	}

	if msg.Per() == 0 || msg.Amount() == 0 {
		log.Printf("Client: Invalid garbage per second.")
		c.garbageTicker.Stop()
		return
	}

	// log.Printf("Client %s: Garbage needed %d/%ds", c.Name, c.garbageAmount, msg.Per())
	ntime := time.Second / time.Duration(msg.Per())
	if c.garbageTicker == nil {
		c.garbageTicker = time.NewTicker(ntime)
	} else {
		c.garbageTicker.Reset(ntime)
	}
	c.garbageAmount = int(msg.Amount())
	c.garbageBase = base
	c.garbageWait.Store(false)

	// log.Printf("Client %s: Garbage needed %d/%ds", c.Name, c.garbageAmount, msg.Per())
}

func (c *Client) HandleGarbageAck(payload []byte) {
	_, valid := DeserializeValid(c.reader, payload, cpnp.ReadRootGameServerGarbageAck)
	if !valid {
		log.Printf("Client %s: Invalid garbage ack. Len %d\n", c.Name, len(payload))
		return
	}
	c.garbageWait.Store(false)
}

func (c *Client) HandlePlayers(payload []byte) {
	_, valid := DeserializeValid(c.reader, payload, cpnp.ReadRootGameServerPlayers)
	if !valid {
		log.Printf("Client %s: Invalid player list. Len %d\n", c.Name, len(payload))
		return
	}
	// Maybe print amount of players?
}
