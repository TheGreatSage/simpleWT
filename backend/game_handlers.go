package backend

import (
	"crypto/sha1"
	"fmt"
	"log"

	"capnproto.org/go/capnp/v3"

	"simpleWT/backend/cpnp"
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
	if !msg.HasText() {
		return
	}
	txt, err := msg.Text()
	if err != nil {
		return
	}

	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		return
	}
	if p.HandleGarbage(txt) {
		w.sendGarbage(s)
	}
}

func (p *Player) HandleGarbage(text capnp.TextList) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !text.IsValid() {
		log.Println("invalid text")
		return false
	}

	ln := text.Len()
	if ln != p.GarbageAmount {
		log.Printf("Player %s, garbage len %d != %d\n", p.Name, text.Len(), p.GarbageAmount)
		return false
	}

	for i := range ln {
		if ln != text.Len() {
			log.Println("What just happened?")
		}
		msg, err := text.At(i)
		if err != nil {
			continue
		}
		// This is probably a horrible way to do this.
		cur := fmt.Sprintf("%s%d", p.GarbageBase, i)
		if fmt.Sprintf("%s", sha1.Sum([]byte(cur))) == msg {
			p.GarbageTotal--
		}

	}

	if p.GarbageTotal <= 0 {
		return true
	}

	return false
}
