package quickfix

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"time"
)

const (
	defaultBufSize = 4096
)

type parser struct {
	//buffer is a slice of bigBuffer
	bigBuffer, buffer []byte
	reader            io.Reader
	lastRead          time.Time
}

func newParser(reader io.Reader) *parser {
	return &parser{reader: reader}
}

func (p *parser) readMore() (int, error) {
	fmt.Println("-------------------- read_more")
	p.Debug()
	if len(p.buffer) == cap(p.buffer) {
		var newBuffer []byte
		switch {
		//initialize the parser
		case len(p.bigBuffer) == 0:
			p.bigBuffer = make([]byte, defaultBufSize)
			newBuffer = p.bigBuffer[0:0]
			fmt.Println("-- s1")
			p.Debug()
		//shift buffer back to the start of bigBuffer
		case 2*len(p.buffer) <= len(p.bigBuffer):
			newBuffer = p.bigBuffer[0:len(p.buffer)]
			fmt.Println("-- s2")
			p.Debug()
		//reallocate big buffer with enough space to shift buffer
		default:
			p.bigBuffer = make([]byte, 2*len(p.buffer))
			newBuffer = p.bigBuffer[0:len(p.buffer)]
			fmt.Println("-- s3")
			p.Debug()
		}

		copy(newBuffer, p.buffer)
		p.buffer = newBuffer
	}

	fmt.Println("---- before read")
	p.Debug()
	n, e := p.reader.Read(p.buffer[len(p.buffer):cap(p.buffer)])
	fmt.Println("---- after read")
	p.Debug()
	p.lastRead = time.Now()
	p.buffer = p.buffer[:len(p.buffer)+n]
	fmt.Println("---- after resize")
	p.Debug()

	if e != nil {
		fmt.Printf("--------------------- e? %v\n", e)
	}
	return n, e
}

func (p *parser) findIndex(delim []byte) (int, error) {
	return p.findIndexAfterOffset(0, delim)
}

func (p *parser) findIndexAfterOffset(offset int, delim []byte) (int, error) {
	fmt.Printf("----------------------------------- findIndexAfterOffset\n")
	i := 0
	for {
		fmt.Printf("---------------------- i %d\n", i)
		i++
		if offset > len(p.buffer) {
			fmt.Println("------------- findIndexAfterOffset 1st")
			if n, err := p.readMore(); n == 0 && err != nil {
				fmt.Printf("------------- findIndexAfterOffset 1st err %v\n", err)
				return -1, err
			}

			continue
		}

		if index := bytes.Index(p.buffer[offset:], delim); index != -1 {
			fmt.Printf("------------- findIndexAfterOffset index offset %d %d\n", index, offset)
			return index + offset, nil
		}

		n, err := p.readMore()

		if n == 0 && err != nil {
			fmt.Printf("------------- findIndexAfterOffset 2nd err %v\n", err)
			return -1, err
		}
	}
}

func (p *parser) findStart() (int, error) {
	return p.findIndex([]byte("8="))
}

func (p *parser) findEndAfterOffset(offset int) (int, error) {
	index, err := p.findIndexAfterOffset(offset, []byte("\00110="))
	if err != nil {
		return index, err
	}

	index, err = p.findIndexAfterOffset(index+1, []byte("\001"))
	if err != nil {
		return index, err
	}

	return index + 1, nil
}

func (p *parser) jumpLength() (int, error) {
	lengthIndex, err := p.findIndex([]byte("9="))
	if err != nil {
		return 0, err
	}

	lengthIndex += 3

	offset, err := p.findIndexAfterOffset(lengthIndex, []byte("\001"))
	if err != nil {
		return 0, err
	}

	if offset == lengthIndex {
		return 0, errors.New("No length given")
	}

	length, err := atoi(p.buffer[lengthIndex:offset])
	if err != nil {
		return length, err
	}

	if length <= 0 {
		return length, errors.New("Invalid length")
	}

	fmt.Printf("------------- jumpLength offset %d length %d \n", offset, length)

	return offset + length, nil
}

func (p *parser) ReadMessage() (msgBytes *bytes.Buffer, err error) {
	start, err := p.findStart()
	if err != nil {
		return
	}
	fmt.Printf("---------------------- ReadMessage start %d\n", start)
	p.Debug()
	p.buffer = p.buffer[start:]
	fmt.Printf("---------------------- ReadMessage after reslice1\n")
	p.Debug()

	index, err := p.jumpLength()
	if err != nil {
		return
	}
	fmt.Printf("---------------------- ReadMessage index1 %d\n", index)
	index, err = p.findEndAfterOffset(index)
	if err != nil {
		return
	}
	fmt.Printf("---------------------- ReadMessage index2 %d\n", index)

	msgBytes = new(bytes.Buffer)
	msgBytes.Reset()
	msgBytes.Write(p.buffer[:index])
	fmt.Println("")
	fmt.Printf("-------------- msgBytes %s\n", msgBytes.String())
	p.buffer = p.buffer[index:]
	fmt.Printf("---------------------- ReadMessage after reslice2\n")
	p.Debug()
	return
}

func (p *parser) Debug() {
	fmt.Printf("---------------------- %d %d buffer %s\n", len(p.buffer), cap(p.buffer), string(p.buffer))
	fmt.Printf("---------------------- %d %d bigBuffer %s\n", len(p.bigBuffer), cap(p.bigBuffer), string(p.bigBuffer))
	start := bytes.Index(p.bigBuffer, p.buffer)
	fmt.Printf("\n---------------------- %d %d start end\n", start, start+len(p.buffer))

}
