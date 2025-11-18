package backend

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"sync"

	beop "wellquite.org/bebop/runtime"

	"github.com/quic-go/webtransport-go"

	"simpleWT/backend/bebop"
)

// OpCodes
const (
	_ uint16 = iota

	// Utility Opcodes
	_
	OpCodeHeartbeat

	// Game Server Broadcast Opcodes
	_
	OpCodeBConnect
	OpCodeBPlayerMoved
	OpCodeBChat

	// Game Server Opcodes
	_
	OpCodeSGarbage
	OpCodeSGarbageAck
	OpCodeSPlayers

	// Game Client Opcodes
	_
	OpCodeCChat
	OpCodeCMoved
	OpCodeCGarbage
)

var (
	ErrStreamNil           = errors.New("stream is nil")
	ErrStreamReading       = errors.New("stream reading error")
	ErrStreamHeaderLength  = errors.New("stream malformed header")
	ErrStreamPayloadLength = errors.New("stream malformed payload")
)

// PacketHeader
// Info about the payload
type PacketHeader struct {
	// OpCode packet type.
	OpCode uint16
	// Length of payload
	Length uint32
}

// Instead of OpCodes maybe using capnp IDs?
// they are int64s which would add a bit to the length of the header
// You would probably want to round the header up to 64x2.
// Probably not worth it.

const PacketHeaderLength = 6

type Packet struct {
	Header  PacketHeader
	Payload []byte
}

// Using a struct might be the wrong call. No idea.
// It is handy though

type PacketHandler interface {
	HandlePacket(PacketHeader, []byte)
}

const PacketBufferSize = 8

// This might be a bad way to do this

// PacketWriter
// A way to create capnp messages
type PacketWriter struct {
	mu  sync.Mutex
	buf []byte
}

// PacketWriteSender
// Probably a dumb way to get the write buffer
type PacketWriteSender interface {
	GetWriteBuffer() []byte
	Expand(int)
}

func (p *PacketWriter) GetWriteBuffer() []byte {
	return p.buf[:cap(p.buf)]
}

func (p *PacketWriter) Expand(size int) {
	p.buf = make([]byte, size)
}

// PacketReader
// For turning buffers into messages.
type PacketReader struct {
	mu  sync.Mutex
	buf []byte
}

func NewPacketWriter() *PacketWriter {
	return &PacketWriter{
		buf: make([]byte, PacketBufferSize),
	}
}

func NewPacketReader() *PacketReader {
	return &PacketReader{
		buf: make([]byte, PacketBufferSize),
	}
}

// SendStream
// Writes a msg to a stream
func SendStream(pk PacketWriteSender, stream io.Writer, msg beop.Bebop, opcode uint16) (int, error) {
	if stream == nil {
		return 0, nil
	}

	buf := pk.GetWriteBuffer()
	payload := buf[PacketHeaderLength:]

	_, err := msg.MarshalBebop(payload)
	if err != nil {
		if errors.Is(err, bebop.ErrBufferTooSmall) {
			pk.Expand(PacketHeaderLength + msg.SizeBebop() + 2)
			return SendStream(pk, stream, msg, opcode)
		}
		return 0, fmt.Errorf("send stream %w", err)
	}

	n := msg.SizeBebop()
	binary.LittleEndian.PutUint16(buf[0:2], opcode)
	binary.LittleEndian.PutUint32(buf[2:PacketHeaderLength], uint32(n))

	// if n > 1024*8 {
	// 	log.Println("packet too big", n)
	// 	return 0, fmt.Errorf("packet too big: %d", n)
	// }
	return stream.Write(buf[:n+PacketHeaderLength])
}

// HandleStream
// Reads from a stream until context is closed. Or a read error.
// Checks for webtransport.StreamError closes too.
func HandleStream(stream io.ReadWriteCloser, handler chan<- Packet, closing <-chan struct{}) error {
	if stream == nil {
		return ErrStreamNil
	}

	defer func() {
		err := stream.Close()
		if err != nil {
			// I'd probably log this but gets annoying on the clients.
			// log.Printf("stream close: %v\n", err)
		}
	}()

	// TODO: find a way to remove this StreamError reference.
	var streamErr *webtransport.StreamError
	var header PacketHeader
	var headBuf [PacketHeaderLength]byte
	// Not super happy with this read loop.
	// Feels like it's bad.
	// And could break super easy.
	for {
		// Better way to close this?
		// I don't understand contexts
		select {
		case <-closing:
			return nil
		default:
			break
		}

		// Binary.Read allocs so gotta use io.ReadFull
		// err := binary.Read(stream, binary.LittleEndian, &header)
		n, err := io.ReadFull(stream, headBuf[:])
		if err != nil {
			// This is hideous. Don't like how webtransport handles this.
			if errors.As(err, &streamErr) && streamErr.ErrorCode == ErrSessionStreamClosed {
				return nil
			}
			// Better option than killing the stream?
			return fmt.Errorf("header %w, %w", ErrStreamReading, err)
		}
		// Should have read the whole thing
		if n != PacketHeaderLength {
			return ErrStreamHeaderLength
		}
		header.OpCode = binary.LittleEndian.Uint16(headBuf[:2])
		header.Length = binary.LittleEndian.Uint32(headBuf[2:6])

		// Read the payload.
		// I wonder if there is a better way than alloc per packet?
		// Maybe a ring buffer or something? This is fine for now though.
		payload := make([]byte, header.Length)
		n, err = io.ReadFull(stream, payload)
		if err != nil {
			if errors.As(err, &streamErr) && streamErr.ErrorCode == ErrSessionStreamClosed {
				return nil
			}
			// Better option here?
			return fmt.Errorf("payload %w: %w", ErrStreamReading, err)
		}
		// Double check that we read the same amount as the header.
		// Probably not needed. Not sure.
		if n != int(header.Length) {
			return ErrStreamPayloadLength
		}

		// This maybe could be better. Should rethink this.
		// handler.HandlePacket(header, payload)
		handler <- Packet{header, payload}
	}
}
