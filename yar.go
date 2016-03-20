package goyar

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/neverlee/glog"
	"io"
)

var errMissingParams = errors.New("yarrpc: request body missing params")
var errUnsupportedEncoding = errors.New("yarrpc: request body with unsupportedEncoding")

// Header Yar transport Header(82 bytes)
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

func ReadHeader(r io.Reader) (*Header, error) {
	var yh Header
	if err := binary.Read(r, binary.BigEndian, &yh); err != nil {
		return nil, err
	}
	return &yh, nil
}

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

func (r *Response) Write(w io.Writer) error {
	jbyte, jerr := json.Marshal(*r)
	fmt.Println(jbyte, jerr)
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
		glog.Extraln("binary h w", err)
		return err
	}
	glog.Extraln("Write header end")

	if n, err := w.Write(jbyte); err != nil {
		glog.Extraln("Write last ", n, err)
		return err
	}
	return nil
}


// Packager yar packager name
type Packager [8]byte

// Equal check it is equal the string
func (p *Packager) Equal(str string) bool {
	for i:=0; i<8 && i<len(str); i++ {
		if (*p)[i] != str[i] {
			return false
		}
	}
	return true
}

func (p *Packager) Set(str string) {
	var i int
	for i=0; i<8 && i<len(str); i++ {
		(*p)[i] = str[i]
	}
	for ; i<8; i++ {
		(*p)[i] = 0
	}
}
