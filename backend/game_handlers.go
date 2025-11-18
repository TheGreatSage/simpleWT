package backend

import (
	"bytes"
	"crypto/sha1"
	"errors"
	"fmt"
	"log"

	"simpleWT/backend/cpnp"
)

var (
	ErrGameGarbageInvalid  = errors.New("garbage invalid")
	ErrGameGarbageHash     = errors.New("garbage hash invalid")
	ErrGameGarbageAmount   = errors.New("garbage amount invalid")
	ErrGameGarbageData     = errors.New("garbage data invalid")
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
	msg, valid := DeserializeValid(s.reader, payload, cpnp.ReadRootGameClientChat)
	if !valid {
		return
	}
	if !msg.HasText() {
		return
	}
	txt, err := msg.Text()
	if err != nil {
		return
	}
	w.sendChat(s, txt)
}

func (w *GameWorld) HandleClientMoved(s *Session, payload []byte) {
	msg, valid := DeserializeValid(s.reader, payload, cpnp.ReadRootGameClientMoved)
	if !valid {
		// Print invalid
		return
	}

	w.movePlayer(s, msg.X(), msg.Y())
}

func (w *GameWorld) HandleClientGarbage(s *Session, payload []byte) {
	msg, valid := DeserializeValid(s.reader, payload, cpnp.ReadRootGameClientGarbage)
	if !valid {
		return
	}
	if !msg.HasHash() {
		return
	}
	txt, err := msg.Hash()
	if err != nil {
		return
	}

	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		return
	}

	needNew, err := p.HandleGarbage(txt)
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

func (p *Player) HandleGarbage(hashes cpnp.GarbageData_List) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !hashes.IsValid() {
		log.Println("invalid hashes")
		return false, ErrGameGarbageInvalid
	}

	ln := hashes.Len()
	if ln != p.GarbageAmount {
		log.Printf("Player %s, garbage len %d != %d\n", p.Name, hashes.Len(), p.GarbageAmount)
		return false, ErrGameGarbageAmount
	}

	for i := range ln {
		msg := hashes.At(i)
		if !msg.HasData() {
			return true, ErrGameGarbageData
		}

		data, err := msg.Data()
		if err != nil {
			log.Println("No data", err)
			return true, err
		}
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
