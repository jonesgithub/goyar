// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goyar

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/rpc"
)

const (
	yarprefix = "yar"
)

type serverCodec struct {
	prefix string

	rwc     io.ReadWriteCloser
	c       io.Closer
	r       io.Reader
	w       io.Writer
	request *serverRequest
}

// NewServerCodec returns a new rpc.ServerCodec using YAR-RPC on tcp conn.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return NewNameServerCodec(yarprefix, conn)
}

// NewNameServerCodec returns a new rpc.ServerCodec using YAR-RPC on tcp conn.
func NewNameServerCodec(name string, conn io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		prefix: name,
		c:      conn,
		r:      conn,
		w:      conn,
	}
}

// NewHTTPServerCodec returns a new rpc.ServerCodec using YAR-RPC on http conn.
func NewHTTPServerCodec(conn io.ReadWriteCloser, w http.ResponseWriter, req *http.Request) rpc.ServerCodec {
	return NewHTTPNameServerCodec(yarprefix, conn, w, req)
}

// NewHTTPNameServerCodec returns a new rpc.ServerCodec using YAR-RPC on http conn.
func NewHTTPNameServerCodec(name string, conn io.ReadWriteCloser, w http.ResponseWriter, req *http.Request) rpc.ServerCodec {
	return &serverCodec{
		prefix: name,
		rwc:    conn,
		c:      req.Body,
		r:      req.Body,
		w:      conn,
	}
}

type serverRequest struct {
	ID     uint32           `json:"i"` // yar rpc id
	Method string           `json:"m"` // calling method name
	Params *json.RawMessage `json:"p"` // all the params
}

func (c *serverCodec) ReadRequestHeader(r *rpc.Request) error {
	yh, yerr := ReadHeader(c.r)
	if yerr != nil {
		return yerr
	}

	if !yh.PkgName.Equal("JSON") {
		return errUnsupportedEncoding
	}

	blen := yh.BodyLen - 8

	buf := make([]byte, blen)
	if rn, _ := c.r.Read(buf); rn != int(blen) {
		return fmt.Errorf("Read request body length %d is not equal bodylen of header %d", rn, yh.BodyLen)
	}

	var req serverRequest
	if jerr := json.Unmarshal(buf, &req); jerr != nil {
		return jerr
	}

	r.ServiceMethod = c.prefix + "." + req.Method
	r.Seq = uint64(req.ID)
	c.request = &req

	return nil
}

func (c *serverCodec) ReadRequestBody(x interface{}) error {
	if x == nil {
		return nil
	}
	if c.request.Params == nil {
		return errMissingParams
	}
	// YAR params is array value.
	// RPC params is struct.
	// Unmarshal into array containing struct for now.
	// Should think about making RPC more general.
	var params [1]interface{}
	params[0] = x
	return json.Unmarshal(*c.request.Params, &params)
}

func (c *serverCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	var resp Response

	resp.ID = uint32(r.Seq)
	if r.Error == "" {
		resp.Result = x
	} else {
		resp.Error = r.Error
	}

	err := resp.Write(c.w)
	if c.rwc != nil {
		c.rwc.Close()
	}
	return err
}

func (c *serverCodec) Close() error {
	return c.c.Close()
}

// YarServer yar rpc server
type YarServer struct {
	*rpc.Server
}

// NewYarServer return a yar rpc server
func NewYarServer() *YarServer {
	return &YarServer{rpc.NewServer()}
}

// Register publishes in the server the set of methods of the
// receiver value. Default register name is 'yar'
func (y *YarServer) Register(rcvr interface{}) {
	y.Server.RegisterName(yarprefix, rcvr)
}

// ServeConn runs the YAR-RPC server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
func (y *YarServer) ServeConn(conn io.ReadWriteCloser) {
	y.Server.ServeCodec(NewServerCodec(conn))
}

// HandleHTTP registers an HTTP handler for YAR RPC messages on rpcPath,
// It is still necessary to invoke http.Serve(), typically in a go statement.
func (y *YarServer) HandleHTTP(rpcPath string) {
	http.Handle(rpcPath, y)
}

// ServeHTTP yar http serve handle
func (y *YarServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	io.WriteString(conn, "HTTP/1.0 200 OK\n\n")

	codec := NewHTTPServerCodec(conn, w, req)
	y.Server.ServeCodec(codec)
}
