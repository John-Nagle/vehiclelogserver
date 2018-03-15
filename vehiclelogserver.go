//
//  Echo server for FastCGI
//
//  Compatible with Dreamhost
//  Echoes back whatever you send it.
//
package vehiclelogserver

import (
	"fmt"
	"net/http"
	"net/http/fcgi"
)

//  Instance of a server. Called as a subroutine for each request
type FastCGIServer struct{}

func (s FastCGIServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Write([]byte("FastCGI request echo server.\n"))
	w.Write([]byte("Method: "))
	w.Write([]byte(req.Method))
	w.Write([]byte("\n"))
	//  Header items
	w.Write([]byte("Header:\n"))
	for k, v := range req.Header {
		w.Write([]byte(" "))
		w.Write([]byte(k))
		w.Write([]byte("="))
		for i := range v {
		    w.Write([]byte(v[i]))
		    w.Write([]byte(" "))
		    }
		w.Write([]byte("\n"))
	}
	if req.Body != nil {
		w.Write([]byte("Body: "))
		body := make([]byte, 1000)  // buffer for body
		len, _ := req.Body.Read(body)
		w.Write(body[0:len])
		w.Write([]byte("\n"))
	}
}

//  Run FCGI server
func main() {
	fmt.Println("Starting server...")
	b := new(FastCGIServer)
	fcgi.Serve(nil, b)
}
