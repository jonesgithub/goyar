// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package goyar_test

import (
	"fmt"
	"github.com/neverlee/goyar"
	"net/http"
)

type Arith int

type Args struct {
	A int
	B int
}

type Reply struct {
	C int
}

func (t *Arith) Add(args *Args, reply *Reply) error {
	reply.C = args.A + args.B
	return nil
}

func (t *Arith) Set(i *int, r *int) error {
	*r = *i
	return nil
}

func ExampleYarHTTPCLient() {
	client := goyar.NewYHClient("http://yarserver/api.php", nil)
	var r int
	if err := client.MCall("multi", &r, 3, 4); err == nil {
		fmt.Println(r)
	}

	if err := client.Call("Set", 10, &r); err == nil {
		fmt.Println(r)
	}
}

func ExampleYarHTTPServer() {
	yar := goyar.NewYarServer()
	arith := new(Arith)
	yar.Register(arith)
	yar.HandleHTTP("/api.php")

	http.ListenAndServe(":8000", nil)
}

func ExampleYarTCPClient() {
	client, err := goyar.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}
	err := client.Call("Set", 15, &r)
}

func ExampleYarTCPServer() {
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
}
