--
--  Vehicle database for recording region crossings of SL vehicles
--
--  Animats
--  March, 2018
--

CREATE DATABASE IF NOT EXISTS vehicles CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE vehicles;
--
--  Events -- raw events sent from vehicle script
--
CREATE TABLE IF NOT EXISTS events (
    serial          INT NOT NULL,               -- client side serial number
    time            BIGINT NOT NULL,            -- UNIX timestamp, client side, not server side
    shard           VARCHAR(255) NOT NULL,      -- server shard
	owner_name      VARCHAR(255) NOT NULL,      -- name of owner
	object_name     VARCHAR(255) NOT NULL,      -- object name
	region_name     VARCHAR(255) NOT NULL,      -- name of region
	region_corner_x   INT NOT NULL,	            -- corner of region
	region_corner_Y   INT NOT NULL,	            -- corner of region
	local_position_x  FLOAT NOT NULL,           -- X and Y only
	local_position_y  FLOAT NOT NULL,           -- X and Y only
	tripid          CHAR(40) NOT NULL,          -- trip ID (random unique identifier)
	severity        TINYINT NOT NULL,           -- an enum, really
	eventtype       VARCHAR(20) NOT NULL,       -- STARTUP, SHUTDOWN, etc.
	msg             TEXT,                       -- human-readable message
	auxval          FLOAT NOT NULL,             -- some other value associated with the event type
	                                            -- summarization status, set by summarizer
	summary         ENUM('unchecked', 'complete', 'incomplete', 'junk') NOT NULL DEFAULT 'unchecked',
	INDEX(tripid),
	UNIQUE INDEX(tripid, serial),               -- catch dups at insert time
	INDEX(eventtype),
	INDEX(summary)
) ENGINE InnoDB;
--
--  Errors -- error log
--
CREATE TABLE IF NOT EXISTS errorlog (
    stamp           TIMESTAMP,                  -- automatic timestamp
    owner_name      VARCHAR(255) DEFAULT NULL,  -- owner if relevant
    tripid          CHAR(40) DEFAULT NULL,      -- trip ID if relevant
    msg             TEXT,                       -- error message
    INDEX(owner_name),
    INDEX(tripid)
) ENGINE InnoDB;
--
--  Trips -- info about trips
--
CREATE TABLE IF NOT EXISTS trips (
    date            TIMESTAMP NOT NULL,         -- time of trip
    tripid          CHAR(40) NOT NULL,          -- ID of trip
    owner_name      VARCHAR(255) NOT NULL,      -- name of owner
    grid_name       VARCHAR(255) NOT NULL,      -- grid name
	object_name     VARCHAR(255) NOT NULL,      -- object name
	driver_name     VARCHAR(255) NOT NULL,      -- name of driver
	driver_display_name VARCHAR(255) NOT NULL,  -- display name of driver
	
	distance        FLOAT NOT NULL,             -- distance traveled
	regions_crossed INT NOT NULL,               -- number of region crossings
	regions_entered INT NOT NULL,               -- number of regions entered
	trip_status     ENUM("OK","FAULT","NOSHUTDOWN"), -- how did trip end?   
	start_pos       POINT NOT NULL,             -- starting position
	end_pos         POINT NOT NULL,             -- ending position
	sw_corner       POINT NOT NULL,             -- lower left limit of travel
	ne_corner       POINT NOT NULL,             -- upper right limit of travel
	msg             TEXT,                       -- message if any
	INDEX(driver_name),
	SPATIAL INDEX(sw_corner),
	SPATIAL INDEx(ne_corner)

) ENGINE InnoDB;
