--
-- PostgreSQL database dump
--

\restrict eRQ2AiZcHeWlcXUxNgYGl1bzExHfFyL5z59PflW1sNB0VtxtXROUrhjO6TCwijn

-- Dumped from database version 17.7 (Homebrew)
-- Dumped by pg_dump version 17.7 (Homebrew)

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: public; Type: SCHEMA; Schema: -; Owner: -
--

-- *not* creating schema, since initdb creates it


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: album; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.album (
    album_id integer NOT NULL,
    title character varying(160) NOT NULL,
    artist_id integer NOT NULL
);


--
-- Name: artist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.artist (
    artist_id integer NOT NULL,
    name character varying(120)
);


--
-- Name: customer; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.customer (
    customer_id integer NOT NULL,
    first_name character varying(40) NOT NULL,
    last_name character varying(20) NOT NULL,
    company character varying(80),
    address character varying(70),
    city character varying(40),
    state character varying(40),
    country character varying(40),
    postal_code character varying(10),
    phone character varying(24),
    fax character varying(24),
    email character varying(60) NOT NULL,
    support_rep_id integer
);


--
-- Name: employee; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.employee (
    employee_id integer NOT NULL,
    last_name character varying(20) NOT NULL,
    first_name character varying(20) NOT NULL,
    title character varying(30),
    reports_to integer,
    birth_date timestamp without time zone,
    hire_date timestamp without time zone,
    address character varying(70),
    city character varying(40),
    state character varying(40),
    country character varying(40),
    postal_code character varying(10),
    phone character varying(24),
    fax character varying(24),
    email character varying(60)
);


--
-- Name: genre; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.genre (
    genre_id integer NOT NULL,
    name character varying(120)
);


--
-- Name: invoice; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.invoice (
    invoice_id integer NOT NULL,
    customer_id integer NOT NULL,
    invoice_date timestamp without time zone NOT NULL,
    billing_address character varying(70),
    billing_city character varying(40),
    billing_state character varying(40),
    billing_country character varying(40),
    billing_postal_code character varying(10),
    total numeric(10,2) NOT NULL
);


--
-- Name: invoice_line; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.invoice_line (
    invoice_line_id integer NOT NULL,
    invoice_id integer NOT NULL,
    track_id integer NOT NULL,
    unit_price numeric(10,2) NOT NULL,
    quantity integer NOT NULL
);


--
-- Name: media_type; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.media_type (
    media_type_id integer NOT NULL,
    name character varying(120)
);


--
-- Name: playlist; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.playlist (
    playlist_id integer NOT NULL,
    name character varying(120)
);


--
-- Name: playlist_track; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.playlist_track (
    playlist_id integer NOT NULL,
    track_id integer NOT NULL
);


--
-- Name: track; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.track (
    track_id integer NOT NULL,
    name character varying(200) NOT NULL,
    album_id integer,
    media_type_id integer NOT NULL,
    genre_id integer,
    composer character varying(220),
    milliseconds integer NOT NULL,
    bytes integer,
    unit_price numeric(10,2) NOT NULL
);


--
-- Name: album album_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.album
    ADD CONSTRAINT album_pkey PRIMARY KEY (album_id);


--
-- Name: artist artist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.artist
    ADD CONSTRAINT artist_pkey PRIMARY KEY (artist_id);


--
-- Name: customer customer_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.customer
    ADD CONSTRAINT customer_pkey PRIMARY KEY (customer_id);


--
-- Name: employee employee_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.employee
    ADD CONSTRAINT employee_pkey PRIMARY KEY (employee_id);


--
-- Name: genre genre_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.genre
    ADD CONSTRAINT genre_pkey PRIMARY KEY (genre_id);


--
-- Name: invoice_line invoice_line_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invoice_line
    ADD CONSTRAINT invoice_line_pkey PRIMARY KEY (invoice_line_id);


--
-- Name: invoice invoice_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invoice
    ADD CONSTRAINT invoice_pkey PRIMARY KEY (invoice_id);


--
-- Name: media_type media_type_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.media_type
    ADD CONSTRAINT media_type_pkey PRIMARY KEY (media_type_id);


--
-- Name: playlist playlist_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.playlist
    ADD CONSTRAINT playlist_pkey PRIMARY KEY (playlist_id);


--
-- Name: playlist_track playlist_track_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.playlist_track
    ADD CONSTRAINT playlist_track_pkey PRIMARY KEY (playlist_id, track_id);


--
-- Name: track track_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.track
    ADD CONSTRAINT track_pkey PRIMARY KEY (track_id);


--
-- Name: album_artist_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX album_artist_id_idx ON public.album USING btree (artist_id);


--
-- Name: customer_support_rep_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX customer_support_rep_id_idx ON public.customer USING btree (support_rep_id);


--
-- Name: employee_reports_to_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX employee_reports_to_idx ON public.employee USING btree (reports_to);


--
-- Name: invoice_customer_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX invoice_customer_id_idx ON public.invoice USING btree (customer_id);


--
-- Name: invoice_line_invoice_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX invoice_line_invoice_id_idx ON public.invoice_line USING btree (invoice_id);


--
-- Name: invoice_line_track_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX invoice_line_track_id_idx ON public.invoice_line USING btree (track_id);


--
-- Name: playlist_track_playlist_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX playlist_track_playlist_id_idx ON public.playlist_track USING btree (playlist_id);


--
-- Name: playlist_track_track_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX playlist_track_track_id_idx ON public.playlist_track USING btree (track_id);


--
-- Name: track_album_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX track_album_id_idx ON public.track USING btree (album_id);


--
-- Name: track_genre_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX track_genre_id_idx ON public.track USING btree (genre_id);


--
-- Name: track_media_type_id_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX track_media_type_id_idx ON public.track USING btree (media_type_id);


--
-- Name: album album_artist_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.album
    ADD CONSTRAINT album_artist_id_fkey FOREIGN KEY (artist_id) REFERENCES public.artist(artist_id);


--
-- Name: customer customer_support_rep_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.customer
    ADD CONSTRAINT customer_support_rep_id_fkey FOREIGN KEY (support_rep_id) REFERENCES public.employee(employee_id);


--
-- Name: employee employee_reports_to_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.employee
    ADD CONSTRAINT employee_reports_to_fkey FOREIGN KEY (reports_to) REFERENCES public.employee(employee_id);


--
-- Name: invoice invoice_customer_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invoice
    ADD CONSTRAINT invoice_customer_id_fkey FOREIGN KEY (customer_id) REFERENCES public.customer(customer_id);


--
-- Name: invoice_line invoice_line_invoice_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invoice_line
    ADD CONSTRAINT invoice_line_invoice_id_fkey FOREIGN KEY (invoice_id) REFERENCES public.invoice(invoice_id);


--
-- Name: invoice_line invoice_line_track_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.invoice_line
    ADD CONSTRAINT invoice_line_track_id_fkey FOREIGN KEY (track_id) REFERENCES public.track(track_id);


--
-- Name: playlist_track playlist_track_playlist_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.playlist_track
    ADD CONSTRAINT playlist_track_playlist_id_fkey FOREIGN KEY (playlist_id) REFERENCES public.playlist(playlist_id);


--
-- Name: playlist_track playlist_track_track_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.playlist_track
    ADD CONSTRAINT playlist_track_track_id_fkey FOREIGN KEY (track_id) REFERENCES public.track(track_id);


--
-- Name: track track_album_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.track
    ADD CONSTRAINT track_album_id_fkey FOREIGN KEY (album_id) REFERENCES public.album(album_id);


--
-- Name: track track_genre_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.track
    ADD CONSTRAINT track_genre_id_fkey FOREIGN KEY (genre_id) REFERENCES public.genre(genre_id);


--
-- Name: track track_media_type_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.track
    ADD CONSTRAINT track_media_type_id_fkey FOREIGN KEY (media_type_id) REFERENCES public.media_type(media_type_id);


--
-- PostgreSQL database dump complete
--

\unrestrict eRQ2AiZcHeWlcXUxNgYGl1bzExHfFyL5z59PflW1sNB0VtxtXROUrhjO6TCwijn

