// Copyright 2010 The Go Authors.  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goyar

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/rpc"
	"sync"
	"fmt"
	"github.com/neverlee/glog"
)

var errMissingParams = errors.New("yarrpc: request body missing params")
var errUnsupportedEncoding = errors.New("yarrpc: request body with unsupportedEncoding")

const (
	YARPREFIX = "yar"
)

type serverCodec struct {
	//c   io.Closer
	//req serverRequest
	//rwc     io.ReadWriteCloser
	prefix string

	rwc     io.ReadWriteCloser
	//rw      http.ResponseWriter
	//req *http.Request

	c io.Closer
	r io.Reader
	w io.Writer
	//header *Header
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
		c: conn,
		r: conn,
		w: conn,
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
		rwc:     conn,
		c: req.Body,
		r: req.Body,
		w: conn,
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

	buf := make([]byte, yh.BodyLen)
	if rn, rerr := c.r.Read(buf); rn != int(yh.BodyLen) {
		fmt.Println("read", rn, rerr)
		return fmt.Errorf("Read request body length %d is not equal bodylen of header %d", rn, yh.BodyLen)
	}

	var req serverRequest
	if jerr := json.Unmarshal(buf, &req); jerr != nil {
		return jerr
	}
	glog.Extraln("serverRequest", req)

	r.ServiceMethod = c.prefix + "." + req.Method
	r.Seq = uint64(req.ID)
	c.request = &req
	//c.header = yh

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
		resp.Retval = x
	} else {
		resp.Errmsg = r.Error
	}
	bb, _ := json.Marshal(resp)

	resp.Write(c.w)
	c.rwc.Close()
	return nil
}

func (c *serverCodec) Close() error {
	glog.Extraln("----------Close")
	return c.c.Close()
}

// ServeConn runs the YAR-RPC server on a single connection.
// ServeConn blocks, serving the connection until the client hangs up.
// The caller typically invokes ServeConn in a go statement.
func ServeConn(conn io.ReadWriteCloser) {
	rpc.ServeCodec(NewServerCodec(conn))
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

func (s *YarRpcServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		//log.Print("rpc hijacking ", req.RemoteAddr, ": ", err.Error())
		return
	}
	io.WriteString(conn, "HTTP/1.0 200 OK\n\n")
	
	codec := NewHTTPServerCodec(conn, w, req)
	s.Server.ServeCodec(codec)
}
