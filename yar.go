package goyar

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
)

// Header Yar transport Header(82 bytes)
type Header struct {
	ID       uint32 // transaction id
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

func ReadHeader(r io.Reader) (*Header, error) {
	var yh Header
	if err := binary.Read(r, binary.BigEndian, &yh); err != nil {
		return nil, err
	}
	return &yh, nil
}

func (r *Request) Pack() io.Reader {
	jbyte, jerr := json.Marshal(*r)
	if jerr != nil {
		return nil
	}

	buf := bytes.NewBuffer(nil)
	yh := Header{
		ID:       r.ID,
		Version:  0,
		MagicNum: 0x80DFEC60,
		Reserved: 0,
		BodyLen:  uint32(len(jbyte)),
	}

	//binary.Write(buf, binary.LittleEndian, yh)
	binary.Write(buf, binary.BigEndian, yh)

	pkg := Packager("JSON")
	pkg.Write(buf)

	buf.Write(jbyte)

	return buf
}

// Response yar response struct(only for json)
type Response struct {
	ID     uint32      `json:"i"` // yar rpc id
	Status int32       `json:"s"` // return status code
	Retval interface{} `json:"r"` // return value
	Output string      `json:"o"` // the called function standard output
	Errmsg string      `json:"e"` // return error message
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
		BodyLen:  uint32(len(jbyte)),
	}

	if err := binary.Write(w, binary.BigEndian, yh); err != nil {
		fmt.Println("binary h w", err)
		return err
	}
	fmt.Println("Write header end")

	pkg := Packager("JSON")
	pkg.Write(w)
	fmt.Println("Write pkg end")

	n, e := w.Write(jbyte)
	fmt.Println("Write last ", n, e)
	return nil
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

// Read read the [8]byte of the Packager into
func (p *Packager) Read(r io.Reader) error {
	buf := make([]byte, 8)
	n, err := r.Read(buf)
	if err != nil {
		return err
	}
	if n != 8 {
		return fmt.Errorf("Read packager info only %d", n)
	}

	for n = 7; n >= 0 && buf[n] == 0; n-- {
	}
	*p = Packager(buf[0 : n+1])
	return nil
}

type Yar struct {
	Header   Header
	Packager Packager
	Request  Request
	Response Response
}


func Pack(yar *Yar) ([]byte, error) {
	jsonData, err := json.Marshal(&yar.Response)
	if err != nil {
		return nil, err
	}
	//fmt.Println("json:", string(jsonData))

	jsonDataLen := len(jsonData)
	dataLen := (82 + 8 + jsonDataLen)
	data := make([]byte, dataLen)

	bodyLen := jsonDataLen + 8
	yar.Header.BodyLen = uint32(bodyLen)

	//copy(data[0:4], Uint32ToBytes(yar.Header.Id))
	//copy(data[4:6], Uint16ToBytes(yar.Header.Version))
	//copy(data[6:10], Uint32ToBytes(yar.Header.MagicNum))
	//copy(data[10:14], Uint32ToBytes(yar.Header.Reserved))
	//copy(data[14:46], yar.Header.Provider[:32])
	//copy(data[46:78], yar.Header.Token[:32])
	//copy(data[78:82], Uint32ToBytes(yar.Header.BodyLen))

	//copy(data[82:90], yar.Packager.Data[:8])

	//copy(data[90:dataLen], jsonData)

	return data, nil
}

func UnpackRequest(r io.Reader) (*Request, error) {
	var yh Header
	if berr := binary.Read(r, binary.BigEndian, &yh); berr != nil {
		return nil, berr
	}

	buf := make([]byte, yh.BodyLen)

	n, err := r.Read(buf)
	if err != nil {
		return nil, err
	}
	if n != int(yh.BodyLen) {
		return nil, fmt.Errorf("Read request body length %d is not equal bodylen of header %d", n, yh.BodyLen)
	}
	var rs Request
	if err = json.Unmarshal(buf, &rs); err != nil {
		return nil, err
	}
	return &rs, nil
}
