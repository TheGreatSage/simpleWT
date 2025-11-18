package backend

// Wanted to split this up.
// Not sure if respond is the right wording.

import (
	"crypto/sha1"
	"log"
	"math/rand/v2"
	"time"

	"simpleWT/backend/cpnp"
)

func (w *GameWorld) playerConnectedSend(session *Session, name string, connect, broadcast bool) {
	w.writer.mu.Lock()
	defer w.writer.mu.Unlock()
	msg, err := NewMessage(w.writer, cpnp.NewRootGameBroadcastConnect)
	if err != nil {
		log.Printf("Error creating connect packet: %v", err)
		return
	}

	// Is there a way to get rid of creating something new here?
	pl, err := msg.NewPlayer()
	if err != nil {
		log.Printf("Error creating connect packet: %v", err)
		return
	}

	_ = pl.SetId(session.ID.String())
	_ = pl.SetName(name)
	_ = msg.SetPlayer(pl)

	msg.SetConnected(connect)

	if broadcast {
		w.Broadcast(msg.Message(), OpCodeBConnect)
	} else {
		session.writer.mu.Lock()
		_, err = SendStream(w.writer, session.stream, msg.Message(), OpCodeBConnect)
		session.writer.mu.Unlock()
		if err != nil {
			log.Printf("Error sending packet: %v\n", err)
		}
	}
}

func (w *GameWorld) sendPlayers(s *Session) {
	type player struct {
		X, Y int
		Name string
		ID   string
	}
	var players []player
	w.pmu.RLock()
	for s, p := range w.Players {
		// Maybe lock the player?
		players = append(players, player{p.X, p.Y, p.Name, s.ID.String()})
	}
	w.pmu.RUnlock()

	w.writer.mu.Lock()
	defer w.writer.mu.Unlock()

	msg, err := NewMessage(w.writer, cpnp.NewRootGameServerPlayers)
	if err != nil {
		log.Printf("Error sending players packet: %v", err)
		return
	}
	pList, err := msg.NewPlayers(int32(len(players)))
	if err != nil {
		log.Printf("Error sending players packet: %v", err)
		return
	}
	for i := range pList.Len() {
		currPlayer := pList.At(i)
		_ = currPlayer.SetName(players[i].Name)
		currPlayer.SetX(int32(players[i].X))
		currPlayer.SetY(int32(players[i].Y))
		_ = currPlayer.SetId(players[i].ID)
	}
	err = msg.SetPlayers(pList)
	if err != nil {
		log.Printf("Error sending players packet: %v", err)
		return
	}

	_, err = SendStream(w.writer, s.stream, msg.Message(), OpCodeSPlayers)
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
	xx := 0
	yy := 0
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

	msg, err := NewMessage(w.writer, cpnp.NewRootGameBroadcastPlayerMove)
	if err != nil {
		log.Printf("Error creating broadcast packet: %v", err)
		return
	}

	// Way to get rid of this?
	who, err := msg.NewWho()
	if err != nil {
		log.Printf("Error creating broadcast packet: %v", err)
		return
	}

	_ = who.SetId(s.ID.String())
	_ = who.SetName(p.Name)
	who.SetX(int32(p.X))
	who.SetY(int32(p.Y))

	_ = msg.SetWho(who)
	w.Broadcast(msg.Message(), OpCodeBPlayerMoved)
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

	msg, err := NewMessage(w.writer, cpnp.NewRootGameBroadcastChat)
	if err != nil {
		log.Printf("Error creating broadcast packet: %v", err)
		return
	}

	err = msg.SetText(text)
	if err != nil {
		log.Printf("Error creating broadcast packet: %v", err)
		return
	}

	err = msg.SetName(p.Name)
	if err != nil {
		log.Printf("Error creating broadcast packet: %v", err)
		return
	}

	w.Broadcast(msg.Message(), OpCodeBChat)
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

	msg, err := NewMessage(s.writer, cpnp.NewRootGameServerGarbage)
	if err != nil {
		log.Printf("Error making garbage packet: %v", err)
		return
	}
	msg.SetAmount(uint32(p.GarbageAmount))
	// msg.SetSeconds(uint32(sec))
	msg.SetPer(uint8(per))
	// log.Printf("Requesting %s: %d/%ds for %ds total of %d base len %d", p.Name, p.GarbageAmount, per, sec, p.GarbageTotal, len(p.GarbageBase))
	err = msg.SetBase(p.GarbageBase[:])
	if err != nil {
		log.Printf("Error making garbage packet: %v", err)
		return
	}

	_, err = SendStream(s.writer, s.stream, msg.Message(), OpCodeSGarbage)
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

	msg, err := NewMessage(s.writer, cpnp.NewRootGameServerGarbageAck)
	if err != nil {
		log.Printf("Error making garbage ack packet: %v", err)
		return
	}
	// This should probably be its own-received value not the total left.
	msg.SetAck(uint32(gTotal))

	_, err = SendStream(s.writer, s.stream, msg.Message(), OpCodeSGarbageAck)
	if err != nil {
		log.Printf("Error sending garbage ack packet: %v\n", err)
	}
}
