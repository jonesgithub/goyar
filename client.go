// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goyar

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/rpc"
	"sync"
)

type clientCodec struct {
	rwc io.ReadWriteCloser
	r   io.Reader
	w   io.Writer
	c   io.Closer

	response clientResponse

	// YAR-RPC responses include the request id but not the request method.
	// Package rpc expects both.
	// We save the request method in pending when sending a request
	// and then look it up by request ID when filling out the rpc Response.
	mutex   sync.Mutex        // protects pending
	pending map[uint32]string // map request id to method name
}

// NewClientCodec returns a new rpc.ClientCodec using YAR-RPC on conn.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		rwc:     conn,
		r:       conn,
		w:       conn,
		c:       conn,
		pending: make(map[uint32]string),
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	c.mutex.Lock()
	c.pending[uint32(r.Seq)] = r.ServiceMethod
	c.mutex.Unlock()

	req := Request{
		ID:     uint32(r.Seq),
		Method: r.ServiceMethod,
		Params: []interface{}{param},
	}

	return req.Write(c.w)
}

type clientResponse struct {
	ID     uint32           `json:"i"` // yar rpc id
	Status int32            `json:"s"` // return status code
	Result *json.RawMessage `json:"r"` // return value raw data
	Output string           `json:"o"` // server standard output
	Error  string           `json:"e"` // return error message
}

func (r *clientResponse) reset() {
	r.ID = 0
	r.Result = nil
	r.Error = ""
}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	c.response.reset()

	yh, yerr := ReadHeader(c.r)
	if yerr != nil {
		return yerr
	}

	c.mutex.Lock()
	r.ServiceMethod = c.pending[yh.ID]
	delete(c.pending, yh.ID)
	c.mutex.Unlock()

	if !yh.PkgName.Equal("JSON") {
		return errUnsupportedEncoding
	}

	blen := yh.BodyLen - 8

	buf := make([]byte, blen)
	if rn, _ := c.r.Read(buf); rn != int(blen) {
		return fmt.Errorf("Read request body length %d is not equal bodylen of header %d", rn, yh.BodyLen)
	}

	resp := &c.response
	if jerr := json.Unmarshal(buf, resp); jerr != nil {
		return jerr
	}

	r.Error = ""
	r.Seq = uint64(resp.ID)
	//r.ServiceMethod("")

	return nil
}

func (c *clientCodec) ReadResponseBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.response.Result != nil {
		return json.Unmarshal(*c.response.Result, x)
	}
	return nil
}

func (c *clientCodec) Close() error {
	return c.c.Close()
}

// NewClient returns a new rpc.Client to handle requests to the
// set of services at the other end of the connection.
func NewClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn))
}

// Dial connects to a YAR-RPC server at the specified network address.
func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewClient(conn), err
}
