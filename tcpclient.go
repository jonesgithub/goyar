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
	"io"
	"net"
	"net/rpc"
	"sync"
)

type clientCodec struct {
	//rwc io.ReadWriteCloser
	rc io.ReadCloser
	wc io.WriteCloser

	// temporary work space
	header   Header
	packager Packager
	request  Request
	response Response
}

// NewClientCodec returns a new rpc.ClientCodec using JSON-RPC on conn.
func NewClientCodec(conn io.ReadWriteCloser) rpc.ClientCodec {
	return &clientCodec{
		rc: conn,
		wc: conn,
	}
}

func (c *clientCodec) WriteRequest(r *rpc.Request, param interface{}) error {
	//c.req.Method = r.ServiceMethod
	//c.req.Params[0] = param
	//c.req.Id = r.Seq
	//br := PackRequest(uint32(r.Seq), r.ServiceMethod, []interface{}{param})
	//_, err := io.Copy(c.wc, br)
	return nil
}

//func (r *clientResponse) reset() {
//	r.Id = 0
//	r.Result = nil
//	r.Error = nil
//}

func (c *clientCodec) ReadResponseHeader(r *rpc.Response) error {
	if err := binary.Read(c.rc, binary.BigEndian, &c.header); err != nil {
		return err
	}

	r.Error = ""
	r.Seq = uint64(c.header.ID)
	return nil
}

func (c *clientCodec) ReadResponseBody(x interface{}) error {
	var pkg Packager
	if err := pkg.Read(c.rc); err != nil {
		return err
	}

	if pkg != "JSON" {
		return fmt.Errorf("unsupported encode type: %s", pkg)
	}

	if x == nil {
		return nil
	}

	c.response.Retval = x
	buf := make([]byte, c.header.BodyLen)
	n, err := c.rc.Read(buf)
	if err != nil {
		return err
	}
	if n != int(c.header.BodyLen) {
		return fmt.Errorf("Read response body length %d is not equal bodylen of header %d", n, c.header.BodyLen)
	}
	return json.Unmarshal(buf, &c.response)
}

func (c *clientCodec) Close() error {
	//return c.rc.Close()
	return nil
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
