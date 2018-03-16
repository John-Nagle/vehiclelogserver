--
--  Vehicle database for recording region crossings of SL vehicles
--
--  Animats
--  March, 2018
--

CREATE DATABASE IF NOT EXISTS vehiclelog CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE vehiclelog;
--
--  Events -- raw events sent from vehicle script
--
CREATE TABLE IF NOT EXISTS events (
    serial          INT NOT NULL,               -- client side serial number
    time            BIGINT NOT NULL,            -- UNIX timestamp, client side, not server side
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
