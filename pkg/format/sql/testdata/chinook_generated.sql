CREATE TABLE "album" (
	"album_id" integer NOT NULL,
	"title" varchar(160) NOT NULL,
	"artist_id" integer NOT NULL,
	CONSTRAINT "album_pkey" PRIMARY KEY("album_id")
);

CREATE TABLE "artist" (
	"artist_id" integer NOT NULL,
	"name" varchar(120),
	CONSTRAINT "artist_pkey" PRIMARY KEY("artist_id")
);

CREATE TABLE "customer" (
	"customer_id" integer NOT NULL,
	"first_name" varchar(40) NOT NULL,
	"last_name" varchar(20) NOT NULL,
	"company" varchar(80),
	"address" varchar(70),
	"city" varchar(40),
	"state" varchar(40),
	"country" varchar(40),
	"postal_code" varchar(10),
	"phone" varchar(24),
	"fax" varchar(24),
	"email" varchar(60) NOT NULL,
	"support_rep_id" integer,
	CONSTRAINT "customer_pkey" PRIMARY KEY("customer_id")
);

CREATE TABLE "employee" (
	"employee_id" integer NOT NULL,
	"last_name" varchar(20) NOT NULL,
	"first_name" varchar(20) NOT NULL,
	"title" varchar(30),
	"reports_to" integer,
	"birth_date" timestamp,
	"hire_date" timestamp,
	"address" varchar(70),
	"city" varchar(40),
	"state" varchar(40),
	"country" varchar(40),
	"postal_code" varchar(10),
	"phone" varchar(24),
	"fax" varchar(24),
	"email" varchar(60),
	CONSTRAINT "employee_pkey" PRIMARY KEY("employee_id")
);

CREATE TABLE "genre" (
	"genre_id" integer NOT NULL,
	"name" varchar(120),
	CONSTRAINT "genre_pkey" PRIMARY KEY("genre_id")
);

CREATE TABLE "invoice" (
	"invoice_id" integer NOT NULL,
	"customer_id" integer NOT NULL,
	"invoice_date" timestamp NOT NULL,
	"billing_address" varchar(70),
	"billing_city" varchar(40),
	"billing_state" varchar(40),
	"billing_country" varchar(40),
	"billing_postal_code" varchar(10),
	"total" numeric(10,2) NOT NULL,
	CONSTRAINT "invoice_pkey" PRIMARY KEY("invoice_id")
);

CREATE TABLE "invoice_line" (
	"invoice_line_id" integer NOT NULL,
	"invoice_id" integer NOT NULL,
	"track_id" integer NOT NULL,
	"unit_price" numeric(10,2) NOT NULL,
	"quantity" integer NOT NULL,
	CONSTRAINT "invoice_line_pkey" PRIMARY KEY("invoice_line_id")
);

CREATE TABLE "media_type" (
	"media_type_id" integer NOT NULL,
	"name" varchar(120),
	CONSTRAINT "media_type_pkey" PRIMARY KEY("media_type_id")
);

CREATE TABLE "playlist" (
	"playlist_id" integer NOT NULL,
	"name" varchar(120),
	CONSTRAINT "playlist_pkey" PRIMARY KEY("playlist_id")
);

CREATE TABLE "playlist_track" (
	"playlist_id" integer NOT NULL,
	"track_id" integer NOT NULL,
	CONSTRAINT "playlist_track_pkey" PRIMARY KEY("playlist_id", "track_id")
);

CREATE TABLE "track" (
	"track_id" integer NOT NULL,
	"name" varchar(200) NOT NULL,
	"album_id" integer,
	"media_type_id" integer NOT NULL,
	"genre_id" integer,
	"composer" varchar(220),
	"milliseconds" integer NOT NULL,
	"bytes" integer,
	"unit_price" numeric(10,2) NOT NULL,
	CONSTRAINT "track_pkey" PRIMARY KEY("track_id")
);

CREATE INDEX "album_artist_id_idx" ON "album" (
	"artist_id"
);

CREATE INDEX "customer_support_rep_id_idx" ON "customer" (
	"support_rep_id"
);

CREATE INDEX "employee_reports_to_idx" ON "employee" (
	"reports_to"
);

CREATE INDEX "invoice_customer_id_idx" ON "invoice" (
	"customer_id"
);

CREATE INDEX "invoice_line_invoice_id_idx" ON "invoice_line" (
	"invoice_id"
);

CREATE INDEX "invoice_line_track_id_idx" ON "invoice_line" (
	"track_id"
);

CREATE INDEX "playlist_track_playlist_id_idx" ON "playlist_track" (
	"playlist_id"
);

CREATE INDEX "playlist_track_track_id_idx" ON "playlist_track" (
	"track_id"
);

CREATE INDEX "track_album_id_idx" ON "track" (
	"album_id"
);

CREATE INDEX "track_genre_id_idx" ON "track" (
	"genre_id"
);

CREATE INDEX "track_media_type_id_idx" ON "track" (
	"media_type_id"
);

ALTER TABLE "album" ADD CONSTRAINT "album_artist_id_fkey" FOREIGN KEY ("artist_id")
	REFERENCES "artist"("artist_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "customer" ADD CONSTRAINT "customer_support_rep_id_fkey" FOREIGN KEY ("support_rep_id")
	REFERENCES "employee"("employee_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "employee" ADD CONSTRAINT "employee_reports_to_fkey" FOREIGN KEY ("reports_to")
	REFERENCES "employee"("employee_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "invoice" ADD CONSTRAINT "invoice_customer_id_fkey" FOREIGN KEY ("customer_id")
	REFERENCES "customer"("customer_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "invoice_line" ADD CONSTRAINT "invoice_line_invoice_id_fkey" FOREIGN KEY ("invoice_id")
	REFERENCES "invoice"("invoice_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "invoice_line" ADD CONSTRAINT "invoice_line_track_id_fkey" FOREIGN KEY ("track_id")
	REFERENCES "track"("track_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "playlist_track" ADD CONSTRAINT "playlist_track_playlist_id_fkey" FOREIGN KEY ("playlist_id")
	REFERENCES "playlist"("playlist_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "playlist_track" ADD CONSTRAINT "playlist_track_track_id_fkey" FOREIGN KEY ("track_id")
	REFERENCES "track"("track_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "track" ADD CONSTRAINT "track_album_id_fkey" FOREIGN KEY ("album_id")
	REFERENCES "album"("album_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "track" ADD CONSTRAINT "track_genre_id_fkey" FOREIGN KEY ("genre_id")
	REFERENCES "genre"("genre_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "track" ADD CONSTRAINT "track_media_type_id_fkey" FOREIGN KEY ("media_type_id")
	REFERENCES "media_type"("media_type_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

