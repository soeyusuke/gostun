package gostun

import (
	"errors"
	"io"
	"log"
	"sync"
	"time"
)

type CallbackHandle struct {
	Cond     *sync.Cond
	process  bool
	callback func(e MessageObj)
}

var callbackPool = sync.Pool{
	New: func() interface{} {
		return &CallbackHandle{
			Cond: sync.NewCond(new(sync.Mutex)),
		}
	},
}

func (c *CallbackHandle) Reset() {
	c.process = false
	c.callback = nil
}

func (c *CallbackHandle) HandleEvent(e MessageObj) {
	if c.callback == nil {
		log.Fatal("callback is nil")
	}
	// implement f Handler
	c.callback(e)
	c.Cond.L.Lock()
	c.process = true
	c.Cond.Broadcast()
	c.Cond.L.Unlock()
}

func (c *CallbackHandle) Wait() {
	c.Cond.L.Lock()
	for !c.process {
		c.Cond.Wait()
	}
	c.Cond.L.Unlock()
}

func (a *Agent) TransactionHandle(id [TransactionIDSize]byte, h Handler, rto time.Time) error {
	a.mux.Lock()
	defer a.mux.Unlock()

	if a.closed {
		return errors.New("agent closed")
	}

	_, exist := a.transactions[id]
	if exist {
		return errors.New("transaction exists with same id")
	}

	a.transactions[id] = TransactionAgent{
		ID:      id,
		handler: h,
		Timeout: rto,
	}

	return nil
}

func (c *Client) TransactionLaunch(m *Message, h Handler, rto time.Time) error {
	if h != nil {
		if err := c.agent.TransactionHandle(m.TransactionID, h, rto); err != nil {
			return err
		}
	}

	err := m.WriteTo(c.conn)
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) Call(m *Message, rto time.Time) (*XORMappedAddr ,error) {
	var addr XORMappedAddr

	f := callbackPool.Get().(*CallbackHandle)
	f.callback = func(msg MessageObj) {
		if msg.Err != nil {
			log.Fatal(msg.Err)
		}
		if err := addr.GetXORMapped(msg.Msg); err != nil {
			log.Fatal(err)
		}
	}

	defer func() {
		f.Reset()
		callbackPool.Put(f)
	}()

	// waiting TransactionLaunch until call callback func
	if err := c.TransactionLaunch(m, f, rto); err != nil {
		return nil, err
	}
	f.Wait()

	return &addr, nil
}

// write the m.Raw=request to conn
func (m *Message) WriteTo(w io.Writer) error {
	_, err := w.Write(m.Raw)
	return err
}
