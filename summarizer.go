//
//  summarizer --  summarize vehicle log events into a summary file
//
//  Animats
//  March, 2018
//
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

//
//  Constants
//
const runEverySecs = 30      // run this no more than once per N seconds
const minSummarizeSecs = 120 // summarize if oldest event is older than this 
const keeplasteventtypes = 6 // keep this many event types in log

//
//  Static variables
//
var lastSummarizeTime time.Time // last time we ran summarization. Zero at init

//
//  Types
//
type trip struct { // used during summarization
	serial         int32       // record serial number
	prevpos        slglobalpos // global position
	event_distance float64     // distance computed from events as check
	starttime      int64       // starting time, UNIX
	sx             tripsummary // trip summary to go to database
}
type tripsummary struct {

	// trip summary data to go to database
	stamp               time.Time   // end time of trip
	elapsed             int32       // elapsed time
	tripid              string      // ID of trip
	owner_name          string      // name of owner
	shard               string      // grid name
	object_name         string      // object name
	driver_key          string      // 36 chars of key, may be empty
	driver_name         string      // name of driver
	driver_display_name string      // display name of driver
	distance            float64     // distance traveled, from client
	regions_crossed     int32       // number of region crossings
	trip_status         string      // ENUM("OK","FAULT","NOSHUTDOWN"), // how did trip end?
	data_status         string      // ENUM("OK","MISSING","INCONSISTENT"), // data problems
	severity            int8        // worst severity level
	start_region_name   string      // starting region
	end_region_name     string      // ending region
	min_pos             slglobalpos // min X value, global
	max_pos             slglobalpos // max X value, global
	last_eventtypes     []string    // last N event types recorded
	msg                 string      // message if any
}

func (r tripsummary) String() string {
	return fmt.Sprintf("tripid: \"%s\"  driver_name: \"%s\"  object_name: \"%s\"  status: %s  data: %s  regions: %d",
		r.tripid,
		r.driver_name, r.object_name,
		r.trip_status, r.data_status,
		r.regions_crossed)
}

func (r trip) String() string {
	return fmt.Sprintf("%s  distance from events %1.2fkm",
		r.sx,
		r.event_distance/1000.0)
}

//
//  inserttrip -- insert trip info in database
//
//  Ignore duplicates
//
func inserttrip(db *sql.DB, r tripsummary) error {
	//   Convert last eventtypes into TYPE-TYPE-TYPE for SQL
	const insstmt string = "INSERT IGNORE INTO trips (stamp, elapsed, tripid, owner_name, shard, object_name, driver_key, driver_name, driver_display_name, distance, regions_crossed, trip_status, data_status, severity, start_region_name, end_region_name, min_pos_x, min_pos_y, max_pos_x, max_pos_y, last_eventtypes, msg) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)"
	_, err := db.Exec(insstmt,
		r.stamp,
		r.elapsed,
		r.tripid,
		r.owner_name,
		r.shard,
		r.object_name,
		r.driver_key,
		r.driver_name,
		r.driver_display_name,
		r.distance,
		r.regions_crossed,
		r.trip_status,
		r.data_status,
		r.severity,
		r.start_region_name,
		r.end_region_name,
		r.min_pos.X,
		r.min_pos.Y,
		r.max_pos.X,
		r.max_pos.Y,
		strings.Join(r.last_eventtypes, ", "),
		r.msg)
	return err
}

//
//  deletetodo  -- delete to-do entry from to-do list
//
func deletetodo(db *sql.DB, tripid string) error {
	if tripid == "" {
		return (errors.New("deletetodo: empty tripid"))
	}
	_, err := db.Exec("DELETE FROM tripstodo WHERE tripid = ?", tripid)
	return (err)
}

//
//  updatetripdb  -- update trip database from trip record
//
//  Also deletes corresponding record from tripstodo.
//
//  Duplicate tripid - ignore update
//
func updatetripdb(db *sql.DB, r tripsummary) error {
	tx, err := db.Begin() // updating events and tripstodo
	if err != nil {
		return err
	}
	err = inserttrip(db, r)
	if err == nil {
		err = deletetodo(db, r.tripid)
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
//  updatefromeevent -- update trip object given a log entry
//
func (r *trip) updatefromevent(event vehlogevent, hdr slheader, first bool) {
	var gpos slglobalpos
	gpos.Set(hdr.Region, hdr.Local_position) // where we are
	if first {                               // first record, must be "STARTUP"
		if event.Eventtype != "STARTUP" || event.Serial != 0 { // not a good first record
			r.sx.data_status = "MISSING"
		} else {
			r.sx.data_status = "OK"
			r.sx.object_name = hdr.Object_name
			r.sx.owner_name = hdr.Owner_name
			r.sx.object_name = hdr.Object_name
			r.sx.shard = hdr.Shard
			names := strings.SplitN(event.Msg, "/", 2) // split into legacy name / display name
			if len(names) == 2 {
				r.sx.driver_name = names[0]
				r.sx.driver_display_name = names[1]
			}
		}
		r.sx.trip_status = "OK"
		r.sx.regions_crossed = 0
		r.event_distance = 0.0
		r.serial = -1
		r.starttime = event.Timestamp                             // start time
		r.sx.tripid = event.Tripid
		r.sx.severity = event.Severity
		r.sx.start_region_name = hdr.Region.Name
		r.sx.min_pos = gpos // update corners of area traveled
		r.sx.max_pos = gpos
		r.prevpos = gpos

	}
	//  Special cases
	if event.Eventtype == "DRIVERKEY" { // DRIVERKEY event contains key in msg field
		key := strings.TrimSpace(event.Msg) // compatibility with 2018 name change plan
		if len(key) == 36 && r.sx.driver_key == "" {
			r.sx.driver_key = key // save key
		}
	}

	//  For all records
	//  Consistency checks
	consistent := hdr.Owner_name == r.sx.owner_name && hdr.Object_name == r.sx.object_name &&
		hdr.Shard == r.sx.shard
	sequential := r.serial+1 == event.Serial // should be in sequence
	if r.sx.data_status == "OK" && !consistent {
		r.sx.data_status = "INCONSISTENT"
	}
	if r.sx.data_status == "OK" && !sequential {
		r.sx.data_status = "MISSING"
	}
	r.serial = event.Serial
	//  Significant bad event?
	trouble := strings.Contains(event.Eventtype, "FAIL") || strings.Contains(event.Eventtype, "ERR")
	if trouble && r.sx.trip_status == "OK" {
		r.sx.trip_status = "FAULT"
	}
	//  Distance calc
	if hdr.Region.Name != r.sx.end_region_name { // region crossing
		r.sx.regions_crossed++ // tally
	}
	r.sx.end_region_name = hdr.Region.Name
	r.sx.min_pos.Min(gpos) // update corners of area traveled
	r.sx.max_pos.Max(gpos)
	r.event_distance += r.prevpos.Distance(gpos)                         // accumulate distance
	r.prevpos = gpos                                                     // previous position
	r.sx.last_eventtypes = append(r.sx.last_eventtypes, event.Eventtype) // recent event types (could truncate this)
}

//
//  doonetrpiid  -- handle one trip ID
//
func doonetripid(db *sql.DB, tripid string, stamp time.Time, verbose bool) error {
	if verbose {
		fmt.Printf("Summarizing trip %s (%s)\n", tripid, stamp)
	}
	//  Read events for this trip in serial order
	rows, err := db.Query("SELECT tripid, time, shard, owner_name, object_name, region_name, region_corner_x, region_corner_y, local_position_x, local_position_y, local_position_z, severity, eventtype, msg, auxval, serial FROM events WHERE tripid = ? ORDER BY serial", tripid)
	if err != nil {
		return err
	}
	defer rows.Close()
	var tr trip           // working trip
	var first bool = true // first
	var lastevent vehlogevent

	for rows.Next() { // over all rows
		var event vehlogevent
		var hdr slheader
		err = rows.Scan(&event.Tripid, &event.Timestamp, &hdr.Shard, &hdr.Owner_name, &hdr.Object_name, &hdr.Region.Name, &hdr.Region.X, &hdr.Region.Y,
			&hdr.Local_position.X, &hdr.Local_position.Y, &hdr.Local_position.Z,
			&event.Severity, &event.Eventtype, &event.Msg, &event.Auxval, &event.Serial)
		if verbose {
			fmt.Printf("%4d. %12s %s %s %s %f\n", event.Serial, event.Eventtype, hdr.Region.Name, hdr.Local_position, event.Msg, event.Auxval)
		}
		tr.updatefromevent(event, hdr, first)
		//  Save last event
		first = false
		lastevent = event
	}
	//  Last event processing
	if lastevent.Eventtype == "SHUTDOWN" {
		tr.sx.distance = float64(lastevent.Auxval) // get distance traveled
	} else {
		if tr.sx.trip_status == "OK" {
			tr.sx.trip_status = "NOSHUTDOWN" // log ended incomplete
		}
	}
	if len(tr.sx.last_eventtypes) > keeplasteventtypes {
		tr.sx.last_eventtypes = tr.sx.last_eventtypes[len(tr.sx.last_eventtypes)-keeplasteventtypes:] // keep last N
	}
	tr.sx.elapsed = int32(lastevent.Timestamp - tr.starttime)  // elapsed time
	tr.sx.stamp = stamp                                         // timestamp trip (end time)
	if verbose {
		fmt.Printf("Summary: %s\n", tr)
	}
	err = updatetripdb(db, tr.sx) // update the database
	return err
}

//
//  dosummarize -- run a summarize cycle if not run recently
//
func dosummarize(db *sql.DB, verbose bool) error {

	if !lastSummarizeTime.IsZero() && time.Since(lastSummarizeTime).Seconds() < minSummarizeSecs {
		return nil // too soon, try later
	}
	lastSummarizeTime = time.Now() // update time stamp

	if verbose {
		fmt.Printf("Starting summarization.\n")
	}

	for { // unti no more work to do
		//  Get earliest tripid at least minSummarizeSeconds old.
		//  We do this one at a time because there might be other summarizers running.
		row := db.QueryRow("SELECT tripid, stamp FROM tripstodo WHERE TIMESTAMPDIFF(SECOND, stamp, NOW()) > ? ORDER BY stamp LIMIT 1", minSummarizeSecs)
		var tripid string // trip ID to be processed
		var stamp time.Time
		err := row.Scan(&tripid, &stamp)
		if err == sql.ErrNoRows {
			if verbose {
				fmt.Printf("Done.\n")
			}
			break
		} // normal EOF
		if err != nil {
			return err
		}
		if err != nil {
			return err
		}
		err = doonetripid(db, tripid, stamp, verbose)
		if err != nil {
			return err
		}
		time.Sleep(500 * time.Millisecond) // avoid overloading server
	}
	return nil // normal end
}
