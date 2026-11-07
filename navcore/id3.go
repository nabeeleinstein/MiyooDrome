package navcore

import (
	"bytes"
	"encoding/binary"
	"strconv"
)

// ID3v2.4 uses UTF-8 text and synchsafe frame sizes.
// This is only added to transcoded MP3s that have no existing ID3 header.
func syncsafe(n int) []byte {
	return []byte{byte((n >> 21) & 0x7f), byte((n >> 14) & 0x7f), byte((n >> 7) & 0x7f), byte(n & 0x7f)}
}
func id3Text(buf *bytes.Buffer, code, value string) {
	if value == "" {
		return
	}
	data := append([]byte{3}, []byte(value)...)
	buf.WriteString(code)
	buf.Write(syncsafe(len(data)))
	binary.Write(buf, binary.BigEndian, uint16(0))
	buf.Write(data)
}
func MakeID3Tag(s Song) []byte {
	b := &bytes.Buffer{}
	id3Text(b, "TIT2", s.Title)
	id3Text(b, "TPE1", s.Artist)
	id3Text(b, "TALB", s.Album)
	if s.Track > 0 {
		id3Text(b, "TRCK", strconv.Itoa(s.Track))
	}
	out := &bytes.Buffer{}
	out.WriteString("ID3")
	out.Write([]byte{4, 0, 0})
	out.Write(syncsafe(b.Len()))
	out.Write(b.Bytes())
	return out.Bytes()
}
