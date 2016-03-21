# goyar
yar rpc library with json code in golang

## Yar http server Example
```go
package main

import (
	"fmt"
	"github.com/neverlee/goyar"
	"net/http"
)

type Arith int

func (t *Arith) Echo(i *int, r *int) error {
	*r = *i
	return nil
}

func main() {
	yar := goyar.NewYarServer()
	ar := new(Arith)
	yar.Register(ar)
	yar.HandleHTTP("/api")

	http.ListenAndServe(":8000", nil)
}
```

## Yar http client Example
```go
import (
    "fmt"
    "github.com/neverlee/goyar"
)

func main() {
    client := goyar.NewYHClient("http://yarserver/api.php", nil)
    var r int
    err := client.MCall("add", &r, 3, 4)
    err := client.MCall("Echo", 100, &r)
}
```

## Yar tcp server Example
```go
package main

import (
	"fmt"
	"github.com/neverlee/goyar"
	"net"
	"os"
)

type Arith int

func (t *Arith) Echo(i *int, r *int) error {
	*r = *i
	fmt.Println("do echo", *i)
	return nil
}

func main() {
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

func checkError(err error) {
	if err != nil {
		fmt.Println("Fatal error ", err.Error())
		os.Exit(1)
	}
}
```

## Yar tcp client Example
```go
package main

import (
	"fmt"
	"github.com/neverlee/goyar"
	"log"
)

type Args struct {
	A, B int
}

type Quotient struct {
	Quo, Rem int
}

func main() {
	client, err := goyar.Dial("tcp", "127.0.0.1:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	var r int
	fmt.Println(client.Call("Echo", 15, &r))
	fmt.Println(r)
}
```


## LICENSE
Apache License
