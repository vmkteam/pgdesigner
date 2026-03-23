ALTER TABLE "products" DROP COLUMN "rating";

ALTER TABLE "reviews" ADD COLUMN "rating" numeric(3,1) NOT NULL DEFAULT 0;

