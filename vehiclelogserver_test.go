//
//  Tests for vehicle log server
//
//  Animats
//  March, 2018
//
package main

import (
	"crypto/sha1" // cryptograpically weak, but SL still uses it
	"fmt"
	"net/http"
	"strings"
	"encoding/hex"
	"math/rand"
	"time"
	"testing"
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
	"X-Authtoken-Hash":            {"0bc935dbb51aaeaf2ae0e98362d3b7500db36350"},
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
var testjson1 = `{"timestamp":1234,"tripid":"TRIPID","severity":2,"type":"STARTUP","msg":"John Doe","auxval":1.0}`
var tokenname = "MAR2018"
var testjson2 = []byte(`{"tripid":"4c8650ab4ceeeddeb8d3e31ca950255cc22918b5","severity":1,"type":"TEST","msg":"Testing","auxval":0.000000,"timestamp":1521264571,"serial":4,"debug":99}`)
var testjson2hash = `0a62ed635221f9503ddb4315cf30a7ac0c3493c1`  // computed by SL script

var testsv *FastCGIServer // the server object

func TestInit(t *testing.T) {
	//  Reads the config file into variable "config".
	testsv = new(FastCGIServer)
	err := initdb("~/keys/vehicledbconf.json", testsv)
	if err != nil {
		testsv = nil
		t.Errorf(err.Error())
		return
	}
	fmt.Printf("Config: %s\n", testsv.config)
}

func TestSHA1Compat(t *testing.T) {
    //  From an SL script:
    //  Hash test. 'ABCD' hashes to 'fb2f85c88567f3c8ce9b799c7c54642d0c7b41f6'
    var t1 = "ABCD"                                             // input to hash
    var t1hash = "fb2f85c88567f3c8ce9b799c7c54642d0c7b41f6"     // expected hex output
    hash := sha1.Sum([]byte(t1))                                      // compute hash as binary bytes
	hashhex := hex.EncodeToString(hash[:])                      // convert to hex to match SL
	if t1hash != hashhex {
	    t.Errorf(fmt.Sprintf("Input: \"%s\" Hash result: \"%s\".  Expected \"%s\".", t1, hashhex, t1hash)) // ***TEMP***
	}
	fmt.Printf("Go SHA1 result matches LSL result.\n")
}

func TestTokenValidation(t *testing.T) {
	if testsv == nil {
		t.Errorf("Config file test failed, can't do next tests.")
		return
	}
	token := testsv.config.Authkey[tokenname]
	hash := Hashwithtoken([]byte(token[:]) , testjson2)
	fmt.Printf("Expected: \"%s\".  Calculated hash: \"%s\"", testjson2hash, hash)
	if hash != testjson2hash {
	    t.Errorf("Token hashes didn't match")
    }
}

func TestEventLog(t *testing.T) {
	if testsv == nil {
		t.Errorf("Config file test failed, can't do next tests.")
		return
	}
	rand.Seed( time.Now().UTC().UnixNano())
	//  Basic parsing test
	//  Make a unique trip ID - 40 chars of hex
	tripid1 := fmt.Sprintf("%d",(rand.Int63()))                                 // 63-bit random number
	triphash := sha1.Sum([]byte(tripid1))                                       // compute hash as binary bytes
	tripid := hex.EncodeToString(triphash[:])                                   // convert to hex to match SL
	testjson := []byte(strings.Replace(testjson1, "TRIPID", tripid,1))          // fill in a new trip ID
	//  Build properly signed test JSON
	var testkey []string
	testkey = append(testkey, "MAR2018")
	testheader1["X-Authtoken-Name"] = testkey
	token := testsv.config.Authkey[testkey[0]]
	hash := Hashwithtoken([]byte(token[:]), testjson)
	var hashes []string
	hashes = append(hashes, string(hash))
	testheader1["X-Authtoken-Hash"] = hashes
	err := Addevent(testjson, testheader1, testsv.config, testsv.db) // call with no database
	if err != nil {
		t.Errorf(err.Error())
	}
}
