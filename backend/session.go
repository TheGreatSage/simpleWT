package backend

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"capnproto.org/go/capnp/v3"
	"github.com/gofrs/uuid/v5"
	"github.com/quic-go/webtransport-go"

	"simpleWT/backend/cpnp"
)

var (
	ErrSessionNotFound      = errors.New("session not found")
	ErrSessionStillActive   = errors.New("session still active")
	ErrSessionInactive      = errors.New("session is inactive")
	ErrSessionIPMismatch    = errors.New("session IP mismatch")
	ErrSessionFailedToStart = errors.New("session failed to start")
)

// SessionPacketHandlerFunc
// This is a bad way to do this.
// Things outside of session need session.
// This should be thought about harder.
type SessionPacketHandlerFunc func(*Session, []byte)

// PingWaitVal 90% of this is used to send pings
const PingWaitVal = 60 * time.Second

type Session struct {
	// User ID
	ID uuid.UUID
	// IP of connection
	IP string

	// Connection active, idea is to be used for reconnects.
	// Not sure if an atomic here is correct
	Active atomic.Bool
	// Only updated on going inactive.
	LastActive time.Time

	stream *webtransport.Stream
	conn   *webtransport.Session

	// This is probably crap
	handlers map[uint16]SessionPacketHandlerFunc
	incoming chan Packet

	// This is all to send msg, should this be per stream?
	writer *PacketWriter
	reader *PacketReader
	// arena  capnp.Arena
	// messageMutex   sync.Mutex
	// writeMsgBuffer *capnp.Message
	// writeBuffer []byte

	// Added to hunt down a client issue.
	// Not sure its needed, maybe just better use of writer.mu?
	sendMutex sync.Mutex

	// Ping info
	PingWait    time.Duration
	PingPeriod  time.Duration
	lastPing    atomic.Int64
	missedPings int

	// Close channel
	// Maybe do a context?
	// Couldn't figure those ou though
	Closing chan struct{}
}

type SessionManager struct {
	sessions map[uuid.UUID]*Session
	mu       sync.RWMutex

	Closing chan struct{}
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[uuid.UUID]*Session),
		Closing:  make(chan struct{}),
	}
}

func (m *SessionManager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()
	close(m.Closing)
	for _, session := range m.sessions {
		_ = session.Close()
	}
}

func (m *SessionManager) CreateSession(id uuid.UUID, ip string, conn *webtransport.Session) *Session {
	m.mu.Lock()
	defer m.mu.Unlock()

	session := &Session{
		ID: id,
		IP: ip,

		LastActive: time.Now(),

		conn: conn,

		handlers: make(map[uint16]SessionPacketHandlerFunc),
		// Size here make sense or should it be open?
		// Could this lock if enough packets get sent enough?
		incoming: make(chan Packet, 1024),

		writer: NewPacketWriter(),
		reader: NewPacketReader(),

		PingWait:   PingWaitVal,
		PingPeriod: (PingWaitVal * 9) / 10,

		// Done in Start
		// Closing: make(chan struct{}),
	}

	session.lastPing.Store(-1)
	session.AddHandler(OpCodeHeartbeat, session.HandlePong)

	m.sessions[id] = session
	return session
}

func (m *SessionManager) GetValidSession(id uuid.UUID, ip string) (*Session, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	session, ok := m.sessions[id]
	if !ok {
		return nil, ErrSessionNotFound
	}
	// Not sure if this is the way to go.
	// This might be bad, but the idea is if the sessions went
	// inactive, but it's still alive then we can turn it back on.
	// I've read WebTransport can change IP, like on Wi-Fi. So this is probably bad.
	if session.IP != ip {
		if session.Active.Load() {
			return nil, ErrSessionIPMismatch
		}
		return session, nil
	}

	return session, nil
}

// Run
// Prunes sessions every minute
func (m *SessionManager) Run() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.Closing:
			return
		case <-ticker.C:
			m.pruneInactive()
		}
	}
}

// pruneInactive
// Removes sessions that have been inactive for 5 minutes
func (m *SessionManager) pruneInactive() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, session := range m.sessions {
		if session.Active.Load() {
			continue
		}
		if session.LastActive.Add(time.Minute * 5).Before(time.Now()) {
			// Save to disk or something here?
			delete(m.sessions, session.ID)
		}
	}
}

func (s *Session) Start() error {
	control, err := s.conn.OpenStream()
	if err != nil {
		// What code?
		_ = s.conn.CloseWithError(500, "Error opening control stream")
		return fmt.Errorf("%w: %w", ErrSessionFailedToStart, err)
	}
	s.stream = control
	s.Active.Store(true)
	s.Closing = make(chan struct{})

	go s.HandleStream(s.stream)
	go s.StartHeartbeat()

	return nil
}

func (s *Session) Reconnect() error {
	// This is probably bad.
	err := s.stream.Close()
	if err != nil {
		return fmt.Errorf("%w: %w", ErrSessionFailedToStart, err)
	}
	return s.Start()
}

func (s *Session) Close() error {
	s.Active.Store(false)
	close(s.Closing)

	var err error
	if s.stream != nil {
		// What codes?
		s.stream.CancelWrite(ErrSessionStreamClosed)
		s.stream.CancelRead(ErrSessionStreamClosed)

		_ = s.stream.Close()
	}

	err = s.conn.CloseWithError(500, "Shutdown")
	if err != nil {
		log.Printf("Error closing session: %v\n", err)
	}

	return err
}

func (s *Session) HandleStream(stream *webtransport.Stream) {
	err := HandleStream(stream, s.incoming, s.Closing)
	if err != nil {
		log.Printf("Error handling stream: %v\n", err)
	}
}

func (s *Session) StartHeartbeat() {
	if s.stream == nil {
		return
	}

	wait := time.NewTicker(s.PingWait)
	ticker := time.NewTicker(s.PingPeriod)
	defer ticker.Stop()
	defer wait.Stop()

	s.lastPing.Store(time.Now().UnixNano())
	err := QueueMessage(s, OpCodeHeartbeat, cpnp.NewRootHeartbeat, nil)
	if err != nil {
		log.Printf("Error heartbeat: %v", err)
	}
	wait.Reset(s.PingWait)
	s.missedPings = 0

	for {
		select {
		case <-s.Closing:
			return
		case <-wait.C:
			if s.missedPings >= 3 {
				// Is this ok?
				_ = s.Close()
				return
			}
			log.Printf("Missed Heartbeat")
			wait.Reset(s.PingWait)
		case <-ticker.C:
			s.lastPing.Store(time.Now().UnixNano())
			err := QueueMessage(s, OpCodeHeartbeat, cpnp.NewRootHeartbeat, nil)
			if err != nil {
				log.Printf("Error heartbeat: %v", err)
			}
			wait.Reset(s.PingWait)
		}
	}
}

func (s *Session) Run() {
	if !s.Active.Load() {
		return
	}
	// Maybe return something?
	for {
		select {
		case <-s.Closing:
			return
		case packet := <-s.incoming:
			fun, ok := s.handlers[packet.Header.OpCode]
			if !ok {
				continue
			}
			fun(s, packet.Payload)
		}
	}
}

func (s *Session) AddHandler(opcode uint16, handler SessionPacketHandlerFunc) {
	s.handlers[opcode] = handler
}

func QueueMessage[T CapnpMessage](s *Session, opcode uint16, ctor func(*capnp.Segment) (T, error), build func(T) error) error {
	// TODO: Move QueueMessage to packet.go
	// Not sure the best way to do that though.
	if !s.Active.Load() {
		return ErrSessionInactive
	}

	s.sendMutex.Lock()
	defer s.sendMutex.Unlock()
	msg, err := NewMessage(s.writer, ctor)
	if err != nil {
		return fmt.Errorf("new message: %w", err)
	}
	if build != nil {
		err = build(msg)
		if err != nil {
			return fmt.Errorf("build message: %w", err)
		}
	}

	// err = s.stream.SetWriteDeadline(time.Now().Add(10 * time.Second))
	// if err != nil {
	// 	log.Printf("Error setting write deadline: %v", err)
	// 	return fmt.Errorf("%w", err)
	// }
	_, err = SendStream(s.writer, s.stream, msg.Message(), opcode)
	return err
}
