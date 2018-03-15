//
//  Package eventlogger -- adds events to the "event" table of the vehicle database
//
//  Animats
//  March, 2018
//
package vehiclelogserver

import (
	"database/sql"
	"fmt"
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
	owner_name      string   // name of owner
	object_name     string   // object name
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

func Parseslregion(s string) (slregion, error) {
    var reg slregion;
    return reg, nil
}

func Parseslvector(s string) (slvector, error) {
    var v slvector;
    return v, nil
}


func Parseheader(headervars http.Header) (slheader, error) {
	var hdr slheader
	var err error
	hdr.owner_name = headervars.Get("X-Secondlife-Owner-Name")
	hdr.object_name = headervars.Get("X-Secondlife-Object-Name")
	hdr.region, err = Parseslregion(headervars.Get("X-Secondlife-Region"))
	if err != nil { return hdr, err }
	hdr.local_position, err = Parseslvector(headervars.Get("X-Secondlife-Local-Position"))
	if err != nil { return hdr, err }
	return hdr, nil
}

func Parsevehevent(s string) (vehlogevent, error) {
	var ev vehlogevent
	return ev, nil
}

func Insertindb(database *sql.DB, hdr slheader, ev vehlogevent) error {
	s, err := Constructinsert(hdr, ev)
	if err != nil {
		return (err)
	}
	fmt.Printf("SQL insert: %s\n", s) // ***TEMP***
	return nil
}

func Getauthtokenkey(name string, database *sql.DB)(string, error) {
    return "",nil   // ***TEMP***    
}

//
//  Validateauthtoken -- validate that string has correct hash for auth token
//
func Validateauthtoken(s string, name string, value string, database *sql.DB)(error) {
    _, err := Getauthtokenkey(name, database)
    if err != nil { return(err) }
    //  ***MORE*** do SHA1 check
    return(nil)
}

func Constructinsert(hdr slheader, ev vehlogevent) (string, error) {
	var stmt string = ""
	return stmt, nil
}

//
//  Addevent -- add an event to the database
//
func Addevent(s string, headervars http.Header, database *sql.DB) error {
    //  Validate auth token first
    err := Validateauthtoken(s, headervars.Get("Authtoken-Name"), headervars.Get("Authtoken-Value"), database)
	if err != nil {
		return (err)
	}
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
