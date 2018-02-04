package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"
)

const (
	stunMessageHeader   = 20
	magicCookie         = 0x2112A442
	stunAttributeHeader = 4
)

type Client struct {
	conn Connection
}

type Message struct {
	Type          MessageType
	Length        uint32
	TransactionID [TransactionIDSize]byte
	Attributes    Attributes
	Raw           []byte
}

type MessageType struct {
	Method Method       // binding
	Class  MessageClass // request
}

type Connection interface {
	io.Reader
}

func main() {
	conn, err := net.Dial("udp", "stun.l.google.com:19302")
	if err != nil {
		log.Fatal(err)
	}
	m := new(Message)
	c := &Client{
		conn: conn,
	}

	m.Raw = make([]byte, 1024)
	raw, err := m.ReadFrom(c.conn)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(raw)
}

func (m *Message) ReadFrom(r io.Reader) (int64, error) {
	tBuf := m.Raw[:cap(m.Raw)]

	n, err := r.Read(tBuf)
	if err != nil {
		return int64(n), err
	}
	m.Raw = tBuf[:n]
	return int64(n), m.Decode()
}

func (m *Message) Decode() error {
	buf := m.Raw
	if len(buf) < stunMessageHeader {
		return errors.New("unexpected EOF: not enough bytes to read header")
	}

	var (
		head        = bin.Uint16(buf[0:2])
		size        = int(bin.Uint16(buf[2:4]))
		magiccookie = bin.Uint32(buf[4:8])
		fullMsg     = messageHeaderSize + size
	)

	if magiccookie != magicCookie {
		err := fmt.Sprintf("%x is invalid magic cookie. maegic Cookie: %x", magiccookie, magicCookie)
		return errors.New(err)
	}

	if len(buf) < fullMsg {
		err := fmt.Sprintf("buffer length is invalid %d, expected message size %d", len(buf), fullMsg)
		return errors.New(err)
	}

	//copy header data
	m.Type.ReadValue(head)

	return nil
}

const (
	firstBit  = 0x1
	secondBit = 0x2

	c0Bit = firstBit
	c1Bit = secondBit

	C0Shift = 4
	C1Shift = 7
)

// RFC5389 Page10 Message Type field
type MessageClass byte

const (
	RequestClass         MessageClass = 0x00 // 0b00
	IndicationClass      MessageClass = 0x01 // 0b01
	SuccessResponseClass MessageClass = 0x02 // 0b10
	ErrorResponseClass   MessageClass = 0x03 // 0b11
)

func (t *MessageType) ReadValue(v uint16) {

}