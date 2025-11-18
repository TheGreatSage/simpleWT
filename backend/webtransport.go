package backend

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

const ErrSessionStreamClosed webtransport.StreamErrorCode = 3000

type WebTransportServer struct {
	db *DatabaseManager

	world    *GameWorld
	sessions *SessionManager

	wt  *webtransport.Server
	udp *net.UDPConn
}

func NewWebTransportServer() *WebTransportServer {
	db := NewDatabaseManager()
	world := NewGameWorld(db)

	go world.Start()

	return &WebTransportServer{
		world:    world,
		db:       db,
		sessions: NewSessionManager(),
	}
}

func (s *WebTransportServer) Start() bool {
	if s.wt != nil || s.udp != nil {
		return false
	}
	tlsConfig, err := LoadTLSConfig(8775)
	if err != nil {
		log.Printf("Error loading TLS config: %s\n", err)
		return false
	}

	udpAddr, err := net.ResolveUDPAddr("udp", ":8771")
	if err != nil {
		log.Printf("Error resolving UDP address: %s\n", err)
		return false
	}
	udp, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Printf("Error listening on UDP: %s\n", err)
		return false
	}
	s.udp = udp
	log.Println("Listening on UDP port 8771")

	// QUIC
	// Are these good defaults? Idk.
	quicConfig := &quic.Config{
		MaxStreamReceiveWindow:     4 * 1024 * 1024,
		MaxConnectionReceiveWindow: 4 * 1024 * 1024,
		MaxIncomingStreams:         1024,
	}

	// WebTransportSession Server Setup
	// TODO: CheckOrigin logic should be looked at.
	s.wt = &webtransport.Server{
		H3: http3.Server{
			TLSConfig:       tlsConfig,
			EnableDatagrams: true,
			QUICConfig:      quicConfig,
			Handler:         WithCORS(http.DefaultServeMux),
		},
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// This chain needed?
	chain := Chain{WithCORS}
	http.Handle("/wt", chain.Then(s.handleWT()))

	go func() {
		log.Printf("Starting WebTransport server on port 8771\n")
		err = s.wt.Serve(udp)
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("Error starting WebTransportSession server: %s\n", err)
		}
	}()

	go s.sessions.Run(s.world)

	return true
}

func (s *WebTransportServer) Stop() {
	if s.sessions != nil {
		s.sessions.Shutdown()
		s.sessions = nil
	}

	if s.wt != nil {
		// Not sure this actually gracefully shuts down or not?
		// TODO: Figure out graceful shutdown.
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		// I don't think this is graceful.
		_ = s.wt.H3.Shutdown(ctx)

		// This is just hard stop
		_ = s.wt.Close()
		s.wt = nil
	}

	if s.udp != nil {
		_ = s.udp.Close()
		s.udp = nil
	}

	log.Println("Stopped WebTransportServer")
}

func (s *WebTransportServer) HandleLogin(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	if !query.Has("name") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request %s\n", r.URL.Path)
		return
	}
	name := query.Get("name")
	if name == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request %s\n", r.URL.Path)
		return
	}

	code, err := s.db.Login(name)
	if err != nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request %s\n", r.URL.Path)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, err = w.Write([]byte(code.String()))
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("Error writing response: %s\n", err)
		return
	}
}

func (s *WebTransportServer) verifyWT(w http.ResponseWriter, r *http.Request) (bool, uuid.UUID) {
	query := r.URL.Query()

	// Check for code
	if !query.Has("code") {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request %s\n", r.URL.Path)
		return false, uuid.Nil
	}

	// Code not empty
	code := query.Get("code")
	if code == "" {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request %s\n", r.URL.Path)
		return false, uuid.Nil
	}

	// Actual UUID
	id := uuid.FromStringOrNil(code)
	if id == uuid.Nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request %s\n", id)
		return false, uuid.Nil
	}

	uid, err := s.db.VerifyTransport(id)
	if err != nil || uid == uuid.Nil {
		http.Error(w, "Bad Request", http.StatusBadRequest)
		log.Printf("Bad Request Not verified %s\n", id)
		return false, uuid.Nil
	}

	return true, uid
}

func (s *WebTransportServer) handleWT() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ok, uid := s.verifyWT(w, r)
		if !ok {
			return
		}
		log.Printf("Starting wt request from %s user %s", r.RemoteAddr, uid.String())
		sess, err := s.wt.Upgrade(w, r)
		if err != nil {
			log.Printf("WebTransportSession upgrade error: %v\n", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)

		var session *Session
		clientIP, _, _ := net.SplitHostPort(r.RemoteAddr)
		existing, err := s.sessions.GetValidSession(uid, clientIP)
		if err == nil {
			// s.sessions.
			log.Printf("Reconnecting session %s from %s\n", uid, clientIP)
			session = existing
			// TODO: Test reconnect
			// This doesn't reset the world connection.
			err = session.Reconnect(sess)
			s.world.Reconnect(session)
		}

		if session == nil {
			log.Printf("Creating new session for %s from %s\n", uid, clientIP)
			session = s.sessions.CreateSession(uid, clientIP, sess)
			err = session.Start()
			s.world.Connect(session)
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Handle Session packets
		go session.Run()
	}
}
