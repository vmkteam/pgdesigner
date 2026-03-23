CREATE TYPE "mpaa_rating" AS ENUM ('G', 'PG', 'PG-13', 'R', 'NC-17');

CREATE DOMAIN "bıgınt" AS bigint;

CREATE DOMAIN "year" AS integer CONSTRAINT "year_check" CHECK (value >= 1901 AND value <= 2155);

CREATE SEQUENCE "customer_customer_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "actor_actor_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "category_category_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "film_film_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "address_address_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "city_city_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "country_country_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "inventory_inventory_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "language_language_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "payment_payment_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "rental_rental_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "staff_staff_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "store_store_id_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE TABLE "actor" (
	"actor_id" integer NOT NULL DEFAULT nextval('public.actor_actor_id_seq'::regclass),
	"first_name" text NOT NULL,
	"last_name" text NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "actor_pkey" PRIMARY KEY("actor_id")
);

CREATE TABLE "address" (
	"address_id" integer NOT NULL DEFAULT nextval('public.address_address_id_seq'::regclass),
	"address" text NOT NULL,
	"address2" text,
	"district" text NOT NULL,
	"city_id" integer NOT NULL,
	"postal_code" text,
	"phone" text NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "address_pkey" PRIMARY KEY("address_id")
);

CREATE TABLE "category" (
	"category_id" integer NOT NULL DEFAULT nextval('public.category_category_id_seq'::regclass),
	"name" text NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "category_pkey" PRIMARY KEY("category_id")
);

CREATE TABLE "city" (
	"city_id" integer NOT NULL DEFAULT nextval('public.city_city_id_seq'::regclass),
	"city" text NOT NULL,
	"country_id" integer NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "city_pkey" PRIMARY KEY("city_id")
);

CREATE TABLE "country" (
	"country_id" integer NOT NULL DEFAULT nextval('public.country_country_id_seq'::regclass),
	"country" text NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "country_pkey" PRIMARY KEY("country_id")
);

CREATE TABLE "customer" (
	"customer_id" integer NOT NULL DEFAULT nextval('public.customer_customer_id_seq'::regclass),
	"store_id" integer NOT NULL,
	"first_name" text NOT NULL,
	"last_name" text NOT NULL,
	"email" text,
	"address_id" integer NOT NULL,
	"activebool" boolean NOT NULL DEFAULT true,
	"create_date" date NOT NULL DEFAULT current_date,
	"last_update" timestamptz DEFAULT now(),
	"active" integer,
	CONSTRAINT "customer_pkey" PRIMARY KEY("customer_id")
);

CREATE TABLE "film" (
	"film_id" integer NOT NULL DEFAULT nextval('public.film_film_id_seq'::regclass),
	"title" text NOT NULL,
	"description" text,
	"release_year" year,
	"language_id" integer NOT NULL,
	"original_language_id" integer,
	"rental_duration" smallint NOT NULL DEFAULT 3,
	"rental_rate" numeric(4,2) NOT NULL DEFAULT 4.99,
	"length" smallint,
	"replacement_cost" numeric(5,2) NOT NULL DEFAULT 19.99,
	"rating" mpaa_rating DEFAULT 'G'::public.mpaa_rating,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	"special_features" text[],
	"fulltext" tsvector NOT NULL,
	CONSTRAINT "film_pkey" PRIMARY KEY("film_id")
);

CREATE TABLE "film_actor" (
	"actor_id" integer NOT NULL,
	"film_id" integer NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "film_actor_pkey" PRIMARY KEY("actor_id", "film_id")
);

CREATE TABLE "film_category" (
	"film_id" integer NOT NULL,
	"category_id" integer NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "film_category_pkey" PRIMARY KEY("film_id", "category_id")
);

CREATE TABLE "inventory" (
	"inventory_id" integer NOT NULL DEFAULT nextval('public.inventory_inventory_id_seq'::regclass),
	"film_id" integer NOT NULL,
	"store_id" integer NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "inventory_pkey" PRIMARY KEY("inventory_id")
);

CREATE TABLE "language" (
	"language_id" integer NOT NULL DEFAULT nextval('public.language_language_id_seq'::regclass),
	"name" bpchar NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "language_pkey" PRIMARY KEY("language_id")
);

CREATE TABLE "payment" (
	"payment_id" integer NOT NULL DEFAULT nextval('public.payment_payment_id_seq'::regclass),
	"customer_id" integer NOT NULL,
	"staff_id" integer NOT NULL,
	"rental_id" integer NOT NULL,
	"amount" numeric(5,2) NOT NULL,
	"payment_date" timestamptz NOT NULL,
	PRIMARY KEY("payment_date", "payment_id")
)
PARTITION BY RANGE ("payment_date");

CREATE TABLE "payment_p2022_01" PARTITION OF "payment"
    FOR VALUES FROM ('2022-01-01 00:00:00+00') TO ('2022-02-01 00:00:00+00');

CREATE TABLE "payment_p2022_02" PARTITION OF "payment"
    FOR VALUES FROM ('2022-02-01 00:00:00+00') TO ('2022-03-01 00:00:00+00');

CREATE TABLE "payment_p2022_03" PARTITION OF "payment"
    FOR VALUES FROM ('2022-03-01 00:00:00+00') TO ('2022-04-01 01:00:00+01');

CREATE TABLE "payment_p2022_04" PARTITION OF "payment"
    FOR VALUES FROM ('2022-04-01 01:00:00+01') TO ('2022-05-01 01:00:00+01');

CREATE TABLE "payment_p2022_05" PARTITION OF "payment"
    FOR VALUES FROM ('2022-05-01 01:00:00+01') TO ('2022-06-01 01:00:00+01');

CREATE TABLE "payment_p2022_06" PARTITION OF "payment"
    FOR VALUES FROM ('2022-06-01 01:00:00+01') TO ('2022-07-01 01:00:00+01');

CREATE TABLE "payment_p2022_07" PARTITION OF "payment"
    FOR VALUES FROM ('2022-07-01 01:00:00+01') TO ('2022-08-01 01:00:00+01');

CREATE TABLE "rental" (
	"rental_id" integer NOT NULL DEFAULT nextval('public.rental_rental_id_seq'::regclass),
	"rental_date" timestamptz NOT NULL,
	"inventory_id" integer NOT NULL,
	"customer_id" integer NOT NULL,
	"return_date" timestamptz,
	"staff_id" integer NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "rental_pkey" PRIMARY KEY("rental_id")
);

CREATE TABLE "staff" (
	"staff_id" integer NOT NULL DEFAULT nextval('public.staff_staff_id_seq'::regclass),
	"first_name" text NOT NULL,
	"last_name" text NOT NULL,
	"address_id" integer NOT NULL,
	"email" text,
	"store_id" integer NOT NULL,
	"active" boolean NOT NULL DEFAULT true,
	"username" text NOT NULL,
	"password" text,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	"picture" bytea,
	CONSTRAINT "staff_pkey" PRIMARY KEY("staff_id")
);

CREATE TABLE "store" (
	"store_id" integer NOT NULL DEFAULT nextval('public.store_store_id_seq'::regclass),
	"manager_staff_id" integer NOT NULL,
	"address_id" integer NOT NULL,
	"last_update" timestamptz NOT NULL DEFAULT now(),
	CONSTRAINT "store_pkey" PRIMARY KEY("store_id")
);

CREATE INDEX "film_fulltext_idx" ON "film" USING GIST (
	"fulltext"
);

CREATE INDEX "idx_actor_last_name" ON "actor" (
	"last_name"
);

CREATE INDEX "idx_fk_address_id" ON "customer" (
	"address_id"
);

CREATE INDEX "idx_fk_city_id" ON "address" (
	"city_id"
);

CREATE INDEX "idx_fk_country_id" ON "city" (
	"country_id"
);

CREATE INDEX "idx_fk_film_id" ON "film_actor" (
	"film_id"
);

CREATE INDEX "idx_fk_inventory_id" ON "rental" (
	"inventory_id"
);

CREATE INDEX "idx_fk_language_id" ON "film" (
	"language_id"
);

CREATE INDEX "idx_fk_original_language_id" ON "film" (
	"original_language_id"
);

CREATE INDEX "idx_fk_store_id" ON "customer" (
	"store_id"
);

CREATE INDEX "idx_last_name" ON "customer" (
	"last_name"
);

CREATE INDEX "idx_store_id_film_id" ON "inventory" (
	"store_id",
	"film_id"
);

CREATE INDEX "idx_title" ON "film" (
	"title"
);

CREATE UNIQUE INDEX "idx_unq_manager_staff_id" ON "store" (
	"manager_staff_id"
);

CREATE UNIQUE INDEX "idx_unq_rental_rental_date_inventory_id_customer_id" ON "rental" (
	"rental_date",
	"inventory_id",
	"customer_id"
);

ALTER TABLE "address" ADD CONSTRAINT "address_city_id_fkey" FOREIGN KEY ("city_id")
	REFERENCES "city"("city_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "city" ADD CONSTRAINT "city_country_id_fkey" FOREIGN KEY ("country_id")
	REFERENCES "country"("country_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "customer" ADD CONSTRAINT "customer_address_id_fkey" FOREIGN KEY ("address_id")
	REFERENCES "address"("address_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "customer" ADD CONSTRAINT "customer_store_id_fkey" FOREIGN KEY ("store_id")
	REFERENCES "store"("store_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "film" ADD CONSTRAINT "film_language_id_fkey" FOREIGN KEY ("language_id")
	REFERENCES "language"("language_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "film" ADD CONSTRAINT "film_original_language_id_fkey" FOREIGN KEY ("original_language_id")
	REFERENCES "language"("language_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "film_actor" ADD CONSTRAINT "film_actor_actor_id_fkey" FOREIGN KEY ("actor_id")
	REFERENCES "actor"("actor_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "film_actor" ADD CONSTRAINT "film_actor_film_id_fkey" FOREIGN KEY ("film_id")
	REFERENCES "film"("film_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "film_category" ADD CONSTRAINT "film_category_category_id_fkey" FOREIGN KEY ("category_id")
	REFERENCES "category"("category_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "film_category" ADD CONSTRAINT "film_category_film_id_fkey" FOREIGN KEY ("film_id")
	REFERENCES "film"("film_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "inventory" ADD CONSTRAINT "inventory_film_id_fkey" FOREIGN KEY ("film_id")
	REFERENCES "film"("film_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "inventory" ADD CONSTRAINT "inventory_store_id_fkey" FOREIGN KEY ("store_id")
	REFERENCES "store"("store_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "rental" ADD CONSTRAINT "rental_customer_id_fkey" FOREIGN KEY ("customer_id")
	REFERENCES "customer"("customer_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "rental" ADD CONSTRAINT "rental_inventory_id_fkey" FOREIGN KEY ("inventory_id")
	REFERENCES "inventory"("inventory_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "rental" ADD CONSTRAINT "rental_staff_id_fkey" FOREIGN KEY ("staff_id")
	REFERENCES "staff"("staff_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "staff" ADD CONSTRAINT "staff_address_id_fkey" FOREIGN KEY ("address_id")
	REFERENCES "address"("address_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

ALTER TABLE "staff" ADD CONSTRAINT "staff_store_id_fkey" FOREIGN KEY ("store_id")
	REFERENCES "store"("store_id")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "store" ADD CONSTRAINT "store_address_id_fkey" FOREIGN KEY ("address_id")
	REFERENCES "address"("address_id")
	ON DELETE RESTRICT
	ON UPDATE CASCADE
	NOT DEFERRABLE;

CREATE OR REPLACE FUNCTION "_group_concat"(text, text)
RETURNS text
LANGUAGE sql
IMMUTABLE
AS $FUNCTION$
SELECT CASE
  WHEN $2 IS NULL THEN $1
  WHEN $1 IS NULL THEN $2
  ELSE $1 || ', ' || $2
END
$FUNCTION$;

CREATE OR REPLACE FUNCTION "get_customer_balance"("p_customer_id" integer, "p_effective_date" timestamptz)
RETURNS numeric
LANGUAGE plpgsql
AS $FUNCTION$
DECLARE
    v_rentfees DECIMAL(5,2); --#FEES PAID TO RENT THE VIDEOS INITIALLY
    v_overfees INTEGER;      --#LATE FEES FOR PRIOR RENTALS
    v_payments DECIMAL(5,2); --#SUM OF PAYMENTS MADE PREVIOUSLY
BEGIN
    SELECT COALESCE(SUM(film.rental_rate),0) INTO v_rentfees
    FROM film, inventory, rental
    WHERE film.film_id = inventory.film_id
      AND inventory.inventory_id = rental.inventory_id
      AND rental.rental_date <= p_effective_date
      AND rental.customer_id = p_customer_id;
    SELECT COALESCE(SUM(IF((rental.return_date - rental.rental_date) > (film.rental_duration * '1 day'::interval),
        ((rental.return_date - rental.rental_date) - (film.rental_duration * '1 day'::interval)),0)),0) INTO v_overfees
    FROM rental, inventory, film
    WHERE film.film_id = inventory.film_id
      AND inventory.inventory_id = rental.inventory_id
      AND rental.rental_date <= p_effective_date
      AND rental.customer_id = p_customer_id;
    SELECT COALESCE(SUM(payment.amount),0) INTO v_payments
    FROM payment
    WHERE payment.payment_date <= p_effective_date
    AND payment.customer_id = p_customer_id;
    RETURN v_rentfees + v_overfees - v_payments;
END
$FUNCTION$;

CREATE OR REPLACE FUNCTION "inventory_held_by_customer"("p_inventory_id" integer)
RETURNS integer
LANGUAGE plpgsql
AS $FUNCTION$
DECLARE
    v_customer_id INTEGER;
BEGIN
  SELECT customer_id INTO v_customer_id
  FROM rental
  WHERE return_date IS NULL
  AND inventory_id = p_inventory_id;
  RETURN v_customer_id;
END
$FUNCTION$;

CREATE OR REPLACE FUNCTION "inventory_in_stock"("p_inventory_id" integer)
RETURNS boolean
LANGUAGE plpgsql
AS $FUNCTION$
DECLARE
    v_rentals INTEGER;
    v_out     INTEGER;
BEGIN
    SELECT count(*) INTO v_rentals
    FROM rental
    WHERE inventory_id = p_inventory_id;
    IF v_rentals = 0 THEN
      RETURN TRUE;
    END IF;
    SELECT COUNT(rental_id) INTO v_out
    FROM inventory LEFT JOIN rental USING(inventory_id)
    WHERE inventory.inventory_id = p_inventory_id
    AND rental.return_date IS NULL;
    IF v_out > 0 THEN
      RETURN FALSE;
    ELSE
      RETURN TRUE;
    END IF;
END
$FUNCTION$;

CREATE OR REPLACE FUNCTION "last_day"(timestamptz)
RETURNS date
LANGUAGE sql
IMMUTABLE
STRICT
AS $FUNCTION$
SELECT CASE
    WHEN EXTRACT(MONTH FROM $1) = 12 THEN
      (((EXTRACT(YEAR FROM $1) + 1) operator(pg_catalog.||) '-01-01')::date - INTERVAL '1 day')::date
    ELSE
      ((EXTRACT(YEAR FROM $1) operator(pg_catalog.||) '-' operator(pg_catalog.||) (EXTRACT(MONTH FROM $1) + 1) operator(pg_catalog.||) '-01')::date - INTERVAL '1 day')::date
    END
$FUNCTION$;

CREATE OR REPLACE FUNCTION "last_updated"()
RETURNS trigger
LANGUAGE plpgsql
AS $FUNCTION$
BEGIN
    NEW.last_update = CURRENT_TIMESTAMP;
    RETURN NEW;
END
$FUNCTION$;

CREATE OR REPLACE FUNCTION "rewards_report"("min_monthly_purchases" integer, "min_dollar_amount_purchased" numeric)
RETURNS SETOF customer
LANGUAGE plpgsql
SECURITY DEFINER
AS $FUNCTION$
DECLARE
    last_month_start DATE;
    last_month_end DATE;
rr RECORD;
tmpSQL TEXT;
BEGIN
    /* Some sanity checks... */
    IF min_monthly_purchases = 0 THEN
        RAISE EXCEPTION 'Minimum monthly purchases parameter must be > 0';
    END IF;
    IF min_dollar_amount_purchased = 0.00 THEN
        RAISE EXCEPTION 'Minimum monthly dollar amount purchased parameter must be > $0.00';
    END IF;
    last_month_start := CURRENT_DATE - '3 month'::interval;
    last_month_start := to_date((extract(YEAR FROM last_month_start) || '-' || extract(MONTH FROM last_month_start) || '-01'),'YYYY-MM-DD');
    last_month_end := LAST_DAY(last_month_start);
    /*
    Create a temporary storage area for Customer IDs.
    */
    CREATE TEMPORARY TABLE tmpCustomer (customer_id INTEGER NOT NULL PRIMARY KEY);
    /*
    Find all customers meeting the monthly purchase requirements
    */
    tmpSQL := 'INSERT INTO tmpCustomer (customer_id)
        SELECT p.customer_id
        FROM payment AS p
        WHERE DATE(p.payment_date) BETWEEN '||quote_literal(last_month_start) ||' AND '|| quote_literal(last_month_end) || '
        GROUP BY customer_id
        HAVING SUM(p.amount) > '|| min_dollar_amount_purchased || '
        AND COUNT(customer_id) > ' ||min_monthly_purchases ;
    EXECUTE tmpSQL;
    /*
    Output ALL customer information of matching rewardees.
    Customize output as needed.
    */
    FOR rr IN EXECUTE 'SELECT c.* FROM tmpCustomer AS t INNER JOIN customer AS c ON t.customer_id = c.customer_id' LOOP
        RETURN NEXT rr;
    END LOOP;
    /* Clean up */
    tmpSQL := 'DROP TABLE tmpCustomer';
    EXECUTE tmpSQL;
RETURN;
END
$FUNCTION$;

CREATE AGGREGATE "group_concat"(text) (
    SFUNC = _group_concat,
    STYPE = text
);

CREATE OR REPLACE FUNCTION "film_in_stock"("p_film_id" integer, "p_store_id" integer, OUT "p_film_count" integer)
RETURNS SETOF integer
LANGUAGE sql
AS $FUNCTION$
SELECT inventory_id
     FROM inventory
     WHERE film_id = $1
     AND store_id = $2
     AND inventory_in_stock(inventory_id);
$FUNCTION$;

CREATE OR REPLACE FUNCTION "film_not_in_stock"("p_film_id" integer, "p_store_id" integer, OUT "p_film_count" integer)
RETURNS SETOF integer
LANGUAGE sql
AS $FUNCTION$
SELECT inventory_id
    FROM inventory
    WHERE film_id = $1
    AND store_id = $2
    AND NOT inventory_in_stock(inventory_id);
$FUNCTION$;

CREATE MATERIALIZED VIEW "rental_by_category" AS
SELECT c.name AS category, sum(p.amount) AS total_sales FROM public.payment p JOIN public.rental r ON p.rental_id = r.rental_id JOIN public.inventory i ON r.inventory_id = i.inventory_id JOIN public.film f ON i.film_id = f.film_id JOIN public.film_category fc ON f.film_id = fc.film_id JOIN public.category c ON fc.category_id = c.category_id GROUP BY c.name ORDER BY sum(p.amount) DESC;

CREATE VIEW "actor_info" AS
SELECT a.actor_id, a.first_name, a.last_name, public.group_concat(DISTINCT (c.name || ': '::text) || (SELECT public.group_concat(f.title) AS group_concat FROM public.film f JOIN public.film_category fc_1 ON f.film_id = fc_1.film_id JOIN public.film_actor fa_1 ON f.film_id = fa_1.film_id WHERE fc_1.category_id = c.category_id AND fa_1.actor_id = a.actor_id GROUP BY fa_1.actor_id)) AS film_info FROM public.actor a LEFT JOIN public.film_actor fa ON a.actor_id = fa.actor_id LEFT JOIN public.film_category fc ON fa.film_id = fc.film_id LEFT JOIN public.category c ON fc.category_id = c.category_id GROUP BY a.actor_id, a.first_name, a.last_name;

CREATE VIEW "customer_list" AS
SELECT cu.customer_id AS id, (cu.first_name || ' '::text) || cu.last_name AS name, a.address, a.postal_code AS "zip code", a.phone, city.city, country.country, CASE WHEN cu.activebool THEN 'active'::text ELSE ''::text END AS notes, cu.store_id AS sid FROM public.customer cu JOIN public.address a ON cu.address_id = a.address_id JOIN public.city ON a.city_id = city.city_id JOIN public.country ON city.country_id = country.country_id;

CREATE VIEW "film_list" AS
SELECT film.film_id AS fid, film.title, film.description, category.name AS category, film.rental_rate AS price, film.length, film.rating, public.group_concat((actor.first_name || ' '::text) || actor.last_name) AS actors FROM public.category LEFT JOIN public.film_category ON category.category_id = film_category.category_id LEFT JOIN public.film ON film_category.film_id = film.film_id JOIN public.film_actor ON film.film_id = film_actor.film_id JOIN public.actor ON film_actor.actor_id = actor.actor_id GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;

CREATE VIEW "nicer_but_slower_film_list" AS
SELECT film.film_id AS fid, film.title, film.description, category.name AS category, film.rental_rate AS price, film.length, film.rating, public.group_concat(((upper("substring"(actor.first_name, 1, 1)) || lower("substring"(actor.first_name, 2))) || upper("substring"(actor.last_name, 1, 1))) || lower("substring"(actor.last_name, 2))) AS actors FROM public.category LEFT JOIN public.film_category ON category.category_id = film_category.category_id LEFT JOIN public.film ON film_category.film_id = film.film_id JOIN public.film_actor ON film.film_id = film_actor.film_id JOIN public.actor ON film_actor.actor_id = actor.actor_id GROUP BY film.film_id, film.title, film.description, category.name, film.rental_rate, film.length, film.rating;

CREATE VIEW "sales_by_film_category" AS
SELECT c.name AS category, sum(p.amount) AS total_sales FROM public.payment p JOIN public.rental r ON p.rental_id = r.rental_id JOIN public.inventory i ON r.inventory_id = i.inventory_id JOIN public.film f ON i.film_id = f.film_id JOIN public.film_category fc ON f.film_id = fc.film_id JOIN public.category c ON fc.category_id = c.category_id GROUP BY c.name ORDER BY sum(p.amount) DESC;

CREATE VIEW "sales_by_store" AS
SELECT (c.city || ','::text) || cy.country AS store, (m.first_name || ' '::text) || m.last_name AS manager, sum(p.amount) AS total_sales FROM public.payment p JOIN public.rental r ON p.rental_id = r.rental_id JOIN public.inventory i ON r.inventory_id = i.inventory_id JOIN public.store s ON i.store_id = s.store_id JOIN public.address a ON s.address_id = a.address_id JOIN public.city c ON a.city_id = c.city_id JOIN public.country cy ON c.country_id = cy.country_id JOIN public.staff m ON s.manager_staff_id = m.staff_id GROUP BY cy.country, c.city, s.store_id, m.first_name, m.last_name ORDER BY cy.country, c.city;

CREATE VIEW "staff_list" AS
SELECT s.staff_id AS id, (s.first_name || ' '::text) || s.last_name AS name, a.address, a.postal_code AS "zip code", a.phone, city.city, country.country, s.store_id AS sid FROM public.staff s JOIN public.address a ON s.address_id = a.address_id JOIN public.city ON a.city_id = city.city_id JOIN public.country ON city.country_id = country.country_id;

CREATE UNIQUE INDEX "rental_category" ON "rental_by_category" (
	"category"
);

CREATE TRIGGER "film_fulltext_trigger"
	BEFORE INSERT OR UPDATE
	ON "film"
	FOR EACH ROW
	EXECUTE FUNCTION "tsvector_update_trigger"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "actor"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "address"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "category"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "city"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "country"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "customer"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "film"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "film_actor"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "film_category"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "inventory"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "language"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "rental"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "staff"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

CREATE TRIGGER "last_updated"
	BEFORE UPDATE
	ON "store"
	FOR EACH ROW
	EXECUTE FUNCTION "last_updated"();

