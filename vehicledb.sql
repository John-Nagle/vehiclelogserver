--
--  Vehicle database for recording region crossings of SL vehicles
--
--  Animats
--  March, 2018
--

CREATE DATABASE IF NOT EXISTS vehicles CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE vehicles;

--
--  events -- raw events sent from vehicle script
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
	local_position_z  FLOAT NOT NULL DEFAULT -1.0,   -- turns out we need Z to detect falls
	tripid          CHAR(40) NOT NULL,          -- trip ID (random unique identifier)
	severity        TINYINT NOT NULL,           -- an enum, really
	eventtype       VARCHAR(20) NOT NULL,       -- STARTUP, SHUTDOWN, etc.
	msg             TEXT,                       -- human-readable message
	auxval          FLOAT NOT NULL,             -- some other value associated with the event type
	INDEX(tripid),
	UNIQUE INDEX(tripid, serial),               -- catch dups at insert time
	INDEX(eventtype)
) ENGINE InnoDB;

--
--  errorlog - internal error logging
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
--  tripstodo  -- trip IDs in events not yet summarized to trips
--  
CREATE TABLE IF NOT EXISTS tripstodo (
    tripid          CHAR(40) NOT NULL PRIMARY KEY,      -- trip ID 
    stamp           TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP -- last update
) ENGINE InnoDB;

--
--  trips -- info about trips
--
CREATE TABLE IF NOT EXISTS trips (
    stamp           TIMESTAMP NOT NULL,         -- end time of trip
    elapsed         INT NOT NULL,               -- elapsed time
    tripid          CHAR(40) NOT NULL,          -- ID of trip
    owner_name      VARCHAR(255) NOT NULL,      -- name of owner
    grid_name       VARCHAR(255) NOT NULL,      -- grid name
	object_name     VARCHAR(255) NOT NULL,      -- object name
	driver_key      CHAR(36) NOT NULL,          -- driver avatar key if available
	driver_name     VARCHAR(255) NOT NULL,      -- name of driver
	driver_display_name VARCHAR(255) NOT NULL,  -- display name of driver
	
	distance        FLOAT NOT NULL,             -- distance traveled, from client
	regions_crossed INT NOT NULL,               -- number of region crossings
	trip_status     ENUM("OK","FAULT","NOSHUTDOWN"), -- how did trip end?
	data_status     ENUM("OK","MISSING","INCONSISTENT"), -- data problems  
	severity        TINYINT NOT NULL,           -- worst severity level 
	start_region_name VARCHAR(255) NOT NULL,    -- starting region
	end_region_name VARCHAR(255) NOT NULL,      -- ending region
	min_x           FLOAT NOT NULL,             -- min X value, global
	min_y           FLOAT NOT NULL,             -- min Y value, global
	max_x           FLOAT NOT NULL,             -- max X value, global
    max_y           FLOAT NOT NULL,             -- max Y value, global
    last_eventtypes TEXT,                       -- last N event types recorded
	msg             TEXT,                       -- message if any
	INDEX(driver_name),
	INDEX(trip_status),
	INDEX(driver_key),
	UNIQUE INDEX(tripid)
) ENGINE InnoDB;
