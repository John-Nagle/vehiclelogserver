//
//  Tests for vehicle log server
//
//  Animats
//  March, 2018
//
package vehiclelogserver

import (
	"fmt"
	"net/http"
	"crypto/sha1" // cryptograpically weak, but SL still uses it
	"testing"
	////"database/sql"
)

//
//  Test data
//
var testheader1 = http.Header{
	"Accept-Encoding":             {"deflate, gzip"},
	"Content-Type":                {"text/plain; charset=utf-8"},
	"X-Forwarded-For":             {"127.0.0.1"},
	"X-Secondlife-Local-Velocity": {"(0.000000, 0.000000, 0.000000)"},
	"X-Secondlife-Owner-Key":      {"dadec334-539a-4875-ad0e-d9654705f437"},
	"X-Secondlife-Shard":          {"Production"},
	"X-Secondlife-Object-Name":    {"Logging tester 0.4"},
	"Pragma":                      {"no-cache"},
	"X-Secondlife-Owner-Name":     {"animats Resident "},
	"X-Secondlife-Object-Key":     {"b23730f8-4105-594a-c359-e72f9fece699"},
	"X-Secondlife-Region":         {"Vallone (462592, 306944)"},
	"X-Authtoken-Value":           {"0bc935dbb51aaeaf2ae0e98362d3b7500db36350"},
	"Connection":                  {"close"},
	"Accept":                      {"text/*", "application/xhtml+xml", "application/atom+xml", "application/json", "application/xml", "application/llsd+xml", "application/x-javascript", "application/javascript", "application/x-www-form-urlencoded", "application/rss+xml"},
	"X-Secondlife-Local-Position": {"(204.783539, 26.682831, 35.563702)"},
	"X-Authtoken-Name":            {"TEST"},
	"X-Secondlife-Local-Rotation": {"(0.000000, 0.000000, 0.000000, 1.000000)"},
	"Via":            {"1.1 sim10317.agni.lindenlab.com:3128 (squid/2.7.STABLE9)"},
	"Cache-Control":  {"max-age=259200"},
	"Accept-Charset": {"utf-8;q=1.0, *;q=0.5"}}

var testjson0 = []byte(`{"event":"Touched","driver":"animats Resident","drivername":"Joe Magarac"}`)

// logdata = logdata + ["tripid"] + gTripId + ["severity"] + severity + ["type"] + msgtype + ["msg"] + msg + ["auxval"] + val;
var testjson1 = []byte(`{"timestamp":1234,"tripid":"ABCDEF","severity":2,"type":"STARTUP","msg":"John Doe","auxval":1.0}`)

func TestConfigRead(t *testing.T) {
    //  Reads the config file into variable "config".
	err := initconfig("~/keys/vehicledbconf.json")
	if err != nil {
		t.Errorf(err.Error())
	}
	fmt.Printf("Config: %s\n", config)
}

func TestDatabaseConnection(t *testing.T) {
	t.Errorf("Unimplemented")
}

func TestEventLog(t *testing.T) {
	//  Basic parsing test
	//  Build properly signed test JSON
	var testkey []string;
	testkey = append(testkey,"MAR2018")
	testheader1["X-Authtoken-Name"] = testkey
	token := config.Authkey[testkey[0]]
	valforhash := append([]byte(token),testjson1...)
	fmt.Printf("Token: %s For hash: \"%s\"\n", token, valforhash);   // ***TEMP***
	hash := sha1.Sum([]byte(valforhash)) // validate that SHA1 of token plus string matches
	var hashes []string;
	hashes = append(hashes, string(hash[:]))
	testheader1["X-Authtoken-Value"] = hashes
	err := Addevent(testjson1, testheader1, nil) // call with no database
	if err != nil {
		t.Errorf(err.Error())
	}
}
