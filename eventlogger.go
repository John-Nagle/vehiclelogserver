//
//  eventlogger -- adds events to the "event" table of the vehicle database
//
//  Animats
//  March, 2018
//
//
//  TODO:
//  - Add JSON fields "echo", "timestamp", and "serial", in client and server. [DONE]
//  - Add clock skew check for timestamp.
//  - Add logging of grid
//
package main

import (
	"crypto/sha1" // cryptograpically weak, but SL still uses it
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"math"
	"net/http"
	"os/user"
	"path/filepath"
	"strings"
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

type slglobalpos struct {
	X float64
	Y float64
}

func (r *slglobalpos) Set(region slregion, pos slvector) {
	r.X = float64(region.X) + float64(pos.X) // region corner plus local offset
	r.Y = float64(region.Y) + float64(pos.Y)
}

func (r *slglobalpos) Min(t slglobalpos) {
	r.X = math.Min(r.X, t.X)
	r.Y = math.Min(r.Y, t.Y)
}

func (r *slglobalpos) Max(t slglobalpos) {
	r.X = math.Max(r.X, t.X)
	r.Y = math.Max(r.Y, t.Y)
}

func (r *slglobalpos) Distance(t slglobalpos) float64 {
	dx := r.X - t.X
	dy := r.Y - t.Y
	return math.Sqrt(dx*dx + dy*dy) // distance, of course
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
	Shard          string   // which grid
	Object_name    string   // object name
	Region         slregion // SL region name and corner
	Local_position slvector // position within region
}

func (r slheader) String() string {
	return fmt.Sprintf("owner_name: \"%s\"  object_name: \"%s\"  region: %s  local_position: %s", r.Owner_name, r.Object_name, r.Region, r.Local_position)
}

type vehlogevent struct {
	Timestamp int64   // UNIX timestamp, long form
	Serial    int32   // serial number from client
	Tripid    string  // trip ID (random unique identifier)
	Severity  int8    // an enum, really
	Eventtype string  // STARTUP, SHUTDOWN, etc.
	Msg       string  // human-readable message
	Auxval    float32 // some other value associated with the event
	Debug     int8    // logging level
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

func readconfig(configpath string) (vdbconfig, error) {
	var config vdbconfig
	configpath, err := expand(configpath) // get absolute path
	file, err := ioutil.ReadFile(configpath)
	if err != nil {
		return config, err
	}
	if err != nil {
		return config, err
	}
	err = json.Unmarshal(file, &config) // config file is json
	return config, err
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

func Getheaderfield(headervars http.Header, key string) (string, error) {
	s := strings.TrimSpace(headervars.Get(key))
	if s == "" {
		return s, errors.New(fmt.Sprintf("HTTP header from Second Life was missing field \"%s\"", key))
	}
	return s, nil
}

func Parseheader(headervars http.Header) (slheader, error) {
	var hdr slheader
	var err error
	hdr.Owner_name, err = Getheaderfield(headervars, "X-Secondlife-Owner-Name")
	if err != nil {
		return hdr, err
	}
	hdr.Object_name, err = Getheaderfield(headervars, "X-Secondlife-Object-Name")
	if err != nil {
		return hdr, err
	}
	hdr.Shard, err = Getheaderfield(headervars, "X-Secondlife-Shard")
	if err != nil {
		return hdr, err
	}
	hdr.Region, err = Parseslregion(headervars.Get("X-Secondlife-Region"))
	if err != nil {
		return hdr, err
	}
	hdr.Local_position, err = Parseslvector(headervars.Get("X-Secondlife-Local-Position"))
	if err != nil {
		return hdr, err
	}
	////fmt.Printf("Parseheader: %s\n", hdr) // ***TEMP***
	return hdr, nil
}

func Parsevehevent(s []byte) (vehlogevent, error) {
	var ev vehlogevent
	err := json.Unmarshal(s, &ev) // decode JSON
	if err != nil {
		return ev, err
	}
	if len(ev.Tripid) != 40 { // must be length of SHA1 hash in hex
		return ev, errors.New(fmt.Sprintf("Trip ID \"%s\" from Second Life was not 40 bytes long", ev.Tripid))
	}
	return ev, err
}

func Hashwithtoken(token []byte, s []byte) string { // our SHA1 validation - must match SL's only secure hash algorithm
	valforhash := append([]byte(token), s...)
	hash := sha1.Sum(valforhash)           // compute hash as binary bytes
	hashhex := hex.EncodeToString(hash[:]) // convert to hex to match SL
	return hashhex
}

//
//  validateauthtoken -- validate that string has correct hash for auth token
//
func Validateauthtoken(s []byte, name string, value string, config vdbconfig) error {
	token := config.Authkey[name] // get auth token
	if token == "" {
		return errors.New(fmt.Sprintf("Logging authorization token \"%s\" not recognized.", name))
	}
	//  Do SHA1 check to validate that log entry is valid.
	hash := Hashwithtoken([]byte(token), []byte(s))
	if hash != value {
		return errors.New(fmt.Sprintf("Logging authorization token \"%s\" failed to validate.\nText: \"%s\"\nHash sent: \"%s\"\nHash calc: \"%s\"",
			name, s, value, hash))
	}
	return (nil)
}

func insertevent(db *sql.DB, hdr slheader, ev vehlogevent) error {
	const insstmt string = "INSERT INTO events  (time, shard, owner_name, object_name, region_name, region_corner_x, region_corner_y, local_position_x, local_position_y, local_position_z, tripid, severity, eventtype, msg, auxval, serial)  VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	_, err := db.Exec(insstmt,
		ev.Timestamp,
		hdr.Shard,
		hdr.Owner_name,
		hdr.Object_name,
		hdr.Region.Name,
		hdr.Region.X,
		hdr.Region.Y,
		hdr.Local_position.X,
		hdr.Local_position.Y,
		hdr.Local_position.Z,
		ev.Tripid,
		ev.Severity,
		ev.Eventtype,
		ev.Msg,
		ev.Auxval,
		ev.Serial)
	return err
}

//
//  inserttodo -- update to-do list of trips in progress
//
func inserttodo(db *sql.DB, tripid string) error {
	const insstmt string = "INSERT INTO tripstodo (tripid) VALUES (?) ON DUPLICATE KEY UPDATE stamp=NOW()"
	_, err := db.Exec(insstmt, tripid)
	return err
}

//
//  dbupdate -- do the database updates to insert an event
//
func dbupdate(db *sql.DB, hdr slheader, ev vehlogevent) error {
	tx, err := db.Begin() // updating events and tripstodo
	if err != nil {
		return err
	}
	err = insertevent(db, hdr, ev)
	if err == nil {
		err = inserttodo(db, ev.Tripid)
		if err == nil {
			err = tx.Commit() // success
			if err != nil {
				return err
			}

		} // all OK, commit
	}
	if err != nil {
		_ = tx.Rollback() // fail, undo
	}
	return err
}

//
//  Addevent -- add an event to the database
//
func Addevent(bodycontent []byte, headervars http.Header, config vdbconfig, db *sql.DB) error {
	//  Validate auth token first
	err := Validateauthtoken(bodycontent,
		strings.TrimSpace(headervars.Get("X-Authtoken-Name")),
		strings.TrimSpace(headervars.Get("X-Authtoken-Hash")),
		config)
	if err != nil {
		return (err)
	}
	hdr, err := Parseheader(headervars) // parse HTTP header
	if err != nil {
		return err
	}
	ev, err := Parsevehevent(bodycontent) // parse JSON from vehicle script
	if err != nil {
		return err
	}
	return (dbupdate(db, hdr, ev)) // insert in database
}

//  Handlerequest -- handle a request from a client
func Handlerequest(sv FastCGIServer, w http.ResponseWriter, bodycontent []byte, req *http.Request) {
	err := Addevent(bodycontent, req.Header, sv.config, sv.db)
	if err == nil {
	    err = dosummarize(sv.db, false)             // do summarization
	}
	if err != nil {
		w.WriteHeader(500)           // internal server error
		w.Write([]byte(err.Error())) // report error as text ***TEMP***
		w.Write([]byte("\n"))
		////dumprequest(sv, w, req, bodycontent) // dump entire request as text ***TEMP***
	}
}
