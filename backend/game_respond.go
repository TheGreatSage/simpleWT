package backend

// Wanted to split this up.
// Not sure if respond is the right wording.

import (
	"crypto/sha1"
	"log"
	"math/rand/v2"
	"time"

	"simpleWT/backend/bebop"
)

func (w *GameWorld) playerConnectedSend(session *Session, name string, connect, broadcast bool) {
	w.writer.mu.Lock()
	defer w.writer.mu.Unlock()

	// Is there a way to get rid of creating something new here?
	msg := &bebop.GameBroadcastConnect{
		Player: bebop.Player{
			ID:   session.ID.String(),
			Name: name,
		},
		Connected: connect,
	}

	if broadcast {
		w.Broadcast(msg, OpCodeBConnect)
	} else {
		session.writer.mu.Lock()
		_, err := SendStream(w.writer, session.stream, msg, OpCodeBConnect)
		session.writer.mu.Unlock()
		if err != nil {
			log.Printf("Error sending packet: %v\n", err)
		}
	}
}

func (w *GameWorld) sendPlayers(s *Session) {

	var players []bebop.Player
	w.pmu.RLock()
	for s, p := range w.Players {
		// Maybe lock the player?
		players = append(players, bebop.Player{X: p.X, Y: p.Y, Name: p.Name, ID: s.ID.String()})
	}
	w.pmu.RUnlock()

	w.writer.mu.Lock()
	defer w.writer.mu.Unlock()

	msg := &bebop.GameServerPlayers{
		Players: players,
	}

	_, err := SendStream(w.writer, s.stream, msg, OpCodeSPlayers)
	if err != nil {
		log.Printf("Error sending players packet: %v\n", err)
	}
}

func (w *GameWorld) movePlayer(s *Session, x, y int8) {
	if s == nil {
		return
	}
	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		log.Println("Player not found")
		return
	}
	// Probably a better way to do this math.
	// I'm tired though, and it's not a big deal.
	xx := int32(0)
	yy := int32(0)
	if x > 0 {
		xx = 1
	} else if x < 0 {
		xx = -1
	}
	if y > 0 {
		yy = 1
	} else if y < 0 {
		yy = -1
	}
	nx := p.X + xx
	ny := p.Y + yy
	if nx > 100 {
		nx = 0
	} else if nx < 0 {
		nx = 100
	}
	if ny > 100 {
		ny = 0
	} else if ny < 0 {
		ny = 100
	}

	w.Players[s].mu.Lock()
	w.Players[s].X = nx
	w.Players[s].Y = ny
	w.Players[s].mu.Unlock()

	if xx == 0 && yy == 0 {
		log.Println("Player not moving")
		return
	}

	// Broadcast so lock the gameworld writer
	w.writer.mu.Lock()
	defer w.writer.mu.Unlock()

	msg := &bebop.GameBroadcastPlayerMove{
		Who: bebop.Player{
			ID:   s.ID.String(),
			Name: p.Name,
			X:    p.X,
			Y:    p.Y,
		},
	}

	w.Broadcast(msg, OpCodeBPlayerMoved)
}

func (w *GameWorld) sendChat(s *Session, text string) {
	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		return
	}

	// Broadcast so lock the game world writer
	w.writer.mu.Lock()
	defer w.writer.mu.Unlock()

	msg := &bebop.GameBroadcastChat{
		Text: text,
		Name: p.Name,
	}

	w.Broadcast(msg, OpCodeBChat)
}

func (w *GameWorld) sendGarbage(s *Session, reset bool) {
	// Read the player list
	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		log.Println("Sending garbage player not exists")
		return
	}

	// Lock the player
	// Really should make garbage its own thing
	p.mu.Lock()
	defer p.mu.Unlock()

	// Reset for failed messages
	if p.GarbageTotal > 0 && !reset {
		log.Println("Sending garbage still need more")
		return
	}

	// Creating a random amount of garbage
	sec := 10 + rand.IntN(10)
	per := rand.IntN(60) + 60
	p.GarbageAmount = rand.IntN(10) + 10
	p.GarbageTotal = per * sec * p.GarbageAmount
	p.GarbageBase = sha1.Sum([]byte(time.Now().Format(time.RFC3339)))

	// Lock the writer
	s.writer.mu.Lock()
	defer s.writer.mu.Unlock()

	msg := &bebop.GameServerGarbage{
		Amount: uint32(p.GarbageAmount),
		Per:    uint8(per),
		Base:   p.GarbageBase[:],
	}

	// log.Printf("Requesting %s: %d/%ds for %ds total of %d base len %d", p.Name, p.GarbageAmount, per, sec, p.GarbageTotal, len(p.GarbageBase))

	_, err := SendStream(s.writer, s.stream, msg, OpCodeSGarbage)
	if err != nil {
		log.Printf("Error sending garbage packet: %v\n", err)
	}
}

func (w *GameWorld) sendGarbageAck(s *Session) {
	// Read player list
	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		log.Println("Sending garbage player not exists")
		return
	}

	// Read player
	p.mu.Lock()
	gTotal := p.GarbageTotal
	p.mu.Unlock()

	// Lock the writer
	s.writer.mu.Lock()
	defer s.writer.mu.Unlock()

	// This should probably be its own-received value not the total left.
	msg := &bebop.GameServerGarbageAck{
		Ack: uint32(gTotal),
	}

	_, err := SendStream(s.writer, s.stream, msg, OpCodeSGarbageAck)
	if err != nil {
		log.Printf("Error sending garbage ack packet: %v\n", err)
	}
}
