//
//  summarizer  -- summarize vehicle log events into a summary file
//
//  Animats
//  March, 2018
//
package main

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

//
//  Constants
//
const runEverySecs = 30                      // run this no more than once per N seconds
const minSummarizeSecs = 120                 // summarize if oldest event is older than this
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
	return errors.New("Not implemented yet.")
	return nil
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
		//  Get earliest tripid at least minSummarizeSeconds old
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
