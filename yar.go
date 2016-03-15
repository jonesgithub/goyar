package goyar

import (
	"io"
)

// Header Yar transport Header(82 bytes)
type Header struct {
	Id       uint32 // transaction id
	Version  uint16 // protocl version
	MagicNum uint32 // default is: 0x80DFEC60
	Reserved uint32
	Provider [32]byte // reqeust from who
	Token    [32]byte // request token, used for authentication
	BodyLen  uint32   // request body len
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

// Packager yar packager name
type Packager string

// Bytes return the [8]byte of the Packager
func (p Packager) Bytes() (ret [8]byte) {
	i := 0
	for ; i < 8 && i < len(p); i++ {
		ret[i] = p[i]
	}
	return
}

// Write write the [8]byte of the Packager into
func (p *Packager) Write(w io.Writer) {
	b := p.Bytes()
	w.Write(b[:])
}
