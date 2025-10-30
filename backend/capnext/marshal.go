package capnext

import (
	"encoding/binary"
	"errors"

	"capnproto.org/go/capnp/v3"
	"capnproto.org/go/capnp/v3/exc"
)

// From https://github.com/knervous/eqrequiem

func streamHeaderSize(maxSeg capnp.SegmentID) uint64 {
	return ((uint64(maxSeg)+2)*4 + 7) &^ 7
}

// ErrBufferTooSmall same as above…
var ErrBufferTooSmall = errors.New("marshal: buffer too small")

const wordSize capnp.Size = 8

// MarshalThree writes m into buf, just like a Message.MarshalTo method would.
// You call it as capnpext.MarshalThree(msg, buf).
func MarshalThree(m *capnp.Message, buf []byte) (int, error) {
	// 1) Count segments
	nsegs := m.NumSegments()
	if nsegs == 0 {
		return 0, errors.New("marshal: message has no segments")
	}

	// 2) Compute header size and total data size
	hdrSize := int(streamHeaderSize(capnp.SegmentID(nsegs - 1)))
	var dataSize int
	for i := int64(0); i < nsegs; i++ {
		seg, err := m.Segment(capnp.SegmentID(i))
		if err != nil {
			return 0, exc.WrapError("marshal", err)
		}
		ln := len(seg.Data())
		// if ln%int(wordSize) != 0 {
		// 	return 0, errors.New("marshal: segment " + Itod(i) + " not word-aligned")
		// }
		dataSize += ln
	}

	total := hdrSize + dataSize
	if len(buf) < total {
		return total, ErrBufferTooSmall
	}

	// 3) Write framing header into buf[0:hdrSize]
	//   - first 4 bytes: segment count minus one
	binary.LittleEndian.PutUint32(buf[0:4], uint32(nsegs-1))
	//   - next 4 bytes per segment: segment length in words
	off := hdrSize
	ln := 0
	for i := int64(0); i < nsegs; i++ {
		seg, _ := m.Segment(capnp.SegmentID(i)) // already validated above
		// each length field is at offset (i+1)*4
		binary.LittleEndian.PutUint32(
			buf[int((i+1)*4):int((i+2)*4)],
			uint32(len(seg.Data())/int(wordSize)),
		)
		ln = len(seg.Data())
		copy(buf[off:off+ln], seg.Data())
		off += ln
	}

	return total, nil
}
