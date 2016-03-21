// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package goyar provides a jsoncodec for access the remote http yar rpc server.
	Package gorpc implements a YAR-RPC ClientCodec and ServerCodec with json codec for the rpc package, and provide a http yar client

	Here are some simple example.

		// Yar http client
		client := goyar.NewYHClient("http://yarserver/api.php", nil)
		var r int
		err := client.MCall("add", &r, 3, 4)
		err := client.Call("Echo", 10, &r)

		// Yar http server
		type Arith int

		func (t *Arith) Add(args *Args, reply *Reply) error {
			reply.C = args.A + args.B
			return nil
		}

		func (t *Arith) Echo(i *int, r *int) error {
			*r = *i
			return nil
		}

		func main() {
			yar := goyar.NewYarServer()
			arith := new(Arith)
			yar.Register(arith)
			yar.HandleHTTP("/api.php")

			http.ListenAndServe(":8000", nil)
		}

		// Yar tcp client
		client, err := goyar.Dial("tcp", "127.0.0.1:1234")
		if err != nil {
			log.Fatal("dialing:", err)
		}
		err := client.Call("Echo", 15, &r)

		// Yar tcp server
		arith := new(Arith)
		yar := goyar.NewYarServer()
		yar.Register(arith)

		tcpAddr, err := net.ResolveTCPAddr("tcp", ":1234")
		checkError(err)

		listener, err := net.ListenTCP("tcp", tcpAddr)
		checkError(err)

		for {
			conn, err := listener.Accept()
			if err != nil {
				continue
			}
			yar.ServeConn(conn)
		}

*/

package goyar

import (
	"encoding/binary"
	"encoding/json"
	"errors"
	"io"
)

var errMissingParams = errors.New("yarrpc: request body missing params")
var errUnsupportedEncoding = errors.New("yarrpc: request body with unsupportedEncoding")

// Header Yar transport Header(90 bytes)
type Header struct {
	ID       uint32 // transaction id
	Version  uint16 // protocl version
	MagicNum uint32 // default is: 0x80DFEC60
	Reserved uint32
	Provider [32]byte // reqeust from who
	Token    [32]byte // request token, used for authentication
	BodyLen  uint32   // request body len
	PkgName  Packager // body encode name
}

// Request yar request struct(only for json)
type Request struct {
	ID     uint32        `json:"i"` // yar rpc id
	Method string        `json:"m"` // calling method name
	Params []interface{} `json:"p"` // all the params
}

// ReadHeader get a yar header
func ReadHeader(r io.Reader) (*Header, error) {
	var yh Header
	if err := binary.Read(r, binary.BigEndian, &yh); err != nil {
		return nil, err
	}
	return &yh, nil
}

// Write write the header and request
func (r *Request) Write(w io.Writer) error {
	jbyte, jerr := json.Marshal(*r)
	if jerr != nil {
		return jerr
	}

	yh := Header{
		ID:       r.ID,
		Version:  0,
		MagicNum: 0x80DFEC60,
		Reserved: 0,
		BodyLen:  uint32(len(jbyte) + 8),
	}
	yh.PkgName.Set("JSON")

	if err := binary.Write(w, binary.BigEndian, yh); err != nil {
		return err
	}

	if _, err := w.Write(jbyte); err != nil {
		return err
	}

	return nil
}

// Response yar response struct(only for json)
type Response struct {
	ID     uint32      `json:"i"` // yar rpc id
	Status int32       `json:"s"` // return status code
	Result interface{} `json:"r"` // return value
	Output string      `json:"o"` // the called function standard output
	Error  string      `json:"e"` // return error message
}

// Write write the header and response
func (r *Response) Write(w io.Writer) error {
	jbyte, jerr := json.Marshal(*r)
	if jerr != nil {
		return nil
	}

	yh := Header{
		ID:       r.ID,
		Version:  0,
		MagicNum: 0x80DFEC60,
		Reserved: 0,
		BodyLen:  uint32(len(jbyte) + 8),
	}
	yh.PkgName.Set("JSON")

	if err := binary.Write(w, binary.BigEndian, yh); err != nil {
		return err
	}

	if _, err := w.Write(jbyte); err != nil {
		return err
	}
	return nil
}

// Packager yar packager name
type Packager [8]byte

// Equal checking it is equal the string
func (p *Packager) Equal(str string) bool {
	for i := 0; i < 8 && i < len(str); i++ {
		if (*p)[i] != str[i] {
			return false
		}
	}
	return true
}

// Set set a string as pkgname
func (p *Packager) Set(str string) {
	var i int
	for i = 0; i < 8 && i < len(str); i++ {
		(*p)[i] = str[i]
	}
	for ; i < 8; i++ {
		(*p)[i] = 0
	}
}
