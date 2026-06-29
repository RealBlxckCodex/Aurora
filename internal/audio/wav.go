package audio

import (
	"bytes"
	"encoding/binary"
	"math"
)

func EncodeWAV(samples []float32, sampleRate int) []byte {
	numSamples := len(samples)
	dataLen := numSamples * 2
	buf := new(bytes.Buffer)

	binary.Write(buf, binary.LittleEndian, []byte("RIFF"))
	binary.Write(buf, binary.LittleEndian, uint32(36+dataLen))
	binary.Write(buf, binary.LittleEndian, []byte("WAVE"))

	binary.Write(buf, binary.LittleEndian, []byte("fmt "))
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate*2))
	binary.Write(buf, binary.LittleEndian, uint16(2))
	binary.Write(buf, binary.LittleEndian, uint16(16))

	binary.Write(buf, binary.LittleEndian, []byte("data"))
	binary.Write(buf, binary.LittleEndian, uint32(dataLen))

	for _, s := range samples {
		sample := int16(math.MaxInt16 * math.Tanh(float64(s)))
		binary.Write(buf, binary.LittleEndian, sample)
	}

	return buf.Bytes()
}
