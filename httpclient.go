// Copyright 2016 Never Lee. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

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

// YHClient yar http client
type YHClient struct {
	http  *http.Client // a http client
	url   string       // remote url
	seq   uint32       // the rpc calling id
	mutex sync.Mutex
}

// NewYHClient returns a new yar http client
func NewYHClient(url string, client *http.Client) *YHClient {
	var c YHClient
	if client == nil {
		client = http.DefaultClient
	}
	c.http = client
	c.url = url
	return &c
}

func (c *YHClient) pack(id uint32, method string, params []interface{}) io.Reader {
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

func (c *YHClient) raise() uint32 {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.seq++
	return c.seq
}

func (c *YHClient) mcall(method string, params []interface{}) ([]byte, error) {
	dpack := c.pack(c.raise(), method, params)

	request, _ := http.NewRequest("POST", c.url, dpack)
	//request.Header.Set("Connection", "close")
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

// MCallRaw calling the remote yarrpc and return the raw byte yar response
func (c *YHClient) MCallRaw(method string, params ...interface{}) ([]byte, error) {
	return c.mcall(method, params)
}

// MCall calling the remote yarrpc, print the output and set return value
func (c *YHClient) MCall(method string, ret interface{}, params ...interface{}) error {
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
func (c *YHClient) Call(method string, param interface{}, ret interface{}) error {
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
