--
-- PostgreSQL database dump
--

\restrict QfDoZvGqKOH64T0p10yhntUs6V3xql0fH2YM7e7M98Q4LjOkbsWrmOYkGFjgOVB

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


--
-- Name: btree_gist; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS btree_gist WITH SCHEMA public;


--
-- Name: EXTENSION btree_gist; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION btree_gist IS 'support for indexing common datatypes in GiST';


--
-- Name: document_status; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.document_status AS ENUM (
    'draft',
    'review',
    'published',
    'archived'
);


--
-- Name: email_address; Type: DOMAIN; Schema: public; Owner: -
--

CREATE DOMAIN public.email_address AS text
	CONSTRAINT email_address_check CHECK ((VALUE ~ '^[^@]+@[^@]+\.[^@]+$'::text));


--
-- Name: positive_int; Type: DOMAIN; Schema: public; Owner: -
--

CREATE DOMAIN public.positive_int AS integer
	CONSTRAINT positive_int_check CHECK ((VALUE > 0));


--
-- Name: slug; Type: DOMAIN; Schema: public; Owner: -
--

CREATE DOMAIN public.slug AS character varying NOT NULL
	CONSTRAINT slug_format CHECK (((VALUE)::text ~ '^[a-z0-9]([a-z0-9-]*[a-z0-9])?$'::text));


--
-- Name: user_role; Type: TYPE; Schema: public; Owner: -
--

CREATE TYPE public.user_role AS ENUM (
    'viewer',
    'editor',
    'admin',
    'owner'
);


--
-- Name: audit_trigger_func(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.audit_trigger_func() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        INSERT INTO audit_log (user_id, table_name, record_id, action, new_data)
        VALUES (current_setting('app.current_user_id', true)::bigint, TG_TABLE_NAME, NEW.id, TG_OP, to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO audit_log (user_id, table_name, record_id, action, old_data, new_data)
        VALUES (current_setting('app.current_user_id', true)::bigint, TG_TABLE_NAME, NEW.id, TG_OP, to_jsonb(OLD), to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'DELETE' THEN
        INSERT INTO audit_log (user_id, table_name, record_id, action, old_data)
        VALUES (current_setting('app.current_user_id', true)::bigint, TG_TABLE_NAME, OLD.id, TG_OP, to_jsonb(OLD));
        RETURN OLD;
    END IF;
    RETURN NULL;
END;
$$;


--
-- Name: update_document_search_vector(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_document_search_vector() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', coalesce(NEW.title, '') || ' ' || coalesce(NEW.body, ''));
    RETURN NEW;
END;
$$;


--
-- Name: update_user_search_vector(); Type: FUNCTION; Schema: public; Owner: -
--

CREATE FUNCTION public.update_user_search_vector() RETURNS trigger
    LANGUAGE plpgsql
    AS $$
BEGIN
    NEW.search_vector := to_tsvector('english', coalesce(NEW.display_name, '') || ' ' || coalesce(NEW.email::text, ''));
    RETURN NEW;
END;
$$;


SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- Name: audit_log; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.audit_log (
    id bigint NOT NULL,
    user_id bigint,
    table_name character varying(64) NOT NULL,
    record_id bigint NOT NULL,
    action character varying(10) NOT NULL,
    old_data jsonb,
    new_data jsonb,
    ip_address inet,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_audit_action CHECK (((action)::text = ANY ((ARRAY['INSERT'::character varying, 'UPDATE'::character varying, 'DELETE'::character varying])::text[])))
);


--
-- Name: TABLE audit_log; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.audit_log IS 'Immutable audit trail for all data changes';


--
-- Name: COLUMN audit_log.old_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.audit_log.old_data IS 'Previous row state as JSON (NULL for INSERT)';


--
-- Name: COLUMN audit_log.new_data; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.audit_log.new_data IS 'New row state as JSON (NULL for DELETE)';


--
-- Name: audit_log_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.audit_log ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.audit_log_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: booking_slots; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.booking_slots (
    id integer NOT NULL,
    room_name character varying(100) NOT NULL,
    booked_during tstzrange NOT NULL,
    booked_by bigint NOT NULL
);


--
-- Name: TABLE booking_slots; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.booking_slots IS 'Room booking with non-overlapping time range constraint';


--
-- Name: COLUMN booking_slots.booked_during; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.booking_slots.booked_during IS 'Time range for the booking (tstzrange with EXCLUDE constraint)';


--
-- Name: booking_slots_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.booking_slots ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.booking_slots_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: categories; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.categories (
    id integer NOT NULL,
    parent_id integer,
    name character varying(100) NOT NULL,
    slug public.slug,
    description text,
    sort_order public.positive_int DEFAULT 1,
    created_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE categories; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.categories IS 'Hierarchical document categories (self-referencing tree)';


--
-- Name: categories_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.categories ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.categories_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: document_permissions; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.document_permissions (
    id bigint NOT NULL,
    document_id bigint NOT NULL,
    user_id bigint NOT NULL,
    role public.user_role DEFAULT 'viewer'::public.user_role NOT NULL,
    granted_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone,
    CONSTRAINT chk_docperm_expires CHECK (((expires_at IS NULL) OR (expires_at > granted_at)))
);


--
-- Name: TABLE document_permissions; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.document_permissions IS 'Per-document user permissions with optional expiration';


--
-- Name: document_permissions_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.document_permissions ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.document_permissions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: document_tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.document_tags (
    document_id bigint NOT NULL,
    tag_id integer NOT NULL
);


--
-- Name: TABLE document_tags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.document_tags IS 'Many-to-many relationship between documents and tags';


--
-- Name: documents; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.documents (
    id bigint NOT NULL,
    category_id integer,
    author_id bigint NOT NULL,
    title character varying(500) NOT NULL,
    body text,
    metadata jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    status public.document_status DEFAULT 'draft'::public.document_status NOT NULL,
    version public.positive_int DEFAULT 1 NOT NULL,
    search_vector tsvector,
    published_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL,
    CONSTRAINT chk_documents_publish CHECK (((status <> 'published'::public.document_status) OR (published_at IS NOT NULL)))
);


--
-- Name: TABLE documents; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.documents IS 'Core document storage with full-text search and versioning';


--
-- Name: COLUMN documents.metadata; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.documents.metadata IS 'Arbitrary document metadata as JSON (custom fields, etc.)';


--
-- Name: COLUMN documents.version; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.documents.version IS 'Monotonically increasing version number';


--
-- Name: COLUMN documents.search_vector; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.documents.search_vector IS 'Full-text search index over title and body';


--
-- Name: documents_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.documents ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.documents_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: tags; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.tags (
    id integer NOT NULL,
    name character varying(60) NOT NULL,
    color bpchar DEFAULT '#cccccc'::bpchar,
    CONSTRAINT chk_tags_color CHECK ((color ~ '^#[0-9a-fA-F]{6}$'::text))
);


--
-- Name: TABLE tags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.tags IS 'Flat tag taxonomy for document classification';


--
-- Name: tags_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.tags ALTER COLUMN id ADD GENERATED BY DEFAULT AS IDENTITY (
    SEQUENCE NAME public.tags_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id bigint NOT NULL,
    email public.email_address NOT NULL,
    display_name character varying(200) NOT NULL,
    role public.user_role DEFAULT 'viewer'::public.user_role NOT NULL,
    settings jsonb DEFAULT '{}'::jsonb NOT NULL,
    tags text[] DEFAULT '{}'::text[] NOT NULL,
    search_vector tsvector,
    is_active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    updated_at timestamp with time zone DEFAULT now() NOT NULL
);


--
-- Name: TABLE users; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON TABLE public.users IS 'System users with role-based access control';


--
-- Name: COLUMN users.settings; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.settings IS 'User preferences as JSON (theme, notifications, etc.)';


--
-- Name: COLUMN users.tags; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.tags IS 'Free-form text tags for user categorization';


--
-- Name: COLUMN users.search_vector; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON COLUMN public.users.search_vector IS 'Full-text search index over display_name and email';


--
-- Name: users_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

ALTER TABLE public.users ALTER COLUMN id ADD GENERATED ALWAYS AS IDENTITY (
    SEQUENCE NAME public.users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1
);


--
-- Name: audit_log audit_log_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_pkey PRIMARY KEY (id);


--
-- Name: booking_slots booking_slots_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.booking_slots
    ADD CONSTRAINT booking_slots_pkey PRIMARY KEY (id);


--
-- Name: categories categories_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_pkey PRIMARY KEY (id);


--
-- Name: document_permissions document_permissions_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_permissions
    ADD CONSTRAINT document_permissions_pkey PRIMARY KEY (id);


--
-- Name: document_tags document_tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_tags
    ADD CONSTRAINT document_tags_pkey PRIMARY KEY (document_id, tag_id);


--
-- Name: documents documents_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_pkey PRIMARY KEY (id);


--
-- Name: tags tags_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT tags_pkey PRIMARY KEY (id);


--
-- Name: document_permissions uq_docperm_document_user; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_permissions
    ADD CONSTRAINT uq_docperm_document_user UNIQUE (document_id, user_id);


--
-- Name: tags uq_tags_name; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.tags
    ADD CONSTRAINT uq_tags_name UNIQUE (name);


--
-- Name: users uq_users_email; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT uq_users_email UNIQUE (email);


--
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);


--
-- Name: idx_audit_created; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_created ON public.audit_log USING btree (created_at DESC);


--
-- Name: idx_audit_new_data_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_new_data_gin ON public.audit_log USING gin (new_data);


--
-- Name: idx_audit_old_data_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_old_data_gin ON public.audit_log USING gin (old_data);


--
-- Name: idx_audit_recent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_recent ON public.audit_log USING btree (created_at DESC) WHERE (created_at > '2024-01-01 00:00:00+03'::timestamp with time zone);


--
-- Name: idx_audit_table_record; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_audit_table_record ON public.audit_log USING btree (table_name, record_id);


--
-- Name: idx_categories_parent; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_categories_parent ON public.categories USING btree (parent_id);


--
-- Name: idx_categories_slug_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_categories_slug_lower ON public.categories USING btree (lower((slug)::text));


--
-- Name: idx_documents_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_active ON public.documents USING btree (created_at DESC) WHERE (status = 'published'::public.document_status);


--
-- Name: INDEX idx_documents_active; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON INDEX public.idx_documents_active IS 'Partial index: only published documents';


--
-- Name: idx_documents_author; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_author ON public.documents USING btree (author_id);


--
-- Name: idx_documents_category; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_category ON public.documents USING btree (category_id);


--
-- Name: idx_documents_draft; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_draft ON public.documents USING btree (author_id, updated_at DESC) WHERE (status = 'draft'::public.document_status);


--
-- Name: idx_documents_metadata_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_metadata_gin ON public.documents USING gin (metadata);


--
-- Name: INDEX idx_documents_metadata_gin; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON INDEX public.idx_documents_metadata_gin IS 'GIN index for JSONB containment queries on document metadata';


--
-- Name: idx_documents_search_gist; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_search_gist ON public.documents USING gist (search_vector);


--
-- Name: INDEX idx_documents_search_gist; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON INDEX public.idx_documents_search_gist IS 'GiST index for full-text search on documents';


--
-- Name: idx_documents_status; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_status ON public.documents USING btree (status);


--
-- Name: idx_documents_tags_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_tags_gin ON public.documents USING gin (tags);


--
-- Name: idx_documents_title_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_documents_title_lower ON public.documents USING btree (lower((title)::text));


--
-- Name: idx_users_active; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_active ON public.users USING btree (email) WHERE (is_active = true);


--
-- Name: idx_users_email_lower; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_email_lower ON public.users USING btree (lower((email)::text));


--
-- Name: idx_users_search_gist; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_search_gist ON public.users USING gist (search_vector);


--
-- Name: idx_users_settings_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_settings_gin ON public.users USING gin (settings);


--
-- Name: INDEX idx_users_settings_gin; Type: COMMENT; Schema: public; Owner: -
--

COMMENT ON INDEX public.idx_users_settings_gin IS 'GIN index for JSONB containment queries on user settings';


--
-- Name: idx_users_tags_gin; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_users_tags_gin ON public.users USING gin (tags);


--
-- Name: documents trg_documents_audit; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_documents_audit AFTER INSERT OR DELETE OR UPDATE ON public.documents FOR EACH ROW EXECUTE FUNCTION public.audit_trigger_func();


--
-- Name: documents trg_documents_search_vector; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_documents_search_vector BEFORE INSERT OR UPDATE ON public.documents FOR EACH ROW EXECUTE FUNCTION public.update_document_search_vector();


--
-- Name: users trg_users_audit; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_users_audit AFTER INSERT OR DELETE OR UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.audit_trigger_func();


--
-- Name: users trg_users_search_vector; Type: TRIGGER; Schema: public; Owner: -
--

CREATE TRIGGER trg_users_search_vector BEFORE INSERT OR UPDATE ON public.users FOR EACH ROW EXECUTE FUNCTION public.update_user_search_vector();


--
-- Name: audit_log audit_log_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.audit_log
    ADD CONSTRAINT audit_log_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE SET NULL;


--
-- Name: booking_slots booking_slots_booked_by_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.booking_slots
    ADD CONSTRAINT booking_slots_booked_by_fkey FOREIGN KEY (booked_by) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: categories categories_parent_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.categories
    ADD CONSTRAINT categories_parent_id_fkey FOREIGN KEY (parent_id) REFERENCES public.categories(id) ON DELETE SET NULL;


--
-- Name: document_permissions document_permissions_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_permissions
    ADD CONSTRAINT document_permissions_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE CASCADE;


--
-- Name: document_permissions document_permissions_user_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_permissions
    ADD CONSTRAINT document_permissions_user_id_fkey FOREIGN KEY (user_id) REFERENCES public.users(id) ON DELETE CASCADE;


--
-- Name: document_tags document_tags_document_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_tags
    ADD CONSTRAINT document_tags_document_id_fkey FOREIGN KEY (document_id) REFERENCES public.documents(id) ON DELETE CASCADE;


--
-- Name: document_tags document_tags_tag_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.document_tags
    ADD CONSTRAINT document_tags_tag_id_fkey FOREIGN KEY (tag_id) REFERENCES public.tags(id) ON DELETE CASCADE;


--
-- Name: documents documents_author_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_author_id_fkey FOREIGN KEY (author_id) REFERENCES public.users(id) ON DELETE RESTRICT;


--
-- Name: documents documents_category_id_fkey; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.documents
    ADD CONSTRAINT documents_category_id_fkey FOREIGN KEY (category_id) REFERENCES public.categories(id) ON DELETE SET NULL;


--
-- PostgreSQL database dump complete
--

\unrestrict QfDoZvGqKOH64T0p10yhntUs6V3xql0fH2YM7e7M98Q4LjOkbsWrmOYkGFjgOVB

