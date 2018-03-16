//
//  Package eventlogger -- adds events to the "event" table of the vehicle database
//
//  Animats
//  March, 2018
//
package vehiclelogserver

import (
	"crypto/sha1" // cryptograpically weak, but SL still uses it
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/user"
	"path/filepath"
	"strings"
	"github.com/go-sql-driver/mysql"
)
//
//  Package-local types
//
//  A logging event
//
type slvector struct {
	X float32
	Y float32
	Z float32
}

func (v slvector) String() string {
	return fmt.Sprintf("(%f,%f,%f)", v.X, v.Y, v.Z)
}

type slregion struct { // region corners, always integer meters
	Name string
	X    int32
	Y    int32
}

func (r slregion) String() string {
	return fmt.Sprintf("%s (%d,%d)", r.Name, r.X, r.Y)
}

//  Typical header data from SL servers
//
//  "X-Secondlife-Owner-Key" : {"dadec334-539a-4875-ad0e-d9654705f437"},
//  "X-Secondlife-Object-Name" : {"Logging tester 0.4"},
//  "X-Secondlife-Owner-Name" : {"animats Resident "},
//  "X-Secondlife-Object-Key" : {"b23730f8-4105-594a-c359-e72f9fece699"},
//  "X-Secondlife-Region" : {"Vallone (462592, 306944)"},
//  "X-Authtoken-Value" : {"0bc935dbb51aaeaf2ae0e98362d3b7500db36350"},
//  "X-Secondlife-Local-Position" : {"(204.783539, 26.682831, 35.563702)"},
//  "X-Authtoken-Name" : {"TEST"},

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

//  Configuration info, from file
type vdbconfig struct {
	Mysql struct {
	    Domain   string // domain for MySQL database
		Database string // for MySQL database
		User     string
		Password string
	}
	Authkey map[string]string // auth keys
}

func (r vdbconfig) String() string {
	var s string = ""
	for k, _ := range r.Authkey { // all the authkey names, but not values
		s = s + " " + k
	}
	return (fmt.Sprintf("domain: %s  database: %s  user: %s authkeys: %s", r.Mysql.Domain, r.Mysql.Database, r.Mysql.User, s))
}
//
//  Package-local variables
//
var config vdbconfig;                   // local config, initialized once at startup

func expand(path string) (string, error) { // expand file paths with tilde for home dir
	if len(path) == 0 || path[0] != '~' {
		return path, nil
	}
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, path[1:]), nil
}

func initconfig(configpath string) (error) {
	configpath, err := expand(configpath) // get absolute path
	file, err := ioutil.ReadFile(configpath)
	if err != nil {
		return err
	}
	if err != nil {
		return err
	}
	err = json.Unmarshal(file, &config)         // config file is json
	return err
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
	reg.Name = strings.TrimSpace(s[0 : ix-1])               // name part
	_, err := fmt.Sscanf(s[ix:], "(%d,%d)", &reg.X, &reg.Y) // location part
	return reg, err
}

//  Parseslvector - parse forms such as "(204.783539, 26.682831, 35.563702)"
func Parseslvector(s string) (slvector, error) {
	var p slvector
	_, err := fmt.Sscanf(s, "(%f,%f,%f)", &p.X, &p.Y, &p.Z)
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
	err := json.Unmarshal(s, &ev) // decode JSON
	return ev, err
}

//
//  Validateauthtoken -- validate that string has correct hash for auth token
//
func Validateauthtoken(s []byte, name string, value string) error {
	token := config.Authkey[name]            // get auth token
	if token == "" {
		return errors.New(fmt.Sprintf("Logging authorization token %s not recognized.", name))
	}
	//  Do SHA1 check to validate that log entry is valid.
	valforhash := append([]byte(token),s...)
	hash := sha1.Sum(valforhash) // validate that SHA1 of token plus string matches
	fmt.Printf("Token: %s For hash: \"%s\"\n", token, valforhash);   // ***TEMP***
	if string(hash[:]) != value {
		return errors.New(fmt.Sprintf("Logging authorization token %s failed to validate.", name))
	}
	return (nil)
}

func Insertindb(db *sql.DB, hdr slheader, ev vehlogevent) error {
	var insstmt string = "INSERT INTO events  (time, owner_name, object_name, region_name, region_corner_x, region_corner_y, local_position_x, local_position_y, tripid, severity, eventtype, msg, auxval)  VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)"
	var args [13]interface{} // args go here
	args[0] = ev.Timestamp
	args[1] = hdr.Owner_name
	args[2] = hdr.Object_name
	args[3] = hdr.Region.Name
	args[4] = hdr.Region.X
	args[5] = hdr.Region.Y
	args[6] = hdr.Local_position.X
	args[7] = hdr.Local_position.Y
	args[8] = ev.Tripid
	args[9] = ev.Severity
	args[10] = ev.Eventtype
	args[11] = ev.Msg
	args[12] = ev.Auxval
	_, err := db.Exec(insstmt, args)
	return err
}

//
//  Addevent -- add an event to the database
//
func Addevent(s []byte, headervars http.Header, database *sql.DB) error {
	//  Validate auth token first
	err := Validateauthtoken(s,
		strings.TrimSpace(headervars.Get("X-Authtoken-Name")),
		strings.TrimSpace(headervars.Get("X-Authtoken-Value")))
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
