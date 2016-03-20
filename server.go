// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goyar

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/neverlee/glog"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/rpc"
	"sync"
)

const (
	YARPREFIX = "yar"
)

type serverCodec struct {
	prefix string

	//req   *http.Request
	//rw      http.ResponseWriter

	rwc io.ReadWriteCloser
	c io.Closer
	r io.Reader
	w io.Writer
	request *serverRequest
}

// NewServerCodec returns a new rpc.ServerCodec using JSON-RPC on conn.
func NewServerCodec(conn io.ReadWriteCloser) rpc.ServerCodec {
	return NewNameServerCodec(YARPREFIX, conn)
}

// NewNameServerCodec returns a new rpc.ServerCodec using JSON-RPC on conn.
func NewNameServerCodec(name string, conn io.ReadWriteCloser) rpc.ServerCodec {
	return &serverCodec{
		prefix: name,
		c:      conn,
		r:      conn,
		w:      conn,
	}
}

// NewHTTPServerCodec returns a new rpc.ServerCodec using JSON-RPC on conn.
func NewHTTPServerCodec(conn io.ReadWriteCloser, w http.ResponseWriter, req *http.Request) rpc.ServerCodec {
	return NewHTTPNameServerCodec(YARPREFIX, conn, w, req)
}

// NewHTTPNameServerCodec returns a new rpc.ServerCodec using JSON-RPC on conn.
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
	glog.Extraln("ReadRequestHeader")
	glog.Extraln(yh, yerr)
	if yerr != nil {
		return yerr
	}

	var pkg Packager
	pkg.Read(c.r)
	if pkg != "JSON" {
		return errUnsupportedEncoding
	}
	glog.Extraln("pkg", pkg)

	blen := yh.BodyLen - 8

	buf := make([]byte, blen)
	if rn, rerr := c.r.Read(buf); rn != int(blen) {
		glog.Extraln("read", rn, rerr, string(buf))
		return fmt.Errorf("Read request body length %d is not equal bodylen of header %d", rn, yh.BodyLen)
	}
	glog.Extraln("readBody", string(buf))

	var req serverRequest
	if jerr := json.Unmarshal(buf, &req); jerr != nil {
		glog.Extraln(jerr)
		return jerr
	}
	glog.Extraln("serverRequest", req)

	r.ServiceMethod = c.prefix + "." + req.Method
	r.Seq = uint64(req.ID)
	c.request = &req

	return nil
}

func (c *serverCodec) ReadRequestBody(x interface{}) error {
	glog.Extraln("----------ReadRequestBody")
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

//var null = json.RawMessage([]byte("null"))

func (c *serverCodec) WriteResponse(r *rpc.Response, x interface{}) error {
	glog.Extraln("----------WriteResponse")
	var resp Response

	resp.ID = uint32(r.Seq)
	if r.Error == "" {
		resp.Result = x
	} else {
		resp.Error = r.Error
	}
	bb, _ := json.Marshal(resp)

	resp.Write(c.w)
	if c.rwc != nil {
		c.rwc.Close()
	}
	return nil
}

func (c *serverCodec) Close() error {
	glog.Extraln("----------Close")
	return c.c.Close()
}

type YarRpcServer struct {
	*rpc.Server
}

func (y *YarRpcServer) Register(rcvr interface{}) {
	y.Server.RegisterName(YARPREFIX, rcvr)
}

func NewYarRpcServer() *YarRpcServer {
	return &YarRpcServer{rpc.NewServer()}
}

func (s *YarRpcServer) ServeConn(conn io.ReadWriteCloser) {
	s.Server.ServeCodec(NewServerCodec(conn))
}

func (s *YarRpcServer) HandleHTTP(rpcPath string) {
	http.Handle(rpcPath, s)
}

func (s *YarRpcServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	io.WriteString(conn, "HTTP/1.0 200 OK\n\n")

	codec := NewHTTPServerCodec(conn, w, req)
	s.Server.ServeCodec(codec)
}
