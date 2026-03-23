CREATE SCHEMA IF NOT EXISTS "bookings";

CREATE SEQUENCE "bookings"."flights_flight_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE TABLE "bookings"."aircrafts_data" (
	"aircraft_code" bpchar NOT NULL,
	"model" jsonb NOT NULL,
	"range" integer NOT NULL,
	CONSTRAINT "aircrafts_pkey" PRIMARY KEY("aircraft_code"),
	CONSTRAINT "aircrafts_range_check" CHECK(range > 0)
);

CREATE TABLE "bookings"."airports_data" (
	"airport_code" bpchar NOT NULL,
	"airport_name" jsonb NOT NULL,
	"city" jsonb NOT NULL,
	"coordinates" point NOT NULL,
	"timezone" text NOT NULL,
	CONSTRAINT "airports_data_pkey" PRIMARY KEY("airport_code")
);

CREATE TABLE "bookings"."boarding_passes" (
	"ticket_no" bpchar NOT NULL,
	"flight_id" integer NOT NULL,
	"boarding_no" integer NOT NULL,
	"seat_no" varchar(4) NOT NULL,
	CONSTRAINT "boarding_passes_pkey" PRIMARY KEY("ticket_no", "flight_id"),
	CONSTRAINT "boarding_passes_flight_id_boarding_no_key" UNIQUE("flight_id", "boarding_no"),
	CONSTRAINT "boarding_passes_flight_id_seat_no_key" UNIQUE("flight_id", "seat_no")
);

CREATE TABLE "bookings"."bookings" (
	"book_ref" bpchar NOT NULL,
	"book_date" timestamptz NOT NULL,
	"total_amount" numeric(10,2) NOT NULL,
	CONSTRAINT "bookings_pkey" PRIMARY KEY("book_ref")
);

CREATE TABLE "bookings"."flights" (
	"flight_id" integer NOT NULL,
	"flight_no" bpchar NOT NULL,
	"scheduled_departure" timestamptz NOT NULL,
	"scheduled_arrival" timestamptz NOT NULL,
	"departure_airport" bpchar NOT NULL,
	"arrival_airport" bpchar NOT NULL,
	"status" varchar(20) NOT NULL,
	"aircraft_code" bpchar NOT NULL,
	"actual_departure" timestamptz,
	"actual_arrival" timestamptz,
	CONSTRAINT "flights_pkey" PRIMARY KEY("flight_id"),
	CONSTRAINT "flights_flight_no_scheduled_departure_key" UNIQUE("flight_no", "scheduled_departure"),
	CONSTRAINT "flights_check" CHECK(scheduled_arrival > scheduled_departure),
	CONSTRAINT "flights_check1" CHECK(actual_arrival IS NULL OR (actual_departure IS NOT NULL AND actual_arrival IS NOT NULL AND actual_arrival > actual_departure)),
	CONSTRAINT "flights_status_check" CHECK(status::text = ANY(ARRAY['On Time'::varchar::text, 'Delayed'::varchar::text, 'Departed'::varchar::text, 'Arrived'::varchar::text, 'Scheduled'::varchar::text, 'Cancelled'::varchar::text]))
);

CREATE TABLE "bookings"."seats" (
	"aircraft_code" bpchar NOT NULL,
	"seat_no" varchar(4) NOT NULL,
	"fare_conditions" varchar(10) NOT NULL,
	CONSTRAINT "seats_pkey" PRIMARY KEY("aircraft_code", "seat_no"),
	CONSTRAINT "seats_fare_conditions_check" CHECK(fare_conditions::text = ANY(ARRAY['Economy'::varchar::text, 'Comfort'::varchar::text, 'Business'::varchar::text]))
);

CREATE TABLE "bookings"."ticket_flights" (
	"ticket_no" bpchar NOT NULL,
	"flight_id" integer NOT NULL,
	"fare_conditions" varchar(10) NOT NULL,
	"amount" numeric(10,2) NOT NULL,
	CONSTRAINT "ticket_flights_pkey" PRIMARY KEY("ticket_no", "flight_id"),
	CONSTRAINT "ticket_flights_amount_check" CHECK(amount >= 0::numeric),
	CONSTRAINT "ticket_flights_fare_conditions_check" CHECK(fare_conditions::text = ANY(ARRAY['Economy'::varchar::text, 'Comfort'::varchar::text, 'Business'::varchar::text]))
);

CREATE TABLE "bookings"."tickets" (
	"ticket_no" bpchar NOT NULL,
	"book_ref" bpchar NOT NULL,
	"passenger_id" varchar(20) NOT NULL,
	"passenger_name" text NOT NULL,
	"contact_data" jsonb,
	CONSTRAINT "tickets_pkey" PRIMARY KEY("ticket_no")
);

ALTER TABLE "bookings"."boarding_passes" ADD CONSTRAINT "boarding_passes_ticket_no_fkey" FOREIGN KEY ("ticket_no", "flight_id")
	REFERENCES "bookings"."ticket_flights"("ticket_no", "flight_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."flights" ADD CONSTRAINT "flights_aircraft_code_fkey" FOREIGN KEY ("aircraft_code")
	REFERENCES "bookings"."aircrafts_data"("aircraft_code")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."flights" ADD CONSTRAINT "flights_arrival_airport_fkey" FOREIGN KEY ("arrival_airport")
	REFERENCES "bookings"."airports_data"("airport_code")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."flights" ADD CONSTRAINT "flights_departure_airport_fkey" FOREIGN KEY ("departure_airport")
	REFERENCES "bookings"."airports_data"("airport_code")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."seats" ADD CONSTRAINT "seats_aircraft_code_fkey" FOREIGN KEY ("aircraft_code")
	REFERENCES "bookings"."aircrafts_data"("aircraft_code")
	ON DELETE CASCADE
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."ticket_flights" ADD CONSTRAINT "ticket_flights_flight_id_fkey" FOREIGN KEY ("flight_id")
	REFERENCES "bookings"."flights"("flight_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."ticket_flights" ADD CONSTRAINT "ticket_flights_ticket_no_fkey" FOREIGN KEY ("ticket_no")
	REFERENCES "bookings"."tickets"("ticket_no")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "bookings"."tickets" ADD CONSTRAINT "tickets_book_ref_fkey" FOREIGN KEY ("book_ref")
	REFERENCES "bookings"."bookings"("book_ref")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

CREATE OR REPLACE FUNCTION "bookings"."lang"()
RETURNS text
LANGUAGE plpgsql
STABLE
AS $FUNCTION$
BEGIN
  RETURN current_setting('bookings.lang');
EXCEPTION
  WHEN undefined_object THEN
    RETURN NULL;
END;
$FUNCTION$;

CREATE OR REPLACE FUNCTION "bookings"."now"()
RETURNS timestamptz
LANGUAGE sql
IMMUTABLE
AS $FUNCTION$
SELECT '2017-08-15 18:00:00'::TIMESTAMP AT TIME ZONE 'Europe/Moscow';
$FUNCTION$;

CREATE VIEW "bookings"."aircrafts" AS
SELECT aircraft_code, model ->> bookings.lang() AS model, range FROM bookings.aircrafts_data ml;

CREATE VIEW "bookings"."airports" AS
SELECT airport_code, airport_name ->> bookings.lang() AS airport_name, city ->> bookings.lang() AS city, coordinates, timezone FROM bookings.airports_data ml;

CREATE VIEW "bookings"."flights_v" AS
SELECT f.flight_id, f.flight_no, f.scheduled_departure, timezone(dep.timezone, f.scheduled_departure) AS scheduled_departure_local, f.scheduled_arrival, timezone(arr.timezone, f.scheduled_arrival) AS scheduled_arrival_local, f.scheduled_arrival - f.scheduled_departure AS scheduled_duration, f.departure_airport, dep.airport_name AS departure_airport_name, dep.city AS departure_city, f.arrival_airport, arr.airport_name AS arrival_airport_name, arr.city AS arrival_city, f.status, f.aircraft_code, f.actual_departure, timezone(dep.timezone, f.actual_departure) AS actual_departure_local, f.actual_arrival, timezone(arr.timezone, f.actual_arrival) AS actual_arrival_local, f.actual_arrival - f.actual_departure AS actual_duration FROM bookings.flights f, bookings.airports dep, bookings.airports arr WHERE f.departure_airport = dep.airport_code AND f.arrival_airport = arr.airport_code;

CREATE VIEW "bookings"."routes" AS
WITH f3 AS (SELECT f2.flight_no, f2.departure_airport, f2.arrival_airport, f2.aircraft_code, f2.duration, array_agg(f2.days_of_week) AS days_of_week FROM (SELECT f1.flight_no, f1.departure_airport, f1.arrival_airport, f1.aircraft_code, f1.duration, f1.days_of_week FROM (SELECT flights.flight_no, flights.departure_airport, flights.arrival_airport, flights.aircraft_code, flights.scheduled_arrival - flights.scheduled_departure AS duration, to_char(flights.scheduled_departure, 'ID'::text)::int AS days_of_week FROM bookings.flights) f1 GROUP BY f1.flight_no, f1.departure_airport, f1.arrival_airport, f1.aircraft_code, f1.duration, f1.days_of_week ORDER BY f1.flight_no, f1.departure_airport, f1.arrival_airport, f1.aircraft_code, f1.duration, f1.days_of_week) f2 GROUP BY f2.flight_no, f2.departure_airport, f2.arrival_airport, f2.aircraft_code, f2.duration) SELECT f3.flight_no, f3.departure_airport, dep.airport_name AS departure_airport_name, dep.city AS departure_city, f3.arrival_airport, arr.airport_name AS arrival_airport_name, arr.city AS arrival_city, f3.aircraft_code, f3.duration, f3.days_of_week FROM f3, bookings.airports dep, bookings.airports arr WHERE f3.departure_airport = dep.airport_code AND f3.arrival_airport = arr.airport_code;

COMMENT ON SCHEMA "bookings" IS 'Airlines demo database schema';
COMMENT ON FUNCTION "bookings"."now" IS 'Point in time according to which the data are generated';
COMMENT ON TABLE "bookings"."aircrafts_data" IS 'Aircrafts (internal data)';
COMMENT ON COLUMN "bookings"."aircrafts_data"."aircraft_code" IS 'Aircraft code, IATA';
COMMENT ON COLUMN "bookings"."aircrafts_data"."model" IS 'Aircraft model';
COMMENT ON COLUMN "bookings"."aircrafts_data"."range" IS 'Maximal flying distance, km';
COMMENT ON VIEW "bookings"."aircrafts" IS 'Aircrafts';
COMMENT ON COLUMN "bookings"."aircrafts"."aircraft_code" IS 'Aircraft code, IATA';
COMMENT ON COLUMN "bookings"."aircrafts"."model" IS 'Aircraft model';
COMMENT ON COLUMN "bookings"."aircrafts"."range" IS 'Maximal flying distance, km';
COMMENT ON TABLE "bookings"."airports_data" IS 'Airports (internal data)';
COMMENT ON COLUMN "bookings"."airports_data"."airport_code" IS 'Airport code';
COMMENT ON COLUMN "bookings"."airports_data"."airport_name" IS 'Airport name';
COMMENT ON COLUMN "bookings"."airports_data"."city" IS 'City';
COMMENT ON COLUMN "bookings"."airports_data"."coordinates" IS 'Airport coordinates (longitude and latitude)';
COMMENT ON COLUMN "bookings"."airports_data"."timezone" IS 'Airport time zone';
COMMENT ON VIEW "bookings"."airports" IS 'Airports';
COMMENT ON COLUMN "bookings"."airports"."airport_code" IS 'Airport code';
COMMENT ON COLUMN "bookings"."airports"."airport_name" IS 'Airport name';
COMMENT ON COLUMN "bookings"."airports"."city" IS 'City';
COMMENT ON COLUMN "bookings"."airports"."coordinates" IS 'Airport coordinates (longitude and latitude)';
COMMENT ON COLUMN "bookings"."airports"."timezone" IS 'Airport time zone';
COMMENT ON TABLE "bookings"."boarding_passes" IS 'Boarding passes';
COMMENT ON COLUMN "bookings"."boarding_passes"."ticket_no" IS 'Ticket number';
COMMENT ON COLUMN "bookings"."boarding_passes"."flight_id" IS 'Flight ID';
COMMENT ON COLUMN "bookings"."boarding_passes"."boarding_no" IS 'Boarding pass number';
COMMENT ON COLUMN "bookings"."boarding_passes"."seat_no" IS 'Seat number';
COMMENT ON TABLE "bookings"."bookings" IS 'Bookings';
COMMENT ON COLUMN "bookings"."bookings"."book_ref" IS 'Booking number';
COMMENT ON COLUMN "bookings"."bookings"."book_date" IS 'Booking date';
COMMENT ON COLUMN "bookings"."bookings"."total_amount" IS 'Total booking cost';
COMMENT ON TABLE "bookings"."flights" IS 'Flights';
COMMENT ON COLUMN "bookings"."flights"."flight_id" IS 'Flight ID';
COMMENT ON COLUMN "bookings"."flights"."flight_no" IS 'Flight number';
COMMENT ON COLUMN "bookings"."flights"."scheduled_departure" IS 'Scheduled departure time';
COMMENT ON COLUMN "bookings"."flights"."scheduled_arrival" IS 'Scheduled arrival time';
COMMENT ON COLUMN "bookings"."flights"."departure_airport" IS 'Airport of departure';
COMMENT ON COLUMN "bookings"."flights"."arrival_airport" IS 'Airport of arrival';
COMMENT ON COLUMN "bookings"."flights"."status" IS 'Flight status';
COMMENT ON COLUMN "bookings"."flights"."aircraft_code" IS 'Aircraft code, IATA';
COMMENT ON COLUMN "bookings"."flights"."actual_departure" IS 'Actual departure time';
COMMENT ON COLUMN "bookings"."flights"."actual_arrival" IS 'Actual arrival time';
COMMENT ON VIEW "bookings"."flights_v" IS 'Flights (extended)';
COMMENT ON COLUMN "bookings"."flights_v"."flight_id" IS 'Flight ID';
COMMENT ON COLUMN "bookings"."flights_v"."flight_no" IS 'Flight number';
COMMENT ON COLUMN "bookings"."flights_v"."scheduled_departure" IS 'Scheduled departure time';
COMMENT ON COLUMN "bookings"."flights_v"."scheduled_departure_local" IS 'Scheduled departure time, local time at the point of departure';
COMMENT ON COLUMN "bookings"."flights_v"."scheduled_arrival" IS 'Scheduled arrival time';
COMMENT ON COLUMN "bookings"."flights_v"."scheduled_arrival_local" IS 'Scheduled arrival time, local time at the point of destination';
COMMENT ON COLUMN "bookings"."flights_v"."scheduled_duration" IS 'Scheduled flight duration';
COMMENT ON COLUMN "bookings"."flights_v"."departure_airport" IS 'Deprature airport code';
COMMENT ON COLUMN "bookings"."flights_v"."departure_airport_name" IS 'Departure airport name';
COMMENT ON COLUMN "bookings"."flights_v"."departure_city" IS 'City of departure';
COMMENT ON COLUMN "bookings"."flights_v"."arrival_airport" IS 'Arrival airport code';
COMMENT ON COLUMN "bookings"."flights_v"."arrival_airport_name" IS 'Arrival airport name';
COMMENT ON COLUMN "bookings"."flights_v"."arrival_city" IS 'City of arrival';
COMMENT ON COLUMN "bookings"."flights_v"."status" IS 'Flight status';
COMMENT ON COLUMN "bookings"."flights_v"."aircraft_code" IS 'Aircraft code, IATA';
COMMENT ON COLUMN "bookings"."flights_v"."actual_departure" IS 'Actual departure time';
COMMENT ON COLUMN "bookings"."flights_v"."actual_departure_local" IS 'Actual departure time, local time at the point of departure';
COMMENT ON COLUMN "bookings"."flights_v"."actual_arrival" IS 'Actual arrival time';
COMMENT ON COLUMN "bookings"."flights_v"."actual_arrival_local" IS 'Actual arrival time, local time at the point of destination';
COMMENT ON COLUMN "bookings"."flights_v"."actual_duration" IS 'Actual flight duration';
COMMENT ON VIEW "bookings"."routes" IS 'Routes';
COMMENT ON COLUMN "bookings"."routes"."flight_no" IS 'Flight number';
COMMENT ON COLUMN "bookings"."routes"."departure_airport" IS 'Code of airport of departure';
COMMENT ON COLUMN "bookings"."routes"."departure_airport_name" IS 'Name of airport of departure';
COMMENT ON COLUMN "bookings"."routes"."departure_city" IS 'City of departure';
COMMENT ON COLUMN "bookings"."routes"."arrival_airport" IS 'Code of airport of arrival';
COMMENT ON COLUMN "bookings"."routes"."arrival_airport_name" IS 'Name of airport of arrival';
COMMENT ON COLUMN "bookings"."routes"."arrival_city" IS 'City of arrival';
COMMENT ON COLUMN "bookings"."routes"."aircraft_code" IS 'Aircraft code, IATA';
COMMENT ON COLUMN "bookings"."routes"."duration" IS 'Scheduled duration of flight';
COMMENT ON COLUMN "bookings"."routes"."days_of_week" IS 'Days of week on which flights are scheduled';
COMMENT ON TABLE "bookings"."seats" IS 'Seats';
COMMENT ON COLUMN "bookings"."seats"."aircraft_code" IS 'Aircraft code, IATA';
COMMENT ON COLUMN "bookings"."seats"."seat_no" IS 'Seat number';
COMMENT ON COLUMN "bookings"."seats"."fare_conditions" IS 'Travel class';
COMMENT ON TABLE "bookings"."ticket_flights" IS 'Flight segment';
COMMENT ON COLUMN "bookings"."ticket_flights"."ticket_no" IS 'Ticket number';
COMMENT ON COLUMN "bookings"."ticket_flights"."flight_id" IS 'Flight ID';
COMMENT ON COLUMN "bookings"."ticket_flights"."fare_conditions" IS 'Travel class';
COMMENT ON COLUMN "bookings"."ticket_flights"."amount" IS 'Travel cost';
COMMENT ON TABLE "bookings"."tickets" IS 'Tickets';
COMMENT ON COLUMN "bookings"."tickets"."ticket_no" IS 'Ticket number';
COMMENT ON COLUMN "bookings"."tickets"."book_ref" IS 'Booking number';
COMMENT ON COLUMN "bookings"."tickets"."passenger_id" IS 'Passenger ID';
COMMENT ON COLUMN "bookings"."tickets"."passenger_name" IS 'Passenger name';
COMMENT ON COLUMN "bookings"."tickets"."contact_data" IS 'Passenger contact information';
