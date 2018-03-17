//
//  Echo server for FastCGI
//
//  Compatible with Dreamhost
//  Echoes back whatever you send it.
//
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/http/fcgi"
)

//
//  Configuration
//
//  Default location of config file
//
var configloc string = "~/keys/vehicledbconf.json"

//
//  initialization
//
func initdb(cfile string, sv *FastCGIServer) error {
	//  Read the config file into the server object
	var err error
	sv.config, err = readconfig(cfile)
	if err != nil {
		return err
	}
	//  Set database parameters (does not actually do an open in Go, so it won't fail)
	s := fmt.Sprintf("%s:%s@tcp(%s)/%s",
		sv.config.Mysql.User, sv.config.Mysql.Password, sv.config.Mysql.Domain, sv.config.Mysql.Database)
	sv.db, err = sql.Open("mysql", s)
	if err != nil {
		return err
	}
	return nil // success
}

//  Instance of a server.
type FastCGIServer struct {
	config vdbconfig // the configuration
	db     *sql.DB   // database
}

//
//  dumprequest  -- raw debug print
//
func dumprequest(sv FastCGIServer, w http.ResponseWriter, req *http.Request, bodycontent []byte) {
	w.Write([]byte("FastCGI request, debug info.\n"))
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
	w.Write([]byte("Body: "))
	w.Write(bodycontent)
	w.Write([]byte("\n"))
}

//
//  Called for each request
//
func (sv FastCGIServer) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	body := make([]byte, 5000) // buffer for body, which should not be too big
	if req.Body != nil {
		len, _ := req.Body.Read(body)           // body of HTTP request
		bodycontent := body[0:len]              // take correct part of buffer
		Handlerequest(sv, w, bodycontent, req)  // handle request
	}
}

//  Run FCGI server
func main() {
	fmt.Println("Starting server...")
	sv := new(FastCGIServer)
	err := initdb(configloc, sv)
	if err != nil {
		log.Fatal(err) // initialization failed, cannot start
	}
	fcgi.Serve(nil, sv)
}
