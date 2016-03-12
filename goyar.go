// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.


/*
	Package goyar provides a client with jsoncodec for calling the remote http yar rpc server.

	Here is a simple example.

		import (
			"fmt"
			"github.com/neverlee/goyar"
		)

		func main() {
			client := goyar.NewClient("http://yarserver/yarphp.php", nil)
			var r int
			err := client.Call("add", &r, 3, 4)
			fmt.Println(r)
		}

*/

package goyar

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

// Client for yar rpc
type Client struct {
	http  *http.Client    // a http client
	seq   uint32          // the rpc call id
	url   string          // remote url
	mutex sync.Mutex
}

// NewClient returns a new goyar.Client
func NewClient(url string, client *http.Client) *Client {
	var c Client
	if client == nil {
		client = http.DefaultClient
	}
	c.http = client
	c.url = url
	return &c
}

// Header Yar transport Header(82 bytes)
type Header struct {
	id        uint32 // transaction id
	version   uint16 // protocl version
	magicnum uint32 // default is: 0x80DFEC60
	reserved  uint32
	provider  [32]byte // reqeust from who
	token     [32]byte // request token, used for authentication
	bodylen  uint32   // request body len
}

// Request yar request struct(only for json)
type Request struct {
	ID     uint32        `json:"i"` // yar rpc id
	Method string        `json:"m"` // calling method name
	Params []interface{} `json:"p"` // all the params
}

// Response yar response struct(only for json)
type Response struct {
	ID     uint32      `json:"i"` // yar rpc id
	Status int32       `json:"s"` // return status code
	Retval interface{} `json:"r"` // return value
	Output string      `json:"o"` // the called function standard output
	Errmsg string      `json:"e"` // return error message
}

// Pack a complete yar request body
func (c *Client) Pack(id uint32, method string, params []interface{}) io.Reader {
	dobj := Request{
		ID:     id,
		Method: method,
		Params: params,
	}

	jbyte, jerr := json.Marshal(dobj)
	if jerr != nil {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	yh := Header{
		id:        c.seq,
		version:   0,
		magicnum: 0x80DFEC60,
		reserved:  0,
		bodylen:  uint32(len(jbyte)),
	}

	//binary.Write(buf, binary.LittleEndian, yh)
	binary.Write(buf, binary.BigEndian, yh)

	buf.WriteString("JSON")
	buf.Write([]byte{0, 0, 0, 0})

	buf.Write(jbyte)

	return buf
}

func (c *Client) raise() uint32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.seq++
	return c.seq
}

func (c *Client) call(method string, params []interface{}) ([]byte, error) {
	dpack := c.Pack(c.raise(), method, params)

	request, _ := http.NewRequest("POST", c.url, dpack)
	request.Header.Set("Connection", "close")
	request.Header.Set("Content-Type", "application/octet-stream")
	resp, rerr := c.http.Do(request)
	if rerr != nil {
		return nil, rerr
	}
	defer resp.Body.Close()
	if body, err := ioutil.ReadAll(resp.Body); err == nil {
		if len(body) > 90 {
			return body, nil
		}
		return nil, fmt.Errorf("Response Code %d", resp.StatusCode)
	}
	return nil, err
}

// CallRaw calling the remote yarrpc and return the raw byte yar response body
func (c *Client) CallRaw(method string, params ...interface{}) ([]byte, error) {
	return c.call(method, params)
}

// Call calling the remote yarrpc, print the output and set return value
func (c *Client) Call(method string, ret interface{}, params ...interface{}) error {
	if data, cerr := c.call(method, params); cerr == nil {
		jdata := data[90:]
		var resp Response
		resp.Retval = ret
		if jerr := json.Unmarshal(jdata, &resp); jerr == nil {
			fmt.Print(resp.Output)
			if resp.Errmsg != "" {
				return fmt.Errorf(resp.Errmsg)
			}
			return nil
		}
		return jerr
	}
	return cerr
}
