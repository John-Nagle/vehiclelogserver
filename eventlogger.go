//
//  Package eventlogger -- adds events to the "event" table of the vehicle database
//
//  Animats
//  March, 2018
//
package vehiclelogserver

import (
	////	"fmt"
	"database/sql"
	"net/http"
	////import "github.com/go-sql-driver/mysql"
)

//
//  A logging event
//
type slvector struct {
	x float32
	y float32
	z float32
}

type slregion struct {
	name string
	x    int32
	y    int32
}

//  Typical header data from SL servers
//
//  "X-Secondlife-Owner-Key" : {"dadec334-539a-4875-ad0e-d9654705f437"},
//  "X-Secondlife-Object-Name" : {"Logging tester 0.4"},
//  "X-Secondlife-Owner-Name" : {"animats Resident "},
//  "X-Secondlife-Object-Key" : {"b23730f8-4105-594a-c359-e72f9fece699"},
//  "X-Secondlife-Region" : {"Vallone (462592, 306944)"},
//  "Authtoken-Value" : {"0bc935dbb51aaeaf2ae0e98362d3b7500db36350"},
//  "X-Secondlife-Local-Position" : {"(204.783539, 26.682831, 35.563702)"},
//  "Authtoken-Name" : {"TEST"},

//  Typical JSON from our logger
//
//  "{\"tripid\":\"ABCDEF\",\"severity\":2,\"type\":\"STARTUP\",\"msg\":\"John Doe\",\"auxval\":1.0}"

type slheader struct {
	authtoken_name  string   // name of auth token
	authtoken_value string   // value of auth token
	owner_name      string   // name of owner
	object_key      string   // object key
	region          slregion // SL region name and corner
	local_position  slvector // position within region
}

type vehlogevent struct {
	tripid    string  // trip ID (random unique identifier)
	severity  int8    // an enum, really
	eventtype string  // STARTUP, SHUTDOWN, etc.
	msg       string  // human-readable message
	auxval    float32 // some other value associated with the event
}

func Parseheader(headervars http.Header) (slheader, error) {
	var hdr slheader
	return hdr, nil
}

func Parsevehevent(s string) (vehlogevent, error) {
	var ev vehlogevent
	return ev, nil
}

func Insertindb(database *sql.DB, hdr slheader, ev vehlogevent) error {
	return nil
}

//
//  Addevent -- add an event to the database
//
func Addevent(s string, headervars http.Header, database *sql.DB) error {
	hdr, err := Parseheader(headervars) // parse HTTP header
	if err != nil {
		return (err)
	}
	ev, err := Parsevehevent(s) // parse JSON from vehicle script
	if err != nil {
		return (err)
	}
	return (Insertindb(database, hdr, ev)) // insert in database
}
