package backend

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"

	"simpleWT/backend/bebop"
)

var (
	ErrGameGarbageHash     = errors.New("garbage hash invalid")
	ErrGameGarbageAmount   = errors.New("garbage amount invalid")
	ErrGameGarbageMismatch = errors.New("garbage mismatch")
)

func (w *GameWorld) connectOpcodes(s *Session) {
	if s == nil {
		return
	}
	s.AddHandler(OpCodeCChat, w.HandleClientChat)
	s.AddHandler(OpCodeCMoved, w.HandleClientMoved)
	s.AddHandler(OpCodeCGarbage, w.HandleClientGarbage)
}

func (w *GameWorld) HandleClientChat(s *Session, payload []byte) {
	var msg bebop.GameClientChat
	_, err := msg.UnmarshalBebop(payload)
	if err != nil {
		return
	}
	w.sendChat(s, msg.Text)
}

func (w *GameWorld) HandleClientMoved(s *Session, payload []byte) {
	var msg bebop.GameClientMoved
	_, err := msg.UnmarshalBebop(payload)
	if err != nil {
		// Print invalid
		return
	}

	w.movePlayer(s, int8(msg.X), int8(msg.Y))
}

func (w *GameWorld) HandleClientGarbage(s *Session, payload []byte) {
	var msg bebop.GameClientGarbage
	_, err := msg.UnmarshalBebop(payload)
	if err != nil {
		return
	}

	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		return
	}

	needNew, err := p.HandleGarbage(msg.Hashes)
	if err != nil {
		log.Println("errored", err)
		p.mu.Lock()
		p.GarbageFailed++
		if p.GarbageFailed > 5 {
			log.Println("Failed 5 garbage requests, closing session.")
			_ = s.Close()
		}
		p.mu.Unlock()
		if needNew {
			w.sendGarbage(s, true)
		}
		return
	}
	p.mu.Lock()
	p.GarbageFailed = 0
	p.mu.Unlock()
	if needNew {
		w.sendGarbage(s, false)
	} else {
		w.sendGarbageAck(s)
	}
}

func (p *Player) HandleGarbage(hashes []bebop.GarbageData) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	ln := len(hashes)
	if ln != p.GarbageAmount {
		log.Printf("Player %s, garbage len %d != %d\n", p.Name, ln, p.GarbageAmount)
		return false, ErrGameGarbageAmount
	}

	for i := range ln {
		data := hashes[i].Data

		// A sha1 hash is of len 20
		if len(data) != 20 {
			log.Println("hash length != 20", len(data))
			return true, ErrGameGarbageHash
		}

		// This is probably a horrible way to do this.
		cur := fmt.Sprintf("%s%d", p.GarbageBase, i)
		hash := sha1.Sum([]byte(cur))
		if bytes.Equal(hash[:], data[:]) {
			p.GarbageTotal--
		} else {
			return true, ErrGameGarbageMismatch
		}

	}

	if p.GarbageTotal <= 0 {
		return true, nil
	}

	return false, nil
}
