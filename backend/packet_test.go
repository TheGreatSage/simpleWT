package backend

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"capnproto.org/go/capnp/v3"
	"github.com/go-faker/faker/v4"

	"simpleWT/backend/capnext"
	"simpleWT/backend/cpnp"
)

func TestNewMessage(t *testing.T) {
	writer := NewPacketWriter()

	_ = testMsg(t, writer)
}

func BenchmarkNewMessage(b *testing.B) {
	writer := NewPacketWriter()

	for b.Loop() {
		_ = testMsg(b, writer)
	}
}

func TestSendStream(t *testing.T) {
	writer := NewPacketWriter()
	reader := NewPacketReader()

	buffer := new(bytes.Buffer)

	name := sendMsg(t, writer, buffer)

	verifyMsg(t, reader, buffer, name)

}

func testMsg(tb testing.TB, writer *PacketWriter) cpnp.GameBroadcastConnect {
	msg, err := NewMessage(writer, cpnp.NewRootGameBroadcastConnect)
	if err != nil {
		tb.Fatal(err)
	}

	player, err := msg.NewPlayer()
	if err != nil {
		tb.Fatal(err)
	}

	err = player.SetName(faker.Name())
	if err != nil {
		tb.Fatal(err)
	}

	err = player.SetId(faker.UUIDDigit())
	if err != nil {
		tb.Fatal(err)
	}

	err = msg.SetPlayer(player)
	if err != nil {
		tb.Fatal(err)
	}

	msg.SetConnected(true)

	if !msg.IsValid() {
		tb.Fatal("message is not valid")
	}

	return msg
}

func sendMsg(tb testing.TB, writer *PacketWriter, buffer *bytes.Buffer) string {
	msg := testMsg(tb, writer)

	n, err := SendStream(writer, buffer, msg.Message(), OpCodeBConnect)
	if err != nil {
		tb.Fatal(err)
	}

	if n == 0 {
		tb.Fatal("no message wrote")
	}

	if !msg.HasPlayer() {
		tb.Fatal("no player wrote")
	}

	player, err := msg.Player()
	if err != nil {
		tb.Fatal(err)
	}

	name, err := player.Name()
	if err != nil {
		tb.Fatal(err)
	}
	return name
}

func verifyMsg(tb testing.TB, reader *PacketReader, buffer *bytes.Buffer, name string) {

	var header PacketHeader
	var hbuffer [PacketHeaderLength]byte
	// Binary read allocs
	// err := binary.Read(buffer, binary.LittleEndian, &header)
	n, err := io.ReadFull(buffer, hbuffer[:])
	if err != nil {
		tb.Fatal(err)
	}
	if n != PacketHeaderLength {
		tb.Fatalf("expected %d bytes, got %d", PacketHeaderLength, n)
	}
	header.OpCode = binary.LittleEndian.Uint16(hbuffer[:2])
	header.Length = binary.LittleEndian.Uint32(hbuffer[2:6])

	if header.OpCode != OpCodeBConnect {
		tb.Fatalf("opcode %d is not BConnect (%d)", header.OpCode, OpCodeBConnect)
	}

	err = capnp.UnmarshalZeroThree(reader.msg, buffer.Bytes())
	if err != nil {
		tb.Fatal(err)
	}

	conn, err := cpnp.ReadRootGameBroadcastConnect(reader.msg)
	if err != nil {
		tb.Fatal(err)
	}

	if !conn.IsValid() {
		tb.Fatal("connection is not valid")
	}

	if !conn.HasPlayer() {
		tb.Fatal("no player wrote")
	}

	player, err := conn.Player()
	if err != nil {
		tb.Fatal(err)
	}

	got, err := player.Name()
	if err != nil {
		tb.Fatal(err)
	}

	if got != name {
		tb.Errorf("got %q, want %q", got, name)
	}
}

func TestMarshal(t *testing.T) {
	writer := NewPacketWriter()
	msg := testMsg(t, writer)
	buf := writer.GetWriteBuffer()
	payload := buf[PacketHeaderLength:]
	_, err := capnext.MarshalThree(msg.Message(), payload)
	if err != nil {
		t.Fatal(err)
	}

	reader := NewPacketReader()
	// scratch := make([]byte, 8*1024)
	err = capnp.UnmarshalZeroThree(reader.msg, payload)
	if err != nil {
		t.Fatal(err)
	}

	unmsg, err := cpnp.ReadRootGameBroadcastConnect(reader.msg)
	if err != nil {
		t.Fatal(err)
	}

	if !unmsg.IsValid() {
		t.Fatal("message is not valid")
	}
}

func TestDeserialize(t *testing.T) {

	stream := new(bytes.Buffer)
	writer := NewPacketWriter()
	var name string
	buffer := make([]byte, 8*1024)
	var n int

	t.Run("Send & Read", func(t *testing.T) {
		name = sendMsg(t, writer, stream)
		var headBuf [PacketHeaderLength]byte
		_, err := io.ReadFull(stream, headBuf[:])
		if err != nil {
			t.Fatal(err)
		}
		n, err = io.ReadFull(stream, buffer[:])
		// This is fine, probably
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) {
			t.Fatalf("%d, %v", n, err)
		}
	})
	reader := NewPacketReader()
	t.Run("Deserialize", func(t *testing.T) {
		msg, err := Deserialize(reader, buffer[:n], cpnp.ReadRootGameBroadcastConnect)
		if err != nil {
			t.Fatal(err)
		}
		if !msg.HasPlayer() {
			t.Fatal("no player wrote")
		}
		player, err := msg.Player()
		if err != nil {
			t.Fatal(err)
		}
		got, err := player.Name()
		if err != nil {
			t.Fatal(err)
		}
		if name != got {
			t.Errorf("got %q, want %q", got, name)
		}
	})
	t.Run("DeserializeValid", func(t *testing.T) {
		msg, valid := DeserializeValid(reader, buffer[:n], cpnp.ReadRootGameBroadcastConnect)
		if !valid {
			t.Fatal("message is not valid")
		}
		if !msg.HasPlayer() {
			t.Fatal("no player wrote")
		}
		player, err := msg.Player()
		if err != nil {
			t.Fatal(err)
		}
		got, err := player.Name()
		if err != nil {
			t.Fatal(err)
		}
		if got != name {
			t.Errorf("got %q, want %q", got, name)
		}
	})
}
