package backend

import (
	"crypto/sha1"
	"fmt"
	"log"
	"math/rand/v2"
	"sync"
	"time"

	"capnproto.org/go/capnp/v3"

	"simpleWT/backend/cpnp"
)

type Player struct {
	Name string
	X, Y int

	mu sync.Mutex

	GarbageAmount int // Amount per message
	GarbageTotal  int // Amount Total needed
	GarbageBase   string
}

type GameWorld struct {
	Players map[*Session]*Player
	pmu     sync.RWMutex

	db *DatabaseManager

	writer *PacketWriter
	reader *PacketReader

	// Added to hunt down a client issue.
	broadcastMutex sync.Mutex

	Closed chan bool
}

type GameMessage struct {
	Broadcast bool
	Session   *Session
	Msg       *capnp.Message
	Opcode    uint16
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

	w.playerConnectedSend(true, session.ID.String(), name)
	w.pmu.Lock()
	pl := new(Player)
	pl.Name = name
	w.Players[session] = pl
	w.pmu.Unlock()
	w.sendPlayers(session)
	w.sendGarbage(session)
}

func (w *GameWorld) playerConnectedSend(connect bool, id, name string) {
	w.broadcastMutex.Lock()
	defer w.broadcastMutex.Unlock()
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

	_ = pl.SetId(id)
	_ = pl.SetName(name)
	_ = msg.SetPlayer(pl)

	msg.SetConnected(connect)

	w.Broadcast(msg.Message(), OpCodeBConnect)

}

func (w *GameWorld) Disconnect(session *Session) {
	player, ok := w.Players[session]
	if !ok {
		return
	}
	delete(w.Players, session)
	w.playerConnectedSend(false, session.ID.String(), player.Name)
}

func (w *GameWorld) Broadcast(msg *capnp.Message, opcode uint16) {
	w.pmu.RLock()
	defer w.pmu.RUnlock()
	for s := range w.Players {
		_, _ = SendStream(w.writer, s.stream, msg, opcode)
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
		nx = -100
	} else if nx < -100 {
		nx = 100
	}
	if ny > 100 {
		ny = -100
	} else if ny < -100 {
		ny = 100
	}

	w.Players[s].mu.Lock()
	w.Players[s].X = nx
	w.Players[s].Y = ny
	w.Players[s].mu.Unlock()

	if xx == 0 && yy == 0 {
		return
	}

	w.broadcastMutex.Lock()
	defer w.broadcastMutex.Unlock()
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
	w.broadcastMutex.Lock()
	defer w.broadcastMutex.Unlock()
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

func (w *GameWorld) sendGarbage(s *Session) {
	w.pmu.RLock()
	p, ok := w.Players[s]
	w.pmu.RUnlock()
	if !ok {
		log.Println("Sending garbage player not exists")
		return
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.GarbageTotal > 0 {
		log.Println("Sending garbage still need more")
		return
	}
	s.sendMutex.Lock()
	defer s.sendMutex.Unlock()
	sec := 10 + rand.IntN(10)
	per := rand.IntN(60) + 60
	p.GarbageAmount = rand.IntN(10) + 10
	p.GarbageTotal = per * sec * p.GarbageAmount
	p.GarbageBase = fmt.Sprintf("%s", sha1.Sum([]byte(time.Now().Format(time.RFC3339))))

	msg, err := NewMessage(s.writer, cpnp.NewRootGameServerGarbage)
	if err != nil {
		log.Printf("Error making garbage packet: %v", err)
		return
	}
	msg.SetAmount(uint32(p.GarbageAmount))
	// msg.SetSeconds(uint32(sec))
	msg.SetPer(uint8(per))
	log.Printf("Requesting %s: %d/%ds for %ds total of %d base len %d", p.Name, p.GarbageAmount, per, sec, p.GarbageTotal, len(p.GarbageBase))
	err = msg.SetBase(p.GarbageBase)
	if err != nil {
		log.Printf("Error making garbage packet: %v", err)
		return
	}

	_, err = SendStream(w.writer, s.stream, msg.Message(), OpCodeSGarbage)
	if err != nil {
		log.Printf("Error sending packet: %v\n", err)
	}
}

func (w *GameWorld) sendPlayers(s *Session) {

	type player struct {
		X, Y int
		Name string
	}
	var players []player
	w.pmu.RLock()
	for _, p := range w.Players {
		// Maybe lock the player?
		players = append(players, player{p.X, p.Y, p.Name})
	}
	w.pmu.RUnlock()

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
	for i, p := range players {
		_ = pList.At(i).SetName(p.Name)
		pList.At(i).SetX(int32(p.X))
		pList.At(i).SetY(int32(p.Y))
	}
	err = msg.SetPlayers(pList)
	if err != nil {
		log.Printf("Error sending players packet: %v", err)
		return
	}

	_, err = SendStream(w.writer, s.stream, msg.Message(), OpCodeSPlayers)
	if err != nil {
		log.Printf("Error sending packet: %v\n", err)
	}
}
