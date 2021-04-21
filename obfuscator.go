package obfuscator

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/tdewolff/minify"
	"github.com/tdewolff/minify/js"
)

const LowWord = 2
const HiWord = 10
const Printables = " !#$%&()*+,-./0123456789:;<=>?@ABCDEFGHIJKLMNOPQRSTUVWXYZ[]^_`abcdefghijklmnopqrstuvwxyz{|}~"
const PrintablesCount = len(Printables)

type Script struct {
	Buf []byte
}

func splitSized(buf []byte, lim int) [][]byte {
	var chunk []byte
	chunks := make([][]byte, 0, len(buf)/lim+1)
	for len(buf) >= lim {
		chunk, buf = buf[:lim], buf[lim:]
		chunks = append(chunks, chunk)
	}
	if len(buf) > 0 {
		chunks = append(chunks, buf[:len(buf)])
	}
	return chunks
}

// TODO(stgleb): Use better pattern matching algorithm
// Creates a dictionary of the given word length and counts occurrences.
func dict(buf []byte, wlen int) map[string]int {
	dict := map[string]int{}

	for offset := 0; offset < wlen; offset++ {
		chunks := splitSized(buf, wlen)
		l := len(chunks)
		for i := 0; i < l; i++ {
			s := string(chunks[i])

			if _, ok := dict[s]; ok == false {
				dict[s] = 0
			}
			dict[s]++
		}
	}

	//l := len(buf)
	//for i := 0; i+wlen < l; i++ {
	//	s := string(buf[i : i+wlen])
	//	if _, ok := dict[s]; ok == false {
	//		dict[s] = 0
	//	}
	//	dict[s]++
	//}

	return dict
}

// Returns the first unused printable byte, we need it to store a chunk.
func firstUnusedPrintable(buf []byte, used map[byte]bool) (ret byte, err error) {
	var c byte
	printableBytes := []byte(Printables)
	for i:=0; i < PrintablesCount; i++ {
		c = printableBytes[i]
		if ((-1 == bytes.IndexByte(buf, c)) && !used[c]) {
			return c, nil
		}
	}

	return byte(0), errors.New("Free printable not found\n")
}


func Implode(buf []byte) ([]byte, []byte, error) {
	used := make(map[byte]bool)
	keys := []byte{}
	ret  := []byte{}

	for {
		key, err := firstUnusedPrintable(buf, used)
		if err != nil {
			break
		}

		chosenPiece := ""
		chosenPieceWeight := 0

		// search for best piece
		for i := LowWord; i < HiWord; i++ {
			d := dict(buf, i)
			for piece, count := range d {
				if chosenPieceWeight < i * count {
					chosenPiece = piece
					chosenPieceWeight = i * count
				}
			}
		}

		if chosenPiece == "" {
			break
		}

		used[key] = true
		buf = bytes.Replace(buf, []byte(chosenPiece), []byte{key}, -1)
		buf = append(buf, key)
		buf = append(buf, []byte(chosenPiece)...)
		keys = append([]byte{key}, keys...)
	}

	ret = append(buf, []byte(ret)...)

	return ret, keys, nil
}


// Uses buf and key to create the javascript decompressor.
func Pack(buf []byte, keys []byte) string {
	return fmt.Sprintf("for(s='%s',i=0;j='%s'[i++];)with(s.split(j))s=join(pop());eval(s)", string(buf), string(keys))
}

func Minify(data []byte) ([]byte, error) {
	m := minify.New()
	m.AddFunc("text/javascript", js.Minify)

	b, err := m.Bytes("text/javascript", data)
	if err != nil {
		return nil, err
	}

	return b, nil
}

func Obfuscate(data []byte) (string, error) {
	buf, err := Minify(data)

	t, k, err := Implode(buf)

	if err != nil {
		return "", err
	}

	return Pack(t, k), nil
}
