// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//	Package goyar provides a client with jsoncodec for calling the remote http yar rpc server.
//
//	Here is a simple example.
//
//		import (
//			"fmt"
//			"github.com/neverlee/goyar"
//		)
//
//		func main() {
//			client := goyar.NewHTTPClient("http://yarserver/yarphp.php", nil)
//			var r int
//			err := client.MCall("add", &r, 3, 4)
//			fmt.Println(r)
//		}

package goyar

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"sync"
)

// Client for yar rpc
type Client struct {
	http  *http.Client // a http client
	seq   uint32       // the rpc call id
	url   string       // remote url
	mutex sync.Mutex
}

// NewHTTPClient returns a new goyar.Client
func NewHTTPClient(url string, client *http.Client) *Client {
	var c Client
	if client == nil {
		client = http.DefaultClient
	}
	c.http = client
	c.url = url
	return &c
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
		ID:       c.seq,
		Version:  0,
		MagicNum: 0x80DFEC60,
		Reserved: 0,
		BodyLen:  uint32(len(jbyte) + 8),
	}
	yh.PkgName.Set("JSON")

	binary.Write(buf, binary.BigEndian, yh)
	buf.Write(jbyte)

	return buf
}

func (c *Client) raise() uint32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.seq++
	return c.seq
}

func (c *Client) mcall(method string, params []interface{}) ([]byte, error) {
	dpack := c.Pack(c.raise(), method, params)

	request, _ := http.NewRequest("POST", c.url, dpack)
	request.Header.Set("Connection", "close")
	request.Header.Set("Content-Type", "application/octet-stream")
	resp, rerr := c.http.Do(request)
	if rerr != nil {
		return nil, rerr
	}
	defer resp.Body.Close()
	body, berr := ioutil.ReadAll(resp.Body)
	if berr == nil {
		if len(body) > 90 {
			return body, nil
		}
		return nil, fmt.Errorf("Response Code %d", resp.StatusCode)
	}
	return nil, berr
}

// MCallRaw calling the remote yarrpc and return the raw byte yar response body
func (c *Client) MCallRaw(method string, params ...interface{}) ([]byte, error) {
	return c.mcall(method, params)
}

// MCall calling the remote yarrpc, print the output and set return value
func (c *Client) MCall(method string, ret interface{}, params ...interface{}) error {
	data, cerr := c.mcall(method, params)
	if cerr == nil {
		jdata := data[90:]
		var resp Response
		resp.Result = ret
		jerr := json.Unmarshal(jdata, &resp)
		if jerr == nil {
			fmt.Print(resp.Output)
			if resp.Error != "" {
				return fmt.Errorf(resp.Error)
			}
			return nil
		}
		return jerr
	}
	return cerr
}

// Call calling the remote yarrpc, only support one param. print the output and set return value
func (c *Client) Call(method string, param interface{}, ret interface{}) error {
	data, cerr := c.mcall(method, []interface{}{param})
	if cerr == nil {
		jdata := data[90:]
		var resp Response
		resp.Result = ret
		jerr := json.Unmarshal(jdata, &resp)
		if jerr == nil {
			fmt.Print(resp.Output)
			if resp.Error != "" {
				return fmt.Errorf(resp.Error)
			}
			return nil
		}
		return jerr
	}
	return cerr
}
