# goyar
Http yar rpc client with json code in golang

## Example
```go
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
```

## LICENSE
Apache License
