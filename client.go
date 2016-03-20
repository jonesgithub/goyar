// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package jsonrpc implements a JSON-RPC ClientCodec and ServerCodec
// for the rpc package.
package goyar

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/neverlee/glog"
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
}

// NewClientCodec returns a new rpc.ClientCodec using JSON-RPC on conn.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		rwc: conn,
		r:   conn,
		w:   conn,
		c:   conn,
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	req := Request{
		ID:     uint32(r.Seq),
		Method: r.ServiceMethod,
		Params: []interface{}{param},
	}

	return req.Write(c.w)
}

// Response yar response struct(only for json)
type clientResponse struct {
	ID     uint32           `json:"i"` // yar rpc id
	Status int32            `json:"s"` // return status code
	Result *json.RawMessage `json:"r"` // return value raw data
	Output string           `json:"o"` // the called function standard output
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
	glog.Extraln("ReadRequestHeader")
	glog.Extraln(yh, yerr)
	if yerr != nil {
		return yerr
	}

	glog.Extraln("pkgname", yh.PkgName)
	if !yh.PkgName.Equal("JSON") {
		return errUnsupportedEncoding
	}

	blen := yh.BodyLen - 8

	buf := make([]byte, blen)
	if rn, rerr := c.r.Read(buf); rn != int(blen) {
		glog.Extraln("read", rn, rerr, string(buf))
		return fmt.Errorf("Read request body length %d is not equal bodylen of header %d", rn, yh.BodyLen)
	}
	glog.Extraln("readBody", string(buf))
	glog.Extraln("readBody", buf)

	resp := &c.response
	if jerr := json.Unmarshal(buf, resp); jerr != nil {
		glog.Extraln(jerr)
		return jerr
	}
	glog.Extraln("clientResponse", resp)

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
func NewTCPClient(conn io.ReadWriteCloser) *rpc.Client {
	return rpc.NewClientWithCodec(NewClientCodec(conn))
}

// Dial connects to a JSON-RPC server at the specified network address.
func Dial(network, address string) (*rpc.Client, error) {
	conn, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	return NewTCPClient(conn), err
}
