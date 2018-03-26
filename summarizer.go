//
//  summarizer  -- summarize vehicle log events into a summary file
//
//  Animats
//  March, 2018
//
package main

import (
	"database/sql"
	////"errors"
	"fmt"
	"time"
)

//
//  Constants
//
const runEverySecs = 30                      // run this no more than once per N seconds
const minSummarizeSecs = 5 ////120                 // summarize if oldest event is older than this
const Format3339Nano = "2006-01-02 15:04:05" // ought to be predefined
//
//  Static variables
//
var lastSummarizeTime time.Time // last time we ran summarization. Zero at init

//
//  doonetrpiid  -- handle one trip ID
//
func doonetripid(db *sql.DB, tripid string, stamp time.Time, verbose bool) error {
	if verbose {
		fmt.Printf("Summarizing trip %s (%s)\n", tripid, stamp)
	}
    //  Read events for this trip in serial order
    rows, err := db.Query("SELECT time, shard, owner_name, object_name, region_name, region_corner_x, region_corner_y, local_position_x, local_position_y, local_position_z, severity, eventtype, msg, auxval, serial FROM events WHERE tripid = ? ORDER BY serial", tripid)
	if err != nil {
		return err
	}
    defer rows.Close()
    for rows.Next () {                          // over all rows
        var event vehlogevent
        var hdr slheader   
        err = rows.Scan(&event.Timestamp, &hdr.Shard, &hdr.Owner_name, &hdr.Object_name, &hdr.Region.Name, &hdr.Region.X, &hdr.Region.Y, 
            &hdr.Local_position.X, &hdr.Local_position.Y, &hdr.Local_position.Z, 
            &event.Severity, &event.Eventtype, &event.Msg, &event.Auxval, &event.Serial)
        if verbose {
            fmt.Printf("%4d. %12s %s %s %s %f\n", event.Serial, event.Eventtype, hdr.Region.Name, hdr.Local_position, event.Msg, event.Auxval)
        }
    }
    //  ***SUMMARIZE AND UPDATE***
    _, err = db.Exec("DELETE FROM tripstodo WHERE tripid = ?", tripid)
	    
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
		var stampstr string
		err := row.Scan(&tripid, &stampstr)
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
		stamp, err := time.Parse(Format3339Nano, stampstr)
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
