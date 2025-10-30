package backend

import (
	"log"

	"simpleWT/backend/cpnp"
)

func (s *Session) HandlePong(_ *Session, payload []byte) {
	_, valid := DeserializeValid(s.reader, payload, cpnp.ReadRootHeartbeat)
	if !valid {
		log.Println("Invalid ping.", s.ID)
		return
	}
	// I had this wrong at one point and time was in the realm of 400-900ms on the same machine.
	// I thought I did something super wrong. But nope, just storing the unix time wrong.
	// sent := time.Unix(0, s.lastPing.Load())
	// log.Printf("Pong Took: %s Len: %v\n", time.Since(sent).String(), len(payload))
}
