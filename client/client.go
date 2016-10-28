package client

import (
	"github.com/tweekmonster/sshclip"
	"golang.org/x/crypto/ssh"
)

type Client struct {
	ch   ssh.Channel
	reqs <-chan *ssh.Request
}

func New(c *ssh.Client) (*Client, error) {
	ch, reqs, err := c.OpenChannel("sshclip", nil)
	if err != nil {
		return nil, err
	}

	go ssh.DiscardRequests(reqs)
	// go func() {
	// 	for req := range reqs {
	// 		sshclip.Dlog("Replying")
	// 		if req.WantReply {
	// 			req.Reply(false, nil)
	// 		}
	// 	}
	// }()

	cli := &Client{
		ch:   ch,
		reqs: reqs,
	}

	return cli, nil
}

func (c *Client) Close() error {
	return c.ch.Close()
}

func (c *Client) Get(reg int) (sshclip.RegisterItem, error) {
	header := sshclip.MakePayloadHeader(sshclip.OpGet, 0, reg)
	_, err := c.ch.Write(header)
	if err != nil {
		return nil, err
	}

	_, attrs, reg, err := sshclip.ReadPayloadHeader(c.ch)
	if err != nil {
		return nil, err
	}

	data, err := sshclip.ReadPayload(c.ch)
	if err != nil {
		return nil, err
	}

	return &sshclip.MemoryRegisterItem{
		Index: reg,
		Attrs: attrs,
		Data:  data,
	}, nil
}

func (c *Client) Put(reg, attrs int, data []byte) (int, error) {
	m := &sshclip.MemoryRegisterItem{
		Index: reg,
		Attrs: attrs,
		Data:  data,
	}

	return sshclip.SendRegister(c.ch, m)
}
