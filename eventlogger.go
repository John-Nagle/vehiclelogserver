//
//  Package eventlogger -- adds events to the "event" table of the vehicle database
//
//  Animats
//  March, 2018
//
package vehiclelogserver

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"encoding/json"
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

func (v slvector) String() string {
	return fmt.Sprintf("(%f,%f,%f)", v.x, v.y, v.z)
}

type slregion struct {
	name string
	x    int32
	y    int32
}

func (r slregion) String() string {
	return fmt.Sprintf("%s (%d,%d)", r.name, r.x, r.y)
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
	Owner_name     string   // name of owner
	Object_name    string   // object name
	Region         slregion // SL region name and corner
	Local_position slvector // position within region
}

func (r slheader) String() string {
	return fmt.Sprintf("owner_name: \"%s\"  object_name: \"%s\"  region: %s  local_position: %s", r.Owner_name, r.Object_name, r.Region, r.Local_position)
}

type vehlogevent struct {
    Timestamp int64   // UNIX timestamp, long form
	Tripid    string  // trip ID (random unique identifier)
	Severity  int8    // an enum, really
	Eventtype string  // STARTUP, SHUTDOWN, etc.
	Msg       string  // human-readable message
	Auxval    float32 // some other value associated with the event
}

func (r vehlogevent) String() string {
	return fmt.Sprintf("timestamp: %d  tripid: \"%s\"  severity: %d  eventtype: %s  msg: %s  auxval: %f", 
	    r.Timestamp, r.Tripid, r.Severity, r.Eventtype, r.Msg, r.Auxval)
}

//  Parseslregion - parse forms such as "Vallone (462592, 306944)"
func Parseslregion(s string) (slregion, error) {
	var reg slregion
	ix := strings.LastIndex(s, "(") // find rightmost paren
	if ix < 0 {
		return reg, errors.New("SL region location not in expected format")
	}
	reg.name = strings.TrimSpace(s[0 : ix-1])               // name part
	_, err := fmt.Sscanf(s[ix:], "(%d,%d)", &reg.x, &reg.y) // location part
	return reg, err
}

//  Parseslvector - parse forms such as "(204.783539, 26.682831, 35.563702)"
func Parseslvector(s string) (slvector, error) {
	var p slvector
	_, err := fmt.Sscanf(s, "(%f,%f,%f)", &p.x, &p.y, &p.z)
	return p, err
}

func Parseheader(headervars http.Header) (slheader, error) {
	var hdr slheader
	var err error
	hdr.Owner_name = strings.TrimSpace(headervars.Get("X-Secondlife-Owner-Name"))
	hdr.Object_name = strings.TrimSpace(headervars.Get("X-Secondlife-Object-Name"))
	hdr.Region, err = Parseslregion(headervars.Get("X-Secondlife-Region"))
	if err != nil {
		return hdr, err
	}
	hdr.Local_position, err = Parseslvector(headervars.Get("X-Secondlife-Local-Position"))
	if err != nil {
		return hdr, err
	}
	fmt.Printf("Parseheader: %s\n", hdr) // ***TEMP***
	return hdr, nil
}

func Parsevehevent(s []byte) (vehlogevent, error) {
	var ev vehlogevent
	err := json.Unmarshal(s, &ev);      // decode JSON
	fmt.Printf("Parsevehevent: %s ->  %s \n", string(s), ev);    // ***TEMP***
	if (err != nil) { fmt.Printf("JSON decode error: %s\n", err.Error()) }
	return ev, err
}

func Insertindb(database *sql.DB, hdr slheader, ev vehlogevent) error {
	s, err := Constructinsert(hdr, ev)
	if err != nil {
		return (err)
	}
	fmt.Printf("SQL insert: %s\n", s) // ***TEMP***
	return nil
}

func Getauthtokenkey(name string, database *sql.DB) (string, error) {
	return "", nil // ***TEMP***
}

//
//  Validateauthtoken -- validate that string has correct hash for auth token
//
func Validateauthtoken(s []byte, name string, value string, database *sql.DB) error {
	_, err := Getauthtokenkey(name, database)
	if err != nil {
		return (err)
	}
	//  ***MORE*** do SHA1 check
	return (nil)
}

func Constructinsert(hdr slheader, ev vehlogevent) (string, error) {
	var stmt string = ""
	return stmt, nil
}

//
//  Addevent -- add an event to the database
//
func Addevent(s []byte, headervars http.Header, database *sql.DB) error {
	//  Validate auth token first
	err := Validateauthtoken(s,
		strings.TrimSpace(headervars.Get("Authtoken-Name")),
		strings.TrimSpace(headervars.Get("Authtoken-Value")), database)
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
