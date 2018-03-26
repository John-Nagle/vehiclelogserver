//
//  Tests for vehicle log server
//
//  Animats
//  March, 2018
//
package main

import (
	"crypto/sha1" // cryptograpically weak, but SL still uses it
	"encoding/csv"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
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
var testjson2hash = `0a62ed635221f9503ddb4315cf30a7ac0c3493c1` // computed by SL script

var testsv *FastCGIServer     // the server object
var verbose = false           // extra debug msgs if true
var testtokenname = "MAR2018" // use this named crypto token for testing

//  Test data row - a string in JSON format, and a HTTP header

type testrow struct {
	hdr  http.Header // a HTTP header
	json []byte      // JSON
}

var randominit = false // random number generator seeded?
func GenerateRandomTripid() string {
	if !randominit { // get low-grade pseudorandomness if needed
		rand.Seed(time.Now().UTC().UnixNano())
		randominit = true
	}
	tripid1 := fmt.Sprintf("%d", (rand.Int63())) // 63-bit random number
	triphash := sha1.Sum([]byte(tripid1))        // compute hash as binary bytes
	tripid := hex.EncodeToString(triphash[:])    // convert to hex to match SL
	return (tripid)
}

func SignLogMsg(testjson []byte, hdr http.Header, tokenname string) { // sign log msg to pass validation
	//  Build properly signed test JSON
	var testkey []string
	testkey = append(testkey, tokenname)
	testheader1["X-Authtoken-Name"] = testkey
	token := testsv.config.Authkey[testkey[0]]
	hash := Hashwithtoken([]byte(token[:]), testjson)
	var hashes []string
	hashes = append(hashes, string(hash))
	hdr["X-Authtoken-Hash"] = hashes
}

//
//  Read tab-delimited file of live data, generate arrays of data for testing
//
func ReadTabTestData(filename string) ([]testrow, error) {
	rows := make([]testrow, 0) // rows of test data
	// read data from tab-delimited file
	csvFile, err := os.Open("./testdata.txt")
	if err != nil {
		return rows, err
	}
	defer csvFile.Close()
	reader := csv.NewReader(csvFile)
	reader.Comma = '\t' // Use tab-delimited instead of comma <---- here!
	reader.FieldsPerRecord = -1
	csvdata, err := reader.ReadAll() // test file is not huge
	if err != nil {
		return rows, err
	}
	//  Put data into table of testrows
	tripid := GenerateRandomTripid() // generate a random trip ID
	for i := range csvdata {
		var ritem testrow // working test row
		ritem.hdr = make(map[string]([]string))
		row := csvdata[i]
		if verbose || i == 0 {
			fmt.Printf("Test entry")
			for j := range row {
				fmt.Printf(" %d=\"%s\"", j, row[j])
			}
			fmt.Printf("\n")
		}
		var v vehlogevent // logger data
		v.Timestamp, err = strconv.ParseInt(row[1], 10, 64)
		if err != nil { // file may have header lines
			continue
		} // skip them
		sr, _ := strconv.ParseInt(row[0], 10, 32)
		v.Serial = int32(sr)
		v.Tripid = row[10]
		sv, _ := strconv.ParseInt(row[11], 10, 8)
		v.Severity = int8(sv)
		v.Eventtype = row[12]
		v.Msg = row[13]
		au, _ := strconv.ParseFloat(row[14], 32)
		v.Auxval = float32(au)
		v.Debug = 0
		ritem.hdr["X-Secondlife-Shard"] = append(make([]string, 0), row[2])
		ritem.hdr["X-Secondlife-Owner-Name"] = append(make([]string, 0), row[3])
		ritem.hdr["X-Secondlife-Object-Name"] = append(make([]string, 0), row[4])
		ritem.hdr["X-Secondlife-Region"] = append(make([]string, 0), fmt.Sprintf("%s (%s,%s)", row[5], row[6], row[7]))
		ritem.hdr["X-Secondlife-Local-Position"] = append(make([]string, 0), fmt.Sprintf("(%s,%s,0.0)", row[8], row[9]))
		//  Adjust test data
		v.Tripid = tripid // use random tripID
		//  Make up JSON
		json, err := json.Marshal(v) // convert to JSON
		if err != nil {
			return rows, err
		}
		if verbose || i == 0 {
			fmt.Printf("JSON: %s\n", json)
		}
		//  Sign JSON
		ritem.hdr["X-Authtoken-Name"] = append(make([]string, 0), testtokenname)
		ritem.json = []byte(json)
		SignLogMsg(ritem.json, ritem.hdr, testtokenname)
		rows = append(rows, ritem) // add new row

	}
	return rows, nil
}

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
	var t1 = "ABCD"                                         // input to hash
	var t1hash = "fb2f85c88567f3c8ce9b799c7c54642d0c7b41f6" // expected hex output
	hash := sha1.Sum([]byte(t1))                            // compute hash as binary bytes
	hashhex := hex.EncodeToString(hash[:])                  // convert to hex to match SL
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
	hash := Hashwithtoken([]byte(token[:]), testjson2)
	if hash != testjson2hash {
		fmt.Printf("Expected: \"%s\".  Calculated hash: \"%s\"\n", testjson2hash, hash)
		t.Errorf("Token hashes didn't match")
	}
}

func TestEventLog(t *testing.T) {
	if testsv == nil {
		t.Errorf("Config file test failed, can't do next tests.")
		return
	}
	//  Basic parsing test
	//  Make a unique trip ID - 40 chars of hex
	tripid := GenerateRandomTripid()
	testjson := []byte(strings.Replace(string(testjson2), "TRIPID", tripid, 1)) // fill in a new trip ID
	//  Build properly signed test JSON
	var testkey []string
	testkey = append(testkey, "MAR2018")
	testheader1["X-Authtoken-Name"] = testkey
	token := testsv.config.Authkey[testkey[0]]
	hash := Hashwithtoken([]byte(token[:]), testjson)
	var hashes []string
	hashes = append(hashes, string(hash))
	SignLogMsg(testjson, testheader1, testtokenname)
	err := Addevent(testjson, testheader1, testsv.config, testsv.db) // call with no database
	if err != nil {
		t.Errorf(err.Error())
	}
}

func TestEventLogFromFile(t *testing.T) {
	var testfile = "testfile.txt" // tab-delimited data from real vehicle run
	rows, err := ReadTabTestData(testfile)
	if err != nil {
		t.Errorf(err.Error())
	}
	for i := range rows {
		row := rows[i] // get row of data
		err := Addevent(row.json, row.hdr, testsv.config, testsv.db)
		if err != nil {
			t.Errorf(err.Error())
			return
		}
	}
}

func TestSummarize(t *testing.T) {
	err := dosummarize(testsv.db, true)
	if err != nil {
		t.Errorf(err.Error())
		return
	}

}
