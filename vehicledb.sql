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
    serial          BIGINT NOT NULL AUTO_INCREMENT, -- serial number
    time            TIMESTAMP NOT NULL,         -- client side, not server side
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
	UNIQUE INDEX(serial),
	INDEX(tripid),
	INDEX(eventtype)
) ENGINE InnoDB;
