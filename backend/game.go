package backend

import (
	"log"
	"math/rand/v2"
	"sync"

	beop "wellquite.org/bebop/runtime"
)

type Player struct {
	Name string
	X, Y int32

	mu sync.Mutex

	GarbageFailed int
	GarbageAmount int      // Amount per message
	GarbageTotal  int      // Amount Total needed
	GarbageBase   [20]byte // SHA1 of time
}

type GameWorld struct {
	Players map[*Session]*Player
	pmu     sync.RWMutex

	db *DatabaseManager

	writer *PacketWriter
	reader *PacketReader

	Closed chan bool
}

func NewGameWorld(db *DatabaseManager) *GameWorld {
	gw := &GameWorld{
		db: db,

		Players: make(map[*Session]*Player),
		// Closed:  make(chan bool, 1),

		writer: NewPacketWriter(),
		reader: NewPacketReader(),
	}
	return gw
}

func (w *GameWorld) Start() {
	w.Closed = make(chan bool, 1)
}

func (w *GameWorld) Shutdown() {
	close(w.Closed)
}

func (w *GameWorld) Connect(session *Session) {
	w.pmu.RLock()
	_, ok := w.Players[session]
	if ok {
		w.pmu.RUnlock()
		log.Println("Player already connected.")
		return
	}
	w.pmu.RUnlock()

	name, err := w.db.GetUserByID(session.ID)
	if err != nil {
		log.Printf("Error getting user by id: %v\n", err)
		return
	}

	// Adds all opcodes for the game
	w.connectOpcodes(session)

	log.Printf("Connecting: %s, ID: %s\n", name, session.ID)

	w.pmu.Lock()
	pl := new(Player)
	pl.Name = name
	pl.X = int32(rand.IntN(100))
	pl.Y = int32(rand.IntN(100))
	w.Players[session] = pl
	w.pmu.Unlock()
	w.playerConnectedSend(session, name, true, true)
	w.sendPlayers(session)
	w.sendGarbage(session, true)
}

func (w *GameWorld) Disconnect(session *Session) {
	player, ok := w.Players[session]
	if !ok {
		return
	}
	delete(w.Players, session)
	w.playerConnectedSend(session, player.Name, false, true)
}

func (w *GameWorld) Reconnect(session *Session) {
	player, ok := w.Players[session]
	if !ok {
		return
	}
	// Only send connect to the one joining
	w.playerConnectedSend(session, player.Name, true, false)
	w.sendPlayers(session)
	w.sendGarbage(session, true)
}

func (w *GameWorld) Broadcast(msg beop.Bebop, opcode uint16) {
	w.pmu.RLock()
	defer w.pmu.RUnlock()
	for s := range w.Players {
		_, _ = SendStream(w.writer, s.stream, msg, opcode)
	}
}
