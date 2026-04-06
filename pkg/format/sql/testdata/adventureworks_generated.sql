CREATE EXTENSION IF NOT EXISTS "tablefunc";

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE DOMAIN "AccountNumber" AS varchar;

CREATE DOMAIN "Flag" AS boolean NOT NULL;

CREATE DOMAIN "Name" AS varchar;

CREATE DOMAIN "NameStyle" AS boolean NOT NULL;

CREATE DOMAIN "OrderNumber" AS varchar;

CREATE DOMAIN "Phone" AS varchar;

CREATE SCHEMA IF NOT EXISTS "hr";

CREATE SCHEMA IF NOT EXISTS "humanresources";

CREATE SCHEMA IF NOT EXISTS "pe";

CREATE SCHEMA IF NOT EXISTS "person";

CREATE SCHEMA IF NOT EXISTS "pr";

CREATE SCHEMA IF NOT EXISTS "production";

CREATE SCHEMA IF NOT EXISTS "pu";

CREATE SCHEMA IF NOT EXISTS "purchasing";

CREATE SCHEMA IF NOT EXISTS "sa";

CREATE SCHEMA IF NOT EXISTS "sales";

CREATE SEQUENCE "humanresources"."department_departmentid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "humanresources"."jobcandidate_jobcandidateid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "humanresources"."shift_shiftid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."address_addressid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."addresstype_addresstypeid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."businessentity_businessentityid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."contacttype_contacttypeid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."emailaddress_emailaddressid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."phonenumbertype_phonenumbertypeid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "person"."stateprovince_stateprovinceid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."billofmaterials_billofmaterialsid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."illustration_illustrationid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."location_locationid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."product_productid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."productcategory_productcategoryid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."productdescription_productdescriptionid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."productmodel_productmodelid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."productphoto_productphotoid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."productreview_productreviewid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."productsubcategory_productsubcategoryid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."scrapreason_scrapreasonid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."transactionhistory_transactionid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "production"."workorder_workorderid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "purchasing"."purchaseorderdetail_purchaseorderdetailid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "purchasing"."purchaseorderheader_purchaseorderid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "purchasing"."shipmethod_shipmethodid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."creditcard_creditcardid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."currencyrate_currencyrateid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."customer_customerid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."salesorderdetail_salesorderdetailid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."salesorderheader_salesorderid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."salesreason_salesreasonid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."salestaxrate_salestaxrateid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."salesterritory_territoryid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."shoppingcartitem_shoppingcartitemid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE SEQUENCE "sales"."specialoffer_specialofferid_seq" INCREMENT BY 1 START WITH 1 CACHE 1;

CREATE TABLE "humanresources"."department" (
	"departmentid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"groupname" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Department_DepartmentID" PRIMARY KEY("departmentid")
);

CREATE TABLE "humanresources"."employee" (
	"businessentityid" integer NOT NULL,
	"nationalidnumber" varchar(15) NOT NULL,
	"loginid" varchar(256) NOT NULL,
	"jobtitle" varchar(50) NOT NULL,
	"birthdate" date NOT NULL,
	"maritalstatus" char(1) NOT NULL,
	"gender" char(1) NOT NULL,
	"hiredate" date NOT NULL,
	"salariedflag" "Flag" NOT NULL DEFAULT true,
	"vacationhours" smallint NOT NULL DEFAULT 0,
	"sickleavehours" smallint NOT NULL DEFAULT 0,
	"currentflag" "Flag" NOT NULL DEFAULT true,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	"organizationnode" varchar DEFAULT '/',
	CONSTRAINT "PK_Employee_BusinessEntityID" PRIMARY KEY("businessentityid"),
	CONSTRAINT "CK_Employee_BirthDate" CHECK(birthdate >= '1930-01-01'::date AND birthdate <= (now() - '18 years'::interval)),
	CONSTRAINT "CK_Employee_Gender" CHECK(upper(gender::text) = ANY(ARRAY['M'::text, 'F'::text])),
	CONSTRAINT "CK_Employee_HireDate" CHECK(hiredate >= '1996-07-01'::date AND hiredate <= (now() + '1 day'::interval)),
	CONSTRAINT "CK_Employee_MaritalStatus" CHECK(upper(maritalstatus::text) = ANY(ARRAY['M'::text, 'S'::text])),
	CONSTRAINT "CK_Employee_SickLeaveHours" CHECK(sickleavehours >= 0 AND sickleavehours <= 120),
	CONSTRAINT "CK_Employee_VacationHours" CHECK(vacationhours >= '-40'::int AND vacationhours <= 240)
);

CREATE TABLE "humanresources"."employeedepartmenthistory" (
	"businessentityid" integer NOT NULL,
	"departmentid" smallint NOT NULL,
	"shiftid" smallint NOT NULL,
	"startdate" date NOT NULL,
	"enddate" date,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_EmployeeDepartmentHistory_BusinessEntityID_StartDate_Departm" PRIMARY KEY("businessentityid", "startdate", "departmentid", "shiftid"),
	CONSTRAINT "CK_EmployeeDepartmentHistory_EndDate" CHECK(enddate >= startdate OR enddate IS NULL)
);

CREATE TABLE "humanresources"."employeepayhistory" (
	"businessentityid" integer NOT NULL,
	"ratechangedate" timestamp NOT NULL,
	"rate" numeric NOT NULL,
	"payfrequency" smallint NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_EmployeePayHistory_BusinessEntityID_RateChangeDate" PRIMARY KEY("businessentityid", "ratechangedate"),
	CONSTRAINT "CK_EmployeePayHistory_PayFrequency" CHECK(payfrequency = ANY(ARRAY[1, 2])),
	CONSTRAINT "CK_EmployeePayHistory_Rate" CHECK(rate >= 6.50 AND rate <= 200.00)
);

CREATE TABLE "humanresources"."jobcandidate" (
	"jobcandidateid" integer NOT NULL,
	"businessentityid" integer,
	"resume" xml,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_JobCandidate_JobCandidateID" PRIMARY KEY("jobcandidateid")
);

CREATE TABLE "humanresources"."shift" (
	"shiftid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"starttime" time NOT NULL,
	"endtime" time NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Shift_ShiftID" PRIMARY KEY("shiftid")
);

CREATE TABLE "person"."address" (
	"addressid" integer NOT NULL,
	"addressline1" varchar(60) NOT NULL,
	"addressline2" varchar(60),
	"city" varchar(30) NOT NULL,
	"stateprovinceid" integer NOT NULL,
	"postalcode" varchar(15) NOT NULL,
	"spatiallocation" bytea,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Address_AddressID" PRIMARY KEY("addressid")
);

CREATE TABLE "person"."addresstype" (
	"addresstypeid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_AddressType_AddressTypeID" PRIMARY KEY("addresstypeid")
);

CREATE TABLE "person"."businessentity" (
	"businessentityid" integer NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_BusinessEntity_BusinessEntityID" PRIMARY KEY("businessentityid")
);

CREATE TABLE "person"."businessentityaddress" (
	"businessentityid" integer NOT NULL,
	"addressid" integer NOT NULL,
	"addresstypeid" integer NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_BusinessEntityAddress_BusinessEntityID_AddressID_AddressType" PRIMARY KEY("businessentityid", "addressid", "addresstypeid")
);

CREATE TABLE "person"."businessentitycontact" (
	"businessentityid" integer NOT NULL,
	"personid" integer NOT NULL,
	"contacttypeid" integer NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_BusinessEntityContact_BusinessEntityID_PersonID_ContactTypeI" PRIMARY KEY("businessentityid", "personid", "contacttypeid")
);

CREATE TABLE "person"."contacttype" (
	"contacttypeid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ContactType_ContactTypeID" PRIMARY KEY("contacttypeid")
);

CREATE TABLE "person"."countryregion" (
	"countryregioncode" varchar(3) NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_CountryRegion_CountryRegionCode" PRIMARY KEY("countryregioncode")
);

CREATE TABLE "person"."emailaddress" (
	"businessentityid" integer NOT NULL,
	"emailaddressid" integer NOT NULL,
	"emailaddress" varchar(50),
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_EmailAddress_BusinessEntityID_EmailAddressID" PRIMARY KEY("businessentityid", "emailaddressid")
);

CREATE TABLE "person"."password" (
	"businessentityid" integer NOT NULL,
	"passwordhash" varchar(128) NOT NULL,
	"passwordsalt" varchar(10) NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Password_BusinessEntityID" PRIMARY KEY("businessentityid")
);

CREATE TABLE "person"."person" (
	"businessentityid" integer NOT NULL,
	"persontype" char(2) NOT NULL,
	"namestyle" "NameStyle" NOT NULL DEFAULT false,
	"title" varchar(8),
	"firstname" "Name" NOT NULL,
	"middlename" "Name",
	"lastname" "Name" NOT NULL,
	"suffix" varchar(10),
	"emailpromotion" integer NOT NULL DEFAULT 0,
	"additionalcontactinfo" xml,
	"demographics" xml,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Person_BusinessEntityID" PRIMARY KEY("businessentityid"),
	CONSTRAINT "CK_Person_EmailPromotion" CHECK(emailpromotion >= 0 AND emailpromotion <= 2),
	CONSTRAINT "CK_Person_PersonType" CHECK(persontype IS NULL OR upper(persontype::text) = ANY(ARRAY['SC'::text, 'VC'::text, 'IN'::text, 'EM'::text, 'SP'::text, 'GC'::text]))
);

CREATE TABLE "person"."personphone" (
	"businessentityid" integer NOT NULL,
	"phonenumber" "Phone" NOT NULL,
	"phonenumbertypeid" integer NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_PersonPhone_BusinessEntityID_PhoneNumber_PhoneNumberTypeID" PRIMARY KEY("businessentityid", "phonenumber", "phonenumbertypeid")
);

CREATE TABLE "person"."phonenumbertype" (
	"phonenumbertypeid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_PhoneNumberType_PhoneNumberTypeID" PRIMARY KEY("phonenumbertypeid")
);

CREATE TABLE "person"."stateprovince" (
	"stateprovinceid" integer NOT NULL,
	"stateprovincecode" char(3) NOT NULL,
	"countryregioncode" varchar(3) NOT NULL,
	"isonlystateprovinceflag" "Flag" NOT NULL DEFAULT true,
	"name" "Name" NOT NULL,
	"territoryid" integer NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_StateProvince_StateProvinceID" PRIMARY KEY("stateprovinceid")
);

CREATE TABLE "production"."billofmaterials" (
	"billofmaterialsid" integer NOT NULL,
	"productassemblyid" integer,
	"componentid" integer NOT NULL,
	"startdate" timestamp NOT NULL DEFAULT now(),
	"enddate" timestamp,
	"unitmeasurecode" char(3) NOT NULL,
	"bomlevel" smallint NOT NULL,
	"perassemblyqty" numeric(8,2) NOT NULL DEFAULT 1.00,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_BillOfMaterials_BillOfMaterialsID" PRIMARY KEY("billofmaterialsid"),
	CONSTRAINT "CK_BillOfMaterials_BOMLevel" CHECK((productassemblyid IS NULL AND bomlevel = 0 AND perassemblyqty = 1.00) OR (productassemblyid IS NOT NULL AND bomlevel >= 1)),
	CONSTRAINT "CK_BillOfMaterials_EndDate" CHECK(enddate > startdate OR enddate IS NULL),
	CONSTRAINT "CK_BillOfMaterials_PerAssemblyQty" CHECK(perassemblyqty >= 1.00),
	CONSTRAINT "CK_BillOfMaterials_ProductAssemblyID" CHECK(productassemblyid <> componentid)
);

CREATE TABLE "production"."culture" (
	"cultureid" char(6) NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Culture_CultureID" PRIMARY KEY("cultureid")
);

CREATE TABLE "production"."document" (
	"title" varchar(50) NOT NULL,
	"owner" integer NOT NULL,
	"folderflag" "Flag" NOT NULL DEFAULT false,
	"filename" varchar(400) NOT NULL,
	"fileextension" varchar(8),
	"revision" char(5) NOT NULL,
	"changenumber" integer NOT NULL DEFAULT 0,
	"status" smallint NOT NULL,
	"documentsummary" text,
	"document" bytea,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	"documentnode" varchar NOT NULL DEFAULT '/',
	CONSTRAINT "PK_Document_DocumentNode" PRIMARY KEY("documentnode"),
	CONSTRAINT "document_rowguid_key" UNIQUE("rowguid"),
	CONSTRAINT "CK_Document_Status" CHECK(status >= 1 AND status <= 3)
);

CREATE TABLE "production"."illustration" (
	"illustrationid" integer NOT NULL,
	"diagram" xml,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Illustration_IllustrationID" PRIMARY KEY("illustrationid")
);

CREATE TABLE "production"."location" (
	"locationid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"costrate" numeric NOT NULL DEFAULT 0.00,
	"availability" numeric(8,2) NOT NULL DEFAULT 0.00,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Location_LocationID" PRIMARY KEY("locationid"),
	CONSTRAINT "CK_Location_Availability" CHECK(availability >= 0.00),
	CONSTRAINT "CK_Location_CostRate" CHECK(costrate >= 0.00)
);

CREATE TABLE "production"."product" (
	"productid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"productnumber" varchar(25) NOT NULL,
	"makeflag" "Flag" NOT NULL DEFAULT true,
	"finishedgoodsflag" "Flag" NOT NULL DEFAULT true,
	"color" varchar(15),
	"safetystocklevel" smallint NOT NULL,
	"reorderpoint" smallint NOT NULL,
	"standardcost" numeric NOT NULL,
	"listprice" numeric NOT NULL,
	"size" varchar(5),
	"sizeunitmeasurecode" char(3),
	"weightunitmeasurecode" char(3),
	"weight" numeric(8,2),
	"daystomanufacture" integer NOT NULL,
	"productline" char(2),
	"class" char(2),
	"style" char(2),
	"productsubcategoryid" integer,
	"productmodelid" integer,
	"sellstartdate" timestamp NOT NULL,
	"sellenddate" timestamp,
	"discontinueddate" timestamp,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Product_ProductID" PRIMARY KEY("productid"),
	CONSTRAINT "CK_Product_Class" CHECK(upper(class::text) = ANY(ARRAY['L'::text, 'M'::text, 'H'::text]) OR class IS NULL),
	CONSTRAINT "CK_Product_DaysToManufacture" CHECK(daystomanufacture >= 0),
	CONSTRAINT "CK_Product_ListPrice" CHECK(listprice >= 0.00),
	CONSTRAINT "CK_Product_ProductLine" CHECK(upper(productline::text) = ANY(ARRAY['S'::text, 'T'::text, 'M'::text, 'R'::text]) OR productline IS NULL),
	CONSTRAINT "CK_Product_ReorderPoint" CHECK(reorderpoint > 0),
	CONSTRAINT "CK_Product_SafetyStockLevel" CHECK(safetystocklevel > 0),
	CONSTRAINT "CK_Product_SellEndDate" CHECK(sellenddate >= sellstartdate OR sellenddate IS NULL),
	CONSTRAINT "CK_Product_StandardCost" CHECK(standardcost >= 0.00),
	CONSTRAINT "CK_Product_Style" CHECK(upper(style::text) = ANY(ARRAY['W'::text, 'M'::text, 'U'::text]) OR style IS NULL),
	CONSTRAINT "CK_Product_Weight" CHECK(weight > 0.00)
);

CREATE TABLE "production"."productcategory" (
	"productcategoryid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductCategory_ProductCategoryID" PRIMARY KEY("productcategoryid")
);

CREATE TABLE "production"."productcosthistory" (
	"productid" integer NOT NULL,
	"startdate" timestamp NOT NULL,
	"enddate" timestamp,
	"standardcost" numeric NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductCostHistory_ProductID_StartDate" PRIMARY KEY("productid", "startdate"),
	CONSTRAINT "CK_ProductCostHistory_EndDate" CHECK(enddate >= startdate OR enddate IS NULL),
	CONSTRAINT "CK_ProductCostHistory_StandardCost" CHECK(standardcost >= 0.00)
);

CREATE TABLE "production"."productdescription" (
	"productdescriptionid" integer NOT NULL,
	"description" varchar(400) NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductDescription_ProductDescriptionID" PRIMARY KEY("productdescriptionid")
);

CREATE TABLE "production"."productdocument" (
	"productid" integer NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	"documentnode" varchar NOT NULL DEFAULT '/',
	CONSTRAINT "PK_ProductDocument_ProductID_DocumentNode" PRIMARY KEY("productid", "documentnode")
);

CREATE TABLE "production"."productinventory" (
	"productid" integer NOT NULL,
	"locationid" smallint NOT NULL,
	"shelf" varchar(10) NOT NULL,
	"bin" smallint NOT NULL,
	"quantity" smallint NOT NULL DEFAULT 0,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductInventory_ProductID_LocationID" PRIMARY KEY("productid", "locationid"),
	CONSTRAINT "CK_ProductInventory_Bin" CHECK(bin >= 0 AND bin <= 100)
);

CREATE TABLE "production"."productlistpricehistory" (
	"productid" integer NOT NULL,
	"startdate" timestamp NOT NULL,
	"enddate" timestamp,
	"listprice" numeric NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductListPriceHistory_ProductID_StartDate" PRIMARY KEY("productid", "startdate"),
	CONSTRAINT "CK_ProductListPriceHistory_EndDate" CHECK(enddate >= startdate OR enddate IS NULL),
	CONSTRAINT "CK_ProductListPriceHistory_ListPrice" CHECK(listprice > 0.00)
);

CREATE TABLE "production"."productmodel" (
	"productmodelid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"catalogdescription" xml,
	"instructions" xml,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductModel_ProductModelID" PRIMARY KEY("productmodelid")
);

CREATE TABLE "production"."productmodelillustration" (
	"productmodelid" integer NOT NULL,
	"illustrationid" integer NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductModelIllustration_ProductModelID_IllustrationID" PRIMARY KEY("productmodelid", "illustrationid")
);

CREATE TABLE "production"."productmodelproductdescriptionculture" (
	"productmodelid" integer NOT NULL,
	"productdescriptionid" integer NOT NULL,
	"cultureid" char(6) NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductModelProductDescriptionCulture_ProductModelID_Product" PRIMARY KEY("productmodelid", "productdescriptionid", "cultureid")
);

CREATE TABLE "production"."productphoto" (
	"productphotoid" integer NOT NULL,
	"thumbnailphoto" bytea,
	"thumbnailphotofilename" varchar(50),
	"largephoto" bytea,
	"largephotofilename" varchar(50),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductPhoto_ProductPhotoID" PRIMARY KEY("productphotoid")
);

CREATE TABLE "production"."productproductphoto" (
	"productid" integer NOT NULL,
	"productphotoid" integer NOT NULL,
	"primary" "Flag" NOT NULL DEFAULT false,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductProductPhoto_ProductID_ProductPhotoID" PRIMARY KEY("productid", "productphotoid")
);

CREATE TABLE "production"."productreview" (
	"productreviewid" integer NOT NULL,
	"productid" integer NOT NULL,
	"reviewername" "Name" NOT NULL,
	"reviewdate" timestamp NOT NULL DEFAULT now(),
	"emailaddress" varchar(50) NOT NULL,
	"rating" integer NOT NULL,
	"comments" varchar(3850),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductReview_ProductReviewID" PRIMARY KEY("productreviewid"),
	CONSTRAINT "CK_ProductReview_Rating" CHECK(rating >= 1 AND rating <= 5)
);

CREATE TABLE "production"."productsubcategory" (
	"productsubcategoryid" integer NOT NULL,
	"productcategoryid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductSubcategory_ProductSubcategoryID" PRIMARY KEY("productsubcategoryid")
);

CREATE TABLE "production"."scrapreason" (
	"scrapreasonid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ScrapReason_ScrapReasonID" PRIMARY KEY("scrapreasonid")
);

CREATE TABLE "production"."transactionhistory" (
	"transactionid" integer NOT NULL,
	"productid" integer NOT NULL,
	"referenceorderid" integer NOT NULL,
	"referenceorderlineid" integer NOT NULL DEFAULT 0,
	"transactiondate" timestamp NOT NULL DEFAULT now(),
	"transactiontype" char(1) NOT NULL,
	"quantity" integer NOT NULL,
	"actualcost" numeric NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_TransactionHistory_TransactionID" PRIMARY KEY("transactionid"),
	CONSTRAINT "CK_TransactionHistory_TransactionType" CHECK(upper(transactiontype::text) = ANY(ARRAY['W'::text, 'S'::text, 'P'::text]))
);

CREATE TABLE "production"."transactionhistoryarchive" (
	"transactionid" integer NOT NULL,
	"productid" integer NOT NULL,
	"referenceorderid" integer NOT NULL,
	"referenceorderlineid" integer NOT NULL DEFAULT 0,
	"transactiondate" timestamp NOT NULL DEFAULT now(),
	"transactiontype" char(1) NOT NULL,
	"quantity" integer NOT NULL,
	"actualcost" numeric NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_TransactionHistoryArchive_TransactionID" PRIMARY KEY("transactionid"),
	CONSTRAINT "CK_TransactionHistoryArchive_TransactionType" CHECK(upper(transactiontype::text) = ANY(ARRAY['W'::text, 'S'::text, 'P'::text]))
);

CREATE TABLE "production"."unitmeasure" (
	"unitmeasurecode" char(3) NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_UnitMeasure_UnitMeasureCode" PRIMARY KEY("unitmeasurecode")
);

CREATE TABLE "production"."workorder" (
	"workorderid" integer NOT NULL,
	"productid" integer NOT NULL,
	"orderqty" integer NOT NULL,
	"scrappedqty" smallint NOT NULL,
	"startdate" timestamp NOT NULL,
	"enddate" timestamp,
	"duedate" timestamp NOT NULL,
	"scrapreasonid" smallint,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_WorkOrder_WorkOrderID" PRIMARY KEY("workorderid"),
	CONSTRAINT "CK_WorkOrder_EndDate" CHECK(enddate >= startdate OR enddate IS NULL),
	CONSTRAINT "CK_WorkOrder_OrderQty" CHECK(orderqty > 0),
	CONSTRAINT "CK_WorkOrder_ScrappedQty" CHECK(scrappedqty >= 0)
);

CREATE TABLE "production"."workorderrouting" (
	"workorderid" integer NOT NULL,
	"productid" integer NOT NULL,
	"operationsequence" smallint NOT NULL,
	"locationid" smallint NOT NULL,
	"scheduledstartdate" timestamp NOT NULL,
	"scheduledenddate" timestamp NOT NULL,
	"actualstartdate" timestamp,
	"actualenddate" timestamp,
	"actualresourcehrs" numeric(9,4),
	"plannedcost" numeric NOT NULL,
	"actualcost" numeric,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_WorkOrderRouting_WorkOrderID_ProductID_OperationSequence" PRIMARY KEY("workorderid", "productid", "operationsequence"),
	CONSTRAINT "CK_WorkOrderRouting_ActualCost" CHECK(actualcost > 0.00),
	CONSTRAINT "CK_WorkOrderRouting_ActualEndDate" CHECK(actualenddate >= actualstartdate OR actualenddate IS NULL OR actualstartdate IS NULL),
	CONSTRAINT "CK_WorkOrderRouting_ActualResourceHrs" CHECK(actualresourcehrs >= 0.0000),
	CONSTRAINT "CK_WorkOrderRouting_PlannedCost" CHECK(plannedcost > 0.00),
	CONSTRAINT "CK_WorkOrderRouting_ScheduledEndDate" CHECK(scheduledenddate >= scheduledstartdate)
);

CREATE TABLE "purchasing"."productvendor" (
	"productid" integer NOT NULL,
	"businessentityid" integer NOT NULL,
	"averageleadtime" integer NOT NULL,
	"standardprice" numeric NOT NULL,
	"lastreceiptcost" numeric,
	"lastreceiptdate" timestamp,
	"minorderqty" integer NOT NULL,
	"maxorderqty" integer NOT NULL,
	"onorderqty" integer,
	"unitmeasurecode" char(3) NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ProductVendor_ProductID_BusinessEntityID" PRIMARY KEY("productid", "businessentityid"),
	CONSTRAINT "CK_ProductVendor_AverageLeadTime" CHECK(averageleadtime >= 1),
	CONSTRAINT "CK_ProductVendor_LastReceiptCost" CHECK(lastreceiptcost > 0.00),
	CONSTRAINT "CK_ProductVendor_MaxOrderQty" CHECK(maxorderqty >= 1),
	CONSTRAINT "CK_ProductVendor_MinOrderQty" CHECK(minorderqty >= 1),
	CONSTRAINT "CK_ProductVendor_OnOrderQty" CHECK(onorderqty >= 0),
	CONSTRAINT "CK_ProductVendor_StandardPrice" CHECK(standardprice > 0.00)
);

CREATE TABLE "purchasing"."purchaseorderdetail" (
	"purchaseorderid" integer NOT NULL,
	"purchaseorderdetailid" integer NOT NULL,
	"duedate" timestamp NOT NULL,
	"orderqty" smallint NOT NULL,
	"productid" integer NOT NULL,
	"unitprice" numeric NOT NULL,
	"receivedqty" numeric(8,2) NOT NULL,
	"rejectedqty" numeric(8,2) NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_PurchaseOrderDetail_PurchaseOrderID_PurchaseOrderDetailID" PRIMARY KEY("purchaseorderid", "purchaseorderdetailid"),
	CONSTRAINT "CK_PurchaseOrderDetail_OrderQty" CHECK(orderqty > 0),
	CONSTRAINT "CK_PurchaseOrderDetail_ReceivedQty" CHECK(receivedqty >= 0.00),
	CONSTRAINT "CK_PurchaseOrderDetail_RejectedQty" CHECK(rejectedqty >= 0.00),
	CONSTRAINT "CK_PurchaseOrderDetail_UnitPrice" CHECK(unitprice >= 0.00)
);

CREATE TABLE "purchasing"."purchaseorderheader" (
	"purchaseorderid" integer NOT NULL,
	"revisionnumber" smallint NOT NULL DEFAULT 0,
	"status" smallint NOT NULL DEFAULT 1,
	"employeeid" integer NOT NULL,
	"vendorid" integer NOT NULL,
	"shipmethodid" integer NOT NULL,
	"orderdate" timestamp NOT NULL DEFAULT now(),
	"shipdate" timestamp,
	"subtotal" numeric NOT NULL DEFAULT 0.00,
	"taxamt" numeric NOT NULL DEFAULT 0.00,
	"freight" numeric NOT NULL DEFAULT 0.00,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_PurchaseOrderHeader_PurchaseOrderID" PRIMARY KEY("purchaseorderid"),
	CONSTRAINT "CK_PurchaseOrderHeader_Freight" CHECK(freight >= 0.00),
	CONSTRAINT "CK_PurchaseOrderHeader_ShipDate" CHECK(shipdate >= orderdate OR shipdate IS NULL),
	CONSTRAINT "CK_PurchaseOrderHeader_Status" CHECK(status >= 1 AND status <= 4),
	CONSTRAINT "CK_PurchaseOrderHeader_SubTotal" CHECK(subtotal >= 0.00),
	CONSTRAINT "CK_PurchaseOrderHeader_TaxAmt" CHECK(taxamt >= 0.00)
);

CREATE TABLE "purchasing"."shipmethod" (
	"shipmethodid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"shipbase" numeric NOT NULL DEFAULT 0.00,
	"shiprate" numeric NOT NULL DEFAULT 0.00,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ShipMethod_ShipMethodID" PRIMARY KEY("shipmethodid"),
	CONSTRAINT "CK_ShipMethod_ShipBase" CHECK(shipbase > 0.00),
	CONSTRAINT "CK_ShipMethod_ShipRate" CHECK(shiprate > 0.00)
);

CREATE TABLE "purchasing"."vendor" (
	"businessentityid" integer NOT NULL,
	"accountnumber" "AccountNumber" NOT NULL,
	"name" "Name" NOT NULL,
	"creditrating" smallint NOT NULL,
	"preferredvendorstatus" "Flag" NOT NULL DEFAULT true,
	"activeflag" "Flag" NOT NULL DEFAULT true,
	"purchasingwebserviceurl" varchar(1024),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Vendor_BusinessEntityID" PRIMARY KEY("businessentityid"),
	CONSTRAINT "CK_Vendor_CreditRating" CHECK(creditrating >= 1 AND creditrating <= 5)
);

CREATE TABLE "sales"."countryregioncurrency" (
	"countryregioncode" varchar(3) NOT NULL,
	"currencycode" char(3) NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_CountryRegionCurrency_CountryRegionCode_CurrencyCode" PRIMARY KEY("countryregioncode", "currencycode")
);

CREATE TABLE "sales"."creditcard" (
	"creditcardid" integer NOT NULL,
	"cardtype" varchar(50) NOT NULL,
	"cardnumber" varchar(25) NOT NULL,
	"expmonth" smallint NOT NULL,
	"expyear" smallint NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_CreditCard_CreditCardID" PRIMARY KEY("creditcardid")
);

CREATE TABLE "sales"."currency" (
	"currencycode" char(3) NOT NULL,
	"name" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Currency_CurrencyCode" PRIMARY KEY("currencycode")
);

CREATE TABLE "sales"."currencyrate" (
	"currencyrateid" integer NOT NULL,
	"currencyratedate" timestamp NOT NULL,
	"fromcurrencycode" char(3) NOT NULL,
	"tocurrencycode" char(3) NOT NULL,
	"averagerate" numeric NOT NULL,
	"endofdayrate" numeric NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_CurrencyRate_CurrencyRateID" PRIMARY KEY("currencyrateid")
);

CREATE TABLE "sales"."customer" (
	"customerid" integer NOT NULL,
	"personid" integer,
	"storeid" integer,
	"territoryid" integer,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Customer_CustomerID" PRIMARY KEY("customerid")
);

CREATE TABLE "sales"."personcreditcard" (
	"businessentityid" integer NOT NULL,
	"creditcardid" integer NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_PersonCreditCard_BusinessEntityID_CreditCardID" PRIMARY KEY("businessentityid", "creditcardid")
);

CREATE TABLE "sales"."salesorderdetail" (
	"salesorderid" integer NOT NULL,
	"salesorderdetailid" integer NOT NULL,
	"carriertrackingnumber" varchar(25),
	"orderqty" smallint NOT NULL,
	"productid" integer NOT NULL,
	"specialofferid" integer NOT NULL,
	"unitprice" numeric NOT NULL,
	"unitpricediscount" numeric NOT NULL DEFAULT 0.0,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesOrderDetail_SalesOrderID_SalesOrderDetailID" PRIMARY KEY("salesorderid", "salesorderdetailid"),
	CONSTRAINT "CK_SalesOrderDetail_OrderQty" CHECK(orderqty > 0),
	CONSTRAINT "CK_SalesOrderDetail_UnitPrice" CHECK(unitprice >= 0.00),
	CONSTRAINT "CK_SalesOrderDetail_UnitPriceDiscount" CHECK(unitpricediscount >= 0.00)
);

CREATE TABLE "sales"."salesorderheader" (
	"salesorderid" integer NOT NULL,
	"revisionnumber" smallint NOT NULL DEFAULT 0,
	"orderdate" timestamp NOT NULL DEFAULT now(),
	"duedate" timestamp NOT NULL,
	"shipdate" timestamp,
	"status" smallint NOT NULL DEFAULT 1,
	"onlineorderflag" "Flag" NOT NULL DEFAULT true,
	"purchaseordernumber" "OrderNumber",
	"accountnumber" "AccountNumber",
	"customerid" integer NOT NULL,
	"salespersonid" integer,
	"territoryid" integer,
	"billtoaddressid" integer NOT NULL,
	"shiptoaddressid" integer NOT NULL,
	"shipmethodid" integer NOT NULL,
	"creditcardid" integer,
	"creditcardapprovalcode" varchar(15),
	"currencyrateid" integer,
	"subtotal" numeric NOT NULL DEFAULT 0.00,
	"taxamt" numeric NOT NULL DEFAULT 0.00,
	"freight" numeric NOT NULL DEFAULT 0.00,
	"totaldue" numeric,
	"comment" varchar(128),
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesOrderHeader_SalesOrderID" PRIMARY KEY("salesorderid"),
	CONSTRAINT "CK_SalesOrderHeader_DueDate" CHECK(duedate >= orderdate),
	CONSTRAINT "CK_SalesOrderHeader_Freight" CHECK(freight >= 0.00),
	CONSTRAINT "CK_SalesOrderHeader_ShipDate" CHECK(shipdate >= orderdate OR shipdate IS NULL),
	CONSTRAINT "CK_SalesOrderHeader_Status" CHECK(status >= 0 AND status <= 8),
	CONSTRAINT "CK_SalesOrderHeader_SubTotal" CHECK(subtotal >= 0.00),
	CONSTRAINT "CK_SalesOrderHeader_TaxAmt" CHECK(taxamt >= 0.00)
);

CREATE TABLE "sales"."salesorderheadersalesreason" (
	"salesorderid" integer NOT NULL,
	"salesreasonid" integer NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesOrderHeaderSalesReason_SalesOrderID_SalesReasonID" PRIMARY KEY("salesorderid", "salesreasonid")
);

CREATE TABLE "sales"."salesperson" (
	"businessentityid" integer NOT NULL,
	"territoryid" integer,
	"salesquota" numeric,
	"bonus" numeric NOT NULL DEFAULT 0.00,
	"commissionpct" numeric NOT NULL DEFAULT 0.00,
	"salesytd" numeric NOT NULL DEFAULT 0.00,
	"saleslastyear" numeric NOT NULL DEFAULT 0.00,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesPerson_BusinessEntityID" PRIMARY KEY("businessentityid"),
	CONSTRAINT "CK_SalesPerson_Bonus" CHECK(bonus >= 0.00),
	CONSTRAINT "CK_SalesPerson_CommissionPct" CHECK(commissionpct >= 0.00),
	CONSTRAINT "CK_SalesPerson_SalesLastYear" CHECK(saleslastyear >= 0.00),
	CONSTRAINT "CK_SalesPerson_SalesQuota" CHECK(salesquota > 0.00),
	CONSTRAINT "CK_SalesPerson_SalesYTD" CHECK(salesytd >= 0.00)
);

CREATE TABLE "sales"."salespersonquotahistory" (
	"businessentityid" integer NOT NULL,
	"quotadate" timestamp NOT NULL,
	"salesquota" numeric NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesPersonQuotaHistory_BusinessEntityID_QuotaDate" PRIMARY KEY("businessentityid", "quotadate"),
	CONSTRAINT "CK_SalesPersonQuotaHistory_SalesQuota" CHECK(salesquota > 0.00)
);

CREATE TABLE "sales"."salesreason" (
	"salesreasonid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"reasontype" "Name" NOT NULL,
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesReason_SalesReasonID" PRIMARY KEY("salesreasonid")
);

CREATE TABLE "sales"."salestaxrate" (
	"salestaxrateid" integer NOT NULL,
	"stateprovinceid" integer NOT NULL,
	"taxtype" smallint NOT NULL,
	"taxrate" numeric NOT NULL DEFAULT 0.00,
	"name" "Name" NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesTaxRate_SalesTaxRateID" PRIMARY KEY("salestaxrateid"),
	CONSTRAINT "CK_SalesTaxRate_TaxType" CHECK(taxtype >= 1 AND taxtype <= 3)
);

CREATE TABLE "sales"."salesterritory" (
	"territoryid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"countryregioncode" varchar(3) NOT NULL,
	"group" varchar(50) NOT NULL,
	"salesytd" numeric NOT NULL DEFAULT 0.00,
	"saleslastyear" numeric NOT NULL DEFAULT 0.00,
	"costytd" numeric NOT NULL DEFAULT 0.00,
	"costlastyear" numeric NOT NULL DEFAULT 0.00,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesTerritory_TerritoryID" PRIMARY KEY("territoryid"),
	CONSTRAINT "CK_SalesTerritory_CostLastYear" CHECK(costlastyear >= 0.00),
	CONSTRAINT "CK_SalesTerritory_CostYTD" CHECK(costytd >= 0.00),
	CONSTRAINT "CK_SalesTerritory_SalesLastYear" CHECK(saleslastyear >= 0.00),
	CONSTRAINT "CK_SalesTerritory_SalesYTD" CHECK(salesytd >= 0.00)
);

CREATE TABLE "sales"."salesterritoryhistory" (
	"businessentityid" integer NOT NULL,
	"territoryid" integer NOT NULL,
	"startdate" timestamp NOT NULL,
	"enddate" timestamp,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SalesTerritoryHistory_BusinessEntityID_StartDate_TerritoryID" PRIMARY KEY("businessentityid", "startdate", "territoryid"),
	CONSTRAINT "CK_SalesTerritoryHistory_EndDate" CHECK(enddate >= startdate OR enddate IS NULL)
);

CREATE TABLE "sales"."shoppingcartitem" (
	"shoppingcartitemid" integer NOT NULL,
	"shoppingcartid" varchar(50) NOT NULL,
	"quantity" integer NOT NULL DEFAULT 1,
	"productid" integer NOT NULL,
	"datecreated" timestamp NOT NULL DEFAULT now(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_ShoppingCartItem_ShoppingCartItemID" PRIMARY KEY("shoppingcartitemid"),
	CONSTRAINT "CK_ShoppingCartItem_Quantity" CHECK(quantity >= 1)
);

CREATE TABLE "sales"."specialoffer" (
	"specialofferid" integer NOT NULL,
	"description" varchar(255) NOT NULL,
	"discountpct" numeric NOT NULL DEFAULT 0.00,
	"type" varchar(50) NOT NULL,
	"category" varchar(50) NOT NULL,
	"startdate" timestamp NOT NULL,
	"enddate" timestamp NOT NULL,
	"minqty" integer NOT NULL DEFAULT 0,
	"maxqty" integer,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SpecialOffer_SpecialOfferID" PRIMARY KEY("specialofferid"),
	CONSTRAINT "CK_SpecialOffer_DiscountPct" CHECK(discountpct >= 0.00),
	CONSTRAINT "CK_SpecialOffer_EndDate" CHECK(enddate >= startdate),
	CONSTRAINT "CK_SpecialOffer_MaxQty" CHECK(maxqty >= 0),
	CONSTRAINT "CK_SpecialOffer_MinQty" CHECK(minqty >= 0)
);

CREATE TABLE "sales"."specialofferproduct" (
	"specialofferid" integer NOT NULL,
	"productid" integer NOT NULL,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_SpecialOfferProduct_SpecialOfferID_ProductID" PRIMARY KEY("specialofferid", "productid")
);

CREATE TABLE "sales"."store" (
	"businessentityid" integer NOT NULL,
	"name" "Name" NOT NULL,
	"salespersonid" integer,
	"demographics" xml,
	"rowguid" uuid NOT NULL DEFAULT public.uuid_generate_v1(),
	"modifieddate" timestamp NOT NULL DEFAULT now(),
	CONSTRAINT "PK_Store_BusinessEntityID" PRIMARY KEY("businessentityid")
);

ALTER TABLE "humanresources"."employee" ADD CONSTRAINT "FK_Employee_Person_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "humanresources"."employeedepartmenthistory" ADD CONSTRAINT "FK_EmployeeDepartmentHistory_Department_DepartmentID" FOREIGN KEY ("departmentid")
	REFERENCES "humanresources"."department"("departmentid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "humanresources"."employeedepartmenthistory" ADD CONSTRAINT "FK_EmployeeDepartmentHistory_Employee_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "humanresources"."employee"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "humanresources"."employeedepartmenthistory" ADD CONSTRAINT "FK_EmployeeDepartmentHistory_Shift_ShiftID" FOREIGN KEY ("shiftid")
	REFERENCES "humanresources"."shift"("shiftid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "humanresources"."employeepayhistory" ADD CONSTRAINT "FK_EmployeePayHistory_Employee_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "humanresources"."employee"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "humanresources"."jobcandidate" ADD CONSTRAINT "FK_JobCandidate_Employee_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "humanresources"."employee"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."address" ADD CONSTRAINT "FK_Address_StateProvince_StateProvinceID" FOREIGN KEY ("stateprovinceid")
	REFERENCES "person"."stateprovince"("stateprovinceid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."businessentityaddress" ADD CONSTRAINT "FK_BusinessEntityAddress_AddressType_AddressTypeID" FOREIGN KEY ("addresstypeid")
	REFERENCES "person"."addresstype"("addresstypeid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."businessentityaddress" ADD CONSTRAINT "FK_BusinessEntityAddress_Address_AddressID" FOREIGN KEY ("addressid")
	REFERENCES "person"."address"("addressid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."businessentityaddress" ADD CONSTRAINT "FK_BusinessEntityAddress_BusinessEntity_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."businessentity"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."businessentitycontact" ADD CONSTRAINT "FK_BusinessEntityContact_BusinessEntity_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."businessentity"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."businessentitycontact" ADD CONSTRAINT "FK_BusinessEntityContact_ContactType_ContactTypeID" FOREIGN KEY ("contacttypeid")
	REFERENCES "person"."contacttype"("contacttypeid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."businessentitycontact" ADD CONSTRAINT "FK_BusinessEntityContact_Person_PersonID" FOREIGN KEY ("personid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."emailaddress" ADD CONSTRAINT "FK_EmailAddress_Person_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."password" ADD CONSTRAINT "FK_Password_Person_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."person" ADD CONSTRAINT "FK_Person_BusinessEntity_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."businessentity"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."personphone" ADD CONSTRAINT "FK_PersonPhone_Person_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."personphone" ADD CONSTRAINT "FK_PersonPhone_PhoneNumberType_PhoneNumberTypeID" FOREIGN KEY ("phonenumbertypeid")
	REFERENCES "person"."phonenumbertype"("phonenumbertypeid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."stateprovince" ADD CONSTRAINT "FK_StateProvince_CountryRegion_CountryRegionCode" FOREIGN KEY ("countryregioncode")
	REFERENCES "person"."countryregion"("countryregioncode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "person"."stateprovince" ADD CONSTRAINT "FK_StateProvince_SalesTerritory_TerritoryID" FOREIGN KEY ("territoryid")
	REFERENCES "sales"."salesterritory"("territoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."billofmaterials" ADD CONSTRAINT "FK_BillOfMaterials_Product_ComponentID" FOREIGN KEY ("componentid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."billofmaterials" ADD CONSTRAINT "FK_BillOfMaterials_Product_ProductAssemblyID" FOREIGN KEY ("productassemblyid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."billofmaterials" ADD CONSTRAINT "FK_BillOfMaterials_UnitMeasure_UnitMeasureCode" FOREIGN KEY ("unitmeasurecode")
	REFERENCES "production"."unitmeasure"("unitmeasurecode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."document" ADD CONSTRAINT "FK_Document_Employee_Owner" FOREIGN KEY ("owner")
	REFERENCES "humanresources"."employee"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."product" ADD CONSTRAINT "FK_Product_ProductModel_ProductModelID" FOREIGN KEY ("productmodelid")
	REFERENCES "production"."productmodel"("productmodelid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."product" ADD CONSTRAINT "FK_Product_ProductSubcategory_ProductSubcategoryID" FOREIGN KEY ("productsubcategoryid")
	REFERENCES "production"."productsubcategory"("productsubcategoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."product" ADD CONSTRAINT "FK_Product_UnitMeasure_SizeUnitMeasureCode" FOREIGN KEY ("sizeunitmeasurecode")
	REFERENCES "production"."unitmeasure"("unitmeasurecode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."product" ADD CONSTRAINT "FK_Product_UnitMeasure_WeightUnitMeasureCode" FOREIGN KEY ("weightunitmeasurecode")
	REFERENCES "production"."unitmeasure"("unitmeasurecode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productcosthistory" ADD CONSTRAINT "FK_ProductCostHistory_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productdocument" ADD CONSTRAINT "FK_ProductDocument_Document_DocumentNode" FOREIGN KEY ("documentnode")
	REFERENCES "production"."document"("documentnode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productdocument" ADD CONSTRAINT "FK_ProductDocument_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productinventory" ADD CONSTRAINT "FK_ProductInventory_Location_LocationID" FOREIGN KEY ("locationid")
	REFERENCES "production"."location"("locationid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productinventory" ADD CONSTRAINT "FK_ProductInventory_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productlistpricehistory" ADD CONSTRAINT "FK_ProductListPriceHistory_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productmodelillustration" ADD CONSTRAINT "FK_ProductModelIllustration_Illustration_IllustrationID" FOREIGN KEY ("illustrationid")
	REFERENCES "production"."illustration"("illustrationid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productmodelillustration" ADD CONSTRAINT "FK_ProductModelIllustration_ProductModel_ProductModelID" FOREIGN KEY ("productmodelid")
	REFERENCES "production"."productmodel"("productmodelid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productmodelproductdescriptionculture" ADD CONSTRAINT "FK_ProductModelProductDescriptionCulture_Culture_CultureID" FOREIGN KEY ("cultureid")
	REFERENCES "production"."culture"("cultureid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productmodelproductdescriptionculture" ADD CONSTRAINT "FK_ProductModelProductDescriptionCulture_ProductDescription_Pro" FOREIGN KEY ("productdescriptionid")
	REFERENCES "production"."productdescription"("productdescriptionid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productmodelproductdescriptionculture" ADD CONSTRAINT "FK_ProductModelProductDescriptionCulture_ProductModel_ProductMo" FOREIGN KEY ("productmodelid")
	REFERENCES "production"."productmodel"("productmodelid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productproductphoto" ADD CONSTRAINT "FK_ProductProductPhoto_ProductPhoto_ProductPhotoID" FOREIGN KEY ("productphotoid")
	REFERENCES "production"."productphoto"("productphotoid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productproductphoto" ADD CONSTRAINT "FK_ProductProductPhoto_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."productsubcategory" ADD CONSTRAINT "FK_ProductSubcategory_ProductCategory_ProductCategoryID" FOREIGN KEY ("productcategoryid")
	REFERENCES "production"."productcategory"("productcategoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."transactionhistory" ADD CONSTRAINT "FK_TransactionHistory_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."workorder" ADD CONSTRAINT "FK_WorkOrder_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."workorder" ADD CONSTRAINT "FK_WorkOrder_ScrapReason_ScrapReasonID" FOREIGN KEY ("scrapreasonid")
	REFERENCES "production"."scrapreason"("scrapreasonid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."workorderrouting" ADD CONSTRAINT "FK_WorkOrderRouting_Location_LocationID" FOREIGN KEY ("locationid")
	REFERENCES "production"."location"("locationid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "production"."workorderrouting" ADD CONSTRAINT "FK_WorkOrderRouting_WorkOrder_WorkOrderID" FOREIGN KEY ("workorderid")
	REFERENCES "production"."workorder"("workorderid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."productvendor" ADD CONSTRAINT "FK_ProductVendor_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."productvendor" ADD CONSTRAINT "FK_ProductVendor_UnitMeasure_UnitMeasureCode" FOREIGN KEY ("unitmeasurecode")
	REFERENCES "production"."unitmeasure"("unitmeasurecode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."productvendor" ADD CONSTRAINT "FK_ProductVendor_Vendor_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "purchasing"."vendor"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."purchaseorderdetail" ADD CONSTRAINT "FK_PurchaseOrderDetail_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."purchaseorderdetail" ADD CONSTRAINT "FK_PurchaseOrderDetail_PurchaseOrderHeader_PurchaseOrderID" FOREIGN KEY ("purchaseorderid")
	REFERENCES "purchasing"."purchaseorderheader"("purchaseorderid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."purchaseorderheader" ADD CONSTRAINT "FK_PurchaseOrderHeader_Employee_EmployeeID" FOREIGN KEY ("employeeid")
	REFERENCES "humanresources"."employee"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."purchaseorderheader" ADD CONSTRAINT "FK_PurchaseOrderHeader_ShipMethod_ShipMethodID" FOREIGN KEY ("shipmethodid")
	REFERENCES "purchasing"."shipmethod"("shipmethodid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."purchaseorderheader" ADD CONSTRAINT "FK_PurchaseOrderHeader_Vendor_VendorID" FOREIGN KEY ("vendorid")
	REFERENCES "purchasing"."vendor"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "purchasing"."vendor" ADD CONSTRAINT "FK_Vendor_BusinessEntity_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."businessentity"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."countryregioncurrency" ADD CONSTRAINT "FK_CountryRegionCurrency_CountryRegion_CountryRegionCode" FOREIGN KEY ("countryregioncode")
	REFERENCES "person"."countryregion"("countryregioncode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."countryregioncurrency" ADD CONSTRAINT "FK_CountryRegionCurrency_Currency_CurrencyCode" FOREIGN KEY ("currencycode")
	REFERENCES "sales"."currency"("currencycode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."currencyrate" ADD CONSTRAINT "FK_CurrencyRate_Currency_FromCurrencyCode" FOREIGN KEY ("fromcurrencycode")
	REFERENCES "sales"."currency"("currencycode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."currencyrate" ADD CONSTRAINT "FK_CurrencyRate_Currency_ToCurrencyCode" FOREIGN KEY ("tocurrencycode")
	REFERENCES "sales"."currency"("currencycode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."customer" ADD CONSTRAINT "FK_Customer_Person_PersonID" FOREIGN KEY ("personid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."customer" ADD CONSTRAINT "FK_Customer_SalesTerritory_TerritoryID" FOREIGN KEY ("territoryid")
	REFERENCES "sales"."salesterritory"("territoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."customer" ADD CONSTRAINT "FK_Customer_Store_StoreID" FOREIGN KEY ("storeid")
	REFERENCES "sales"."store"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."personcreditcard" ADD CONSTRAINT "FK_PersonCreditCard_CreditCard_CreditCardID" FOREIGN KEY ("creditcardid")
	REFERENCES "sales"."creditcard"("creditcardid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."personcreditcard" ADD CONSTRAINT "FK_PersonCreditCard_Person_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."person"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderdetail" ADD CONSTRAINT "FK_SalesOrderDetail_SalesOrderHeader_SalesOrderID" FOREIGN KEY ("salesorderid")
	REFERENCES "sales"."salesorderheader"("salesorderid")
	ON DELETE CASCADE
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderdetail" ADD CONSTRAINT "FK_SalesOrderDetail_SpecialOfferProduct_SpecialOfferIDProductID" FOREIGN KEY ("specialofferid", "productid")
	REFERENCES "sales"."specialofferproduct"("specialofferid", "productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_Address_BillToAddressID" FOREIGN KEY ("billtoaddressid")
	REFERENCES "person"."address"("addressid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_Address_ShipToAddressID" FOREIGN KEY ("shiptoaddressid")
	REFERENCES "person"."address"("addressid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_CreditCard_CreditCardID" FOREIGN KEY ("creditcardid")
	REFERENCES "sales"."creditcard"("creditcardid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_CurrencyRate_CurrencyRateID" FOREIGN KEY ("currencyrateid")
	REFERENCES "sales"."currencyrate"("currencyrateid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_Customer_CustomerID" FOREIGN KEY ("customerid")
	REFERENCES "sales"."customer"("customerid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_SalesPerson_SalesPersonID" FOREIGN KEY ("salespersonid")
	REFERENCES "sales"."salesperson"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_SalesTerritory_TerritoryID" FOREIGN KEY ("territoryid")
	REFERENCES "sales"."salesterritory"("territoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheader" ADD CONSTRAINT "FK_SalesOrderHeader_ShipMethod_ShipMethodID" FOREIGN KEY ("shipmethodid")
	REFERENCES "purchasing"."shipmethod"("shipmethodid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheadersalesreason" ADD CONSTRAINT "FK_SalesOrderHeaderSalesReason_SalesOrderHeader_SalesOrderID" FOREIGN KEY ("salesorderid")
	REFERENCES "sales"."salesorderheader"("salesorderid")
	ON DELETE CASCADE
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesorderheadersalesreason" ADD CONSTRAINT "FK_SalesOrderHeaderSalesReason_SalesReason_SalesReasonID" FOREIGN KEY ("salesreasonid")
	REFERENCES "sales"."salesreason"("salesreasonid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesperson" ADD CONSTRAINT "FK_SalesPerson_Employee_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "humanresources"."employee"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesperson" ADD CONSTRAINT "FK_SalesPerson_SalesTerritory_TerritoryID" FOREIGN KEY ("territoryid")
	REFERENCES "sales"."salesterritory"("territoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salespersonquotahistory" ADD CONSTRAINT "FK_SalesPersonQuotaHistory_SalesPerson_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "sales"."salesperson"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salestaxrate" ADD CONSTRAINT "FK_SalesTaxRate_StateProvince_StateProvinceID" FOREIGN KEY ("stateprovinceid")
	REFERENCES "person"."stateprovince"("stateprovinceid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesterritory" ADD CONSTRAINT "FK_SalesTerritory_CountryRegion_CountryRegionCode" FOREIGN KEY ("countryregioncode")
	REFERENCES "person"."countryregion"("countryregioncode")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesterritoryhistory" ADD CONSTRAINT "FK_SalesTerritoryHistory_SalesPerson_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "sales"."salesperson"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."salesterritoryhistory" ADD CONSTRAINT "FK_SalesTerritoryHistory_SalesTerritory_TerritoryID" FOREIGN KEY ("territoryid")
	REFERENCES "sales"."salesterritory"("territoryid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."shoppingcartitem" ADD CONSTRAINT "FK_ShoppingCartItem_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."specialofferproduct" ADD CONSTRAINT "FK_SpecialOfferProduct_Product_ProductID" FOREIGN KEY ("productid")
	REFERENCES "production"."product"("productid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."specialofferproduct" ADD CONSTRAINT "FK_SpecialOfferProduct_SpecialOffer_SpecialOfferID" FOREIGN KEY ("specialofferid")
	REFERENCES "sales"."specialoffer"("specialofferid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."store" ADD CONSTRAINT "FK_Store_BusinessEntity_BusinessEntityID" FOREIGN KEY ("businessentityid")
	REFERENCES "person"."businessentity"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

ALTER TABLE "sales"."store" ADD CONSTRAINT "FK_Store_SalesPerson_SalesPersonID" FOREIGN KEY ("salespersonid")
	REFERENCES "sales"."salesperson"("businessentityid")
	ON DELETE NO ACTION
	ON UPDATE NO ACTION
	NOT DEFERRABLE;

CREATE MATERIALIZED VIEW "person"."vstateprovincecountryregion" AS
SELECT sp.stateprovinceid, sp.stateprovincecode, sp.isonlystateprovinceflag, sp.name AS stateprovincename, sp.territoryid, cr.countryregioncode, cr.name AS countryregionname FROM person.stateprovince sp JOIN person.countryregion cr ON sp.countryregioncode::text = cr.countryregioncode::text;

CREATE MATERIALIZED VIEW "production"."vproductanddescription" AS
SELECT p.productid, p.name, pm.name AS productmodel, pmx.cultureid, pd.description FROM production.product p JOIN production.productmodel pm ON p.productmodelid = pm.productmodelid JOIN production.productmodelproductdescriptionculture pmx ON pm.productmodelid = pmx.productmodelid JOIN production.productdescription pd ON pmx.productdescriptionid = pd.productdescriptionid;

CREATE VIEW "hr"."d" AS
SELECT departmentid AS id, departmentid, name, groupname, modifieddate FROM humanresources.department;

CREATE VIEW "hr"."e" AS
SELECT businessentityid AS id, businessentityid, nationalidnumber, loginid, jobtitle, birthdate, maritalstatus, gender, hiredate, salariedflag, vacationhours, sickleavehours, currentflag, rowguid, modifieddate, organizationnode FROM humanresources.employee;

CREATE VIEW "hr"."edh" AS
SELECT businessentityid AS id, businessentityid, departmentid, shiftid, startdate, enddate, modifieddate FROM humanresources.employeedepartmenthistory;

CREATE VIEW "hr"."eph" AS
SELECT businessentityid AS id, businessentityid, ratechangedate, rate, payfrequency, modifieddate FROM humanresources.employeepayhistory;

CREATE VIEW "hr"."jc" AS
SELECT jobcandidateid AS id, jobcandidateid, businessentityid, resume, modifieddate FROM humanresources.jobcandidate;

CREATE VIEW "hr"."s" AS
SELECT shiftid AS id, shiftid, name, starttime, endtime, modifieddate FROM humanresources.shift;

CREATE VIEW "humanresources"."vemployee" AS
SELECT e.businessentityid, p.title, p.firstname, p.middlename, p.lastname, p.suffix, e.jobtitle, pp.phonenumber, pnt.name AS phonenumbertype, ea.emailaddress, p.emailpromotion, a.addressline1, a.addressline2, a.city, sp.name AS stateprovincename, a.postalcode, cr.name AS countryregionname, p.additionalcontactinfo FROM humanresources.employee e JOIN person.person p ON p.businessentityid = e.businessentityid JOIN person.businessentityaddress bea ON bea.businessentityid = e.businessentityid JOIN person.address a ON a.addressid = bea.addressid JOIN person.stateprovince sp ON sp.stateprovinceid = a.stateprovinceid JOIN person.countryregion cr ON cr.countryregioncode::text = sp.countryregioncode::text LEFT JOIN person.personphone pp ON pp.businessentityid = p.businessentityid LEFT JOIN person.phonenumbertype pnt ON pp.phonenumbertypeid = pnt.phonenumbertypeid LEFT JOIN person.emailaddress ea ON p.businessentityid = ea.businessentityid;

CREATE VIEW "humanresources"."vemployeedepartment" AS
SELECT e.businessentityid, p.title, p.firstname, p.middlename, p.lastname, p.suffix, e.jobtitle, d.name AS department, d.groupname, edh.startdate FROM humanresources.employee e JOIN person.person p ON p.businessentityid = e.businessentityid JOIN humanresources.employeedepartmenthistory edh ON e.businessentityid = edh.businessentityid JOIN humanresources.department d ON edh.departmentid = d.departmentid WHERE edh.enddate IS NULL;

CREATE VIEW "humanresources"."vemployeedepartmenthistory" AS
SELECT e.businessentityid, p.title, p.firstname, p.middlename, p.lastname, p.suffix, s.name AS shift, d.name AS department, d.groupname, edh.startdate, edh.enddate FROM humanresources.employee e JOIN person.person p ON p.businessentityid = e.businessentityid JOIN humanresources.employeedepartmenthistory edh ON e.businessentityid = edh.businessentityid JOIN humanresources.department d ON edh.departmentid = d.departmentid JOIN humanresources.shift s ON s.shiftid = edh.shiftid;

CREATE VIEW "humanresources"."vjobcandidate" AS
SELECT jobcandidateid, businessentityid, (xpath('/n:Resume/n:Name/n:Name.Prefix/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(30) AS "Name.Prefix", (xpath('/n:Resume/n:Name/n:Name.First/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(30) AS "Name.First", (xpath('/n:Resume/n:Name/n:Name.Middle/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(30) AS "Name.Middle", (xpath('/n:Resume/n:Name/n:Name.Last/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(30) AS "Name.Last", (xpath('/n:Resume/n:Name/n:Name.Suffix/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(30) AS "Name.Suffix", (xpath('/n:Resume/n:Skills/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar AS "Skills", (xpath('n:Address/n:Addr.Type/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(30) AS "Addr.Type", (xpath('n:Address/n:Addr.Location/n:Location/n:Loc.CountryRegion/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(100) AS "Addr.Loc.CountryRegion", (xpath('n:Address/n:Addr.Location/n:Location/n:Loc.State/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(100) AS "Addr.Loc.State", (xpath('n:Address/n:Addr.Location/n:Location/n:Loc.City/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(100) AS "Addr.Loc.City", (xpath('n:Address/n:Addr.PostalCode/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar(20) AS "Addr.PostalCode", (xpath('/n:Resume/n:EMail/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar AS "EMail", (xpath('/n:Resume/n:WebSite/text()'::text, resume, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))[1]::varchar AS "WebSite", modifieddate FROM humanresources.jobcandidate;

CREATE VIEW "humanresources"."vjobcandidateeducation" AS
SELECT jobcandidateid, (xpath('/root/ns:Education/ns:Edu.Level/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(50) AS "Edu.Level", (xpath('/root/ns:Education/ns:Edu.StartDate/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(20)::date AS "Edu.StartDate", (xpath('/root/ns:Education/ns:Edu.EndDate/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(20)::date AS "Edu.EndDate", (xpath('/root/ns:Education/ns:Edu.Degree/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(50) AS "Edu.Degree", (xpath('/root/ns:Education/ns:Edu.Major/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(50) AS "Edu.Major", (xpath('/root/ns:Education/ns:Edu.Minor/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(50) AS "Edu.Minor", (xpath('/root/ns:Education/ns:Edu.GPA/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(5) AS "Edu.GPA", (xpath('/root/ns:Education/ns:Edu.GPAScale/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(5) AS "Edu.GPAScale", (xpath('/root/ns:Education/ns:Edu.School/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(100) AS "Edu.School", (xpath('/root/ns:Education/ns:Edu.Location/ns:Location/ns:Loc.CountryRegion/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(100) AS "Edu.Loc.CountryRegion", (xpath('/root/ns:Education/ns:Edu.Location/ns:Location/ns:Loc.State/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(100) AS "Edu.Loc.State", (xpath('/root/ns:Education/ns:Edu.Location/ns:Location/ns:Loc.City/text()'::text, doc, '{{ns,http://adventureworks.com}}'::text[]))[1]::varchar(100) AS "Edu.Loc.City" FROM (SELECT unnesting.jobcandidateid, CAST(('<root xmlns:ns="http://adventureworks.com">'::text || unnesting.education::varchar::text) || '</root>'::text AS xml) AS doc FROM (SELECT jobcandidate.jobcandidateid, unnest(xpath('/ns:Resume/ns:Education'::text, jobcandidate.resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[])) AS education FROM humanresources.jobcandidate) unnesting) jc;

CREATE VIEW "humanresources"."vjobcandidateemployment" AS
SELECT jobcandidateid, unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.StartDate/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar(20)::date AS "Emp.StartDate", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.EndDate/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar(20)::date AS "Emp.EndDate", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.OrgName/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar(100) AS "Emp.OrgName", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.JobTitle/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar(100) AS "Emp.JobTitle", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.Responsibility/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar AS "Emp.Responsibility", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.FunctionCategory/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar AS "Emp.FunctionCategory", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.IndustryCategory/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar AS "Emp.IndustryCategory", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.Location/ns:Location/ns:Loc.CountryRegion/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar AS "Emp.Loc.CountryRegion", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.Location/ns:Location/ns:Loc.State/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar AS "Emp.Loc.State", unnest(xpath('/ns:Resume/ns:Employment/ns:Emp.Location/ns:Location/ns:Loc.City/text()'::text, resume, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/Resume}}'::text[]))::varchar AS "Emp.Loc.City" FROM humanresources.jobcandidate;

CREATE VIEW "pe"."a" AS
SELECT addressid AS id, addressid, addressline1, addressline2, city, stateprovinceid, postalcode, spatiallocation, rowguid, modifieddate FROM person.address;

CREATE VIEW "pe"."at" AS
SELECT addresstypeid AS id, addresstypeid, name, rowguid, modifieddate FROM person.addresstype;

CREATE VIEW "pe"."be" AS
SELECT businessentityid AS id, businessentityid, rowguid, modifieddate FROM person.businessentity;

CREATE VIEW "pe"."bea" AS
SELECT businessentityid AS id, businessentityid, addressid, addresstypeid, rowguid, modifieddate FROM person.businessentityaddress;

CREATE VIEW "pe"."bec" AS
SELECT businessentityid AS id, businessentityid, personid, contacttypeid, rowguid, modifieddate FROM person.businessentitycontact;

CREATE VIEW "pe"."cr" AS
SELECT countryregioncode, name, modifieddate FROM person.countryregion;

CREATE VIEW "pe"."ct" AS
SELECT contacttypeid AS id, contacttypeid, name, modifieddate FROM person.contacttype;

CREATE VIEW "pe"."e" AS
SELECT emailaddressid AS id, businessentityid, emailaddressid, emailaddress, rowguid, modifieddate FROM person.emailaddress;

CREATE VIEW "pe"."p" AS
SELECT businessentityid AS id, businessentityid, persontype, namestyle, title, firstname, middlename, lastname, suffix, emailpromotion, additionalcontactinfo, demographics, rowguid, modifieddate FROM person.person;

CREATE VIEW "pe"."pa" AS
SELECT businessentityid AS id, businessentityid, passwordhash, passwordsalt, rowguid, modifieddate FROM person.password;

CREATE VIEW "pe"."pnt" AS
SELECT phonenumbertypeid AS id, phonenumbertypeid, name, modifieddate FROM person.phonenumbertype;

CREATE VIEW "pe"."pp" AS
SELECT businessentityid AS id, businessentityid, phonenumber, phonenumbertypeid, modifieddate FROM person.personphone;

CREATE VIEW "pe"."sp" AS
SELECT stateprovinceid AS id, stateprovinceid, stateprovincecode, countryregioncode, isonlystateprovinceflag, name, territoryid, rowguid, modifieddate FROM person.stateprovince;

CREATE VIEW "person"."vadditionalcontactinfo" AS
SELECT p.businessentityid, p.firstname, p.middlename, p.lastname, (xpath('(act:telephoneNumber)[1]/act:number/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS telephonenumber, btrim((xpath('(act:telephoneNumber)[1]/act:SpecialInstructions/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1]::varchar::text) AS telephonespecialinstructions, (xpath('(act:homePostalAddress)[1]/act:Street/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS street, (xpath('(act:homePostalAddress)[1]/act:City/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS city, (xpath('(act:homePostalAddress)[1]/act:StateProvince/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS stateprovince, (xpath('(act:homePostalAddress)[1]/act:PostalCode/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS postalcode, (xpath('(act:homePostalAddress)[1]/act:CountryRegion/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS countryregion, (xpath('(act:homePostalAddress)[1]/act:SpecialInstructions/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS homeaddressspecialinstructions, (xpath('(act:eMail)[1]/act:eMailAddress/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS emailaddress, btrim((xpath('(act:eMail)[1]/act:SpecialInstructions/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1]::varchar::text) AS emailspecialinstructions, (xpath('((act:eMail)[1]/act:SpecialInstructions/act:telephoneNumber)[1]/act:number/text()'::text, additional.node, '{{act,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactTypes}}'::text[]))[1] AS emailtelephonenumber, p.rowguid, p.modifieddate FROM person.person p LEFT JOIN (SELECT person.businessentityid, unnest(xpath('/ci:AdditionalContactInfo'::text, person.additionalcontactinfo, '{{ci,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ContactInfo}}'::text[])) AS node FROM person.person WHERE person.additionalcontactinfo IS NOT NULL) additional ON p.businessentityid = additional.businessentityid;

CREATE VIEW "pr"."bom" AS
SELECT billofmaterialsid AS id, billofmaterialsid, productassemblyid, componentid, startdate, enddate, unitmeasurecode, bomlevel, perassemblyqty, modifieddate FROM production.billofmaterials;

CREATE VIEW "pr"."c" AS
SELECT cultureid AS id, cultureid, name, modifieddate FROM production.culture;

CREATE VIEW "pr"."d" AS
SELECT title, owner, folderflag, filename, fileextension, revision, changenumber, status, documentsummary, document, rowguid, modifieddate, documentnode FROM production.document;

CREATE VIEW "pr"."i" AS
SELECT illustrationid AS id, illustrationid, diagram, modifieddate FROM production.illustration;

CREATE VIEW "pr"."l" AS
SELECT locationid AS id, locationid, name, costrate, availability, modifieddate FROM production.location;

CREATE VIEW "pr"."p" AS
SELECT productid AS id, productid, name, productnumber, makeflag, finishedgoodsflag, color, safetystocklevel, reorderpoint, standardcost, listprice, size, sizeunitmeasurecode, weightunitmeasurecode, weight, daystomanufacture, productline, class, style, productsubcategoryid, productmodelid, sellstartdate, sellenddate, discontinueddate, rowguid, modifieddate FROM production.product;

CREATE VIEW "pr"."pc" AS
SELECT productcategoryid AS id, productcategoryid, name, rowguid, modifieddate FROM production.productcategory;

CREATE VIEW "pr"."pch" AS
SELECT productid AS id, productid, startdate, enddate, standardcost, modifieddate FROM production.productcosthistory;

CREATE VIEW "pr"."pd" AS
SELECT productdescriptionid AS id, productdescriptionid, description, rowguid, modifieddate FROM production.productdescription;

CREATE VIEW "pr"."pdoc" AS
SELECT productid AS id, productid, modifieddate, documentnode FROM production.productdocument;

CREATE VIEW "pr"."pi" AS
SELECT productid AS id, productid, locationid, shelf, bin, quantity, rowguid, modifieddate FROM production.productinventory;

CREATE VIEW "pr"."plph" AS
SELECT productid AS id, productid, startdate, enddate, listprice, modifieddate FROM production.productlistpricehistory;

CREATE VIEW "pr"."pm" AS
SELECT productmodelid AS id, productmodelid, name, catalogdescription, instructions, rowguid, modifieddate FROM production.productmodel;

CREATE VIEW "pr"."pmi" AS
SELECT productmodelid, illustrationid, modifieddate FROM production.productmodelillustration;

CREATE VIEW "pr"."pmpdc" AS
SELECT productmodelid, productdescriptionid, cultureid, modifieddate FROM production.productmodelproductdescriptionculture;

CREATE VIEW "pr"."pp" AS
SELECT productphotoid AS id, productphotoid, thumbnailphoto, thumbnailphotofilename, largephoto, largephotofilename, modifieddate FROM production.productphoto;

CREATE VIEW "pr"."ppp" AS
SELECT productid, productphotoid, "primary", modifieddate FROM production.productproductphoto;

CREATE VIEW "pr"."pr" AS
SELECT productreviewid AS id, productreviewid, productid, reviewername, reviewdate, emailaddress, rating, comments, modifieddate FROM production.productreview;

CREATE VIEW "pr"."psc" AS
SELECT productsubcategoryid AS id, productsubcategoryid, productcategoryid, name, rowguid, modifieddate FROM production.productsubcategory;

CREATE VIEW "pr"."sr" AS
SELECT scrapreasonid AS id, scrapreasonid, name, modifieddate FROM production.scrapreason;

CREATE VIEW "pr"."th" AS
SELECT transactionid AS id, transactionid, productid, referenceorderid, referenceorderlineid, transactiondate, transactiontype, quantity, actualcost, modifieddate FROM production.transactionhistory;

CREATE VIEW "pr"."tha" AS
SELECT transactionid AS id, transactionid, productid, referenceorderid, referenceorderlineid, transactiondate, transactiontype, quantity, actualcost, modifieddate FROM production.transactionhistoryarchive;

CREATE VIEW "pr"."um" AS
SELECT unitmeasurecode AS id, unitmeasurecode, name, modifieddate FROM production.unitmeasure;

CREATE VIEW "pr"."w" AS
SELECT workorderid AS id, workorderid, productid, orderqty, scrappedqty, startdate, enddate, duedate, scrapreasonid, modifieddate FROM production.workorder;

CREATE VIEW "pr"."wr" AS
SELECT workorderid AS id, workorderid, productid, operationsequence, locationid, scheduledstartdate, scheduledenddate, actualstartdate, actualenddate, actualresourcehrs, plannedcost, actualcost, modifieddate FROM production.workorderrouting;

CREATE VIEW "production"."vproductmodelcatalogdescription" AS
SELECT productmodelid, name, (xpath('/p1:ProductDescription/p1:Summary/html:p/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{html,http://www.w3.org/1999/xhtml}}'::text[]))[1]::varchar AS "Summary", (xpath('/p1:ProductDescription/p1:Manufacturer/p1:Name/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar AS manufacturer, (xpath('/p1:ProductDescription/p1:Manufacturer/p1:Copyright/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(30) AS copyright, (xpath('/p1:ProductDescription/p1:Manufacturer/p1:ProductURL/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS producturl, (xpath('/p1:ProductDescription/p1:Features/wm:Warranty/wm:WarrantyPeriod/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wm,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelWarrAndMain}}'::text[]))[1]::varchar(256) AS warrantyperiod, (xpath('/p1:ProductDescription/p1:Features/wm:Warranty/wm:Description/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wm,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelWarrAndMain}}'::text[]))[1]::varchar(256) AS warrantydescription, (xpath('/p1:ProductDescription/p1:Features/wm:Maintenance/wm:NoOfYears/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wm,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelWarrAndMain}}'::text[]))[1]::varchar(256) AS noofyears, (xpath('/p1:ProductDescription/p1:Features/wm:Maintenance/wm:Description/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wm,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelWarrAndMain}}'::text[]))[1]::varchar(256) AS maintenancedescription, (xpath('/p1:ProductDescription/p1:Features/wf:wheel/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wf,http://www.adventure-works.com/schemas/OtherFeatures}}'::text[]))[1]::varchar(256) AS wheel, (xpath('/p1:ProductDescription/p1:Features/wf:saddle/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wf,http://www.adventure-works.com/schemas/OtherFeatures}}'::text[]))[1]::varchar(256) AS saddle, (xpath('/p1:ProductDescription/p1:Features/wf:pedal/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wf,http://www.adventure-works.com/schemas/OtherFeatures}}'::text[]))[1]::varchar(256) AS pedal, (xpath('/p1:ProductDescription/p1:Features/wf:BikeFrame/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wf,http://www.adventure-works.com/schemas/OtherFeatures}}'::text[]))[1]::varchar AS bikeframe, (xpath('/p1:ProductDescription/p1:Features/wf:crankset/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription},{wf,http://www.adventure-works.com/schemas/OtherFeatures}}'::text[]))[1]::varchar(256) AS crankset, (xpath('/p1:ProductDescription/p1:Picture/p1:Angle/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS pictureangle, (xpath('/p1:ProductDescription/p1:Picture/p1:Size/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS picturesize, (xpath('/p1:ProductDescription/p1:Picture/p1:ProductPhotoID/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS productphotoid, (xpath('/p1:ProductDescription/p1:Specifications/Material/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS material, (xpath('/p1:ProductDescription/p1:Specifications/Color/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS color, (xpath('/p1:ProductDescription/p1:Specifications/ProductLine/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS productline, (xpath('/p1:ProductDescription/p1:Specifications/Style/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(256) AS style, (xpath('/p1:ProductDescription/p1:Specifications/RiderExperience/text()'::text, catalogdescription, '{{p1,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelDescription}}'::text[]))[1]::varchar(1024) AS riderexperience, rowguid, modifieddate FROM production.productmodel WHERE catalogdescription IS NOT NULL;

CREATE VIEW "production"."vproductmodelinstructions" AS
SELECT productmodelid, name, (xpath('/ns:root/text()'::text, instructions, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelManuInstructions}}'::text[]))[1]::varchar AS instructions, (xpath('@LocationID'::text, mfginstructions))[1]::varchar::int AS "LocationID", (xpath('@SetupHours'::text, mfginstructions))[1]::varchar::numeric(9, 4) AS "SetupHours", (xpath('@MachineHours'::text, mfginstructions))[1]::varchar::numeric(9, 4) AS "MachineHours", (xpath('@LaborHours'::text, mfginstructions))[1]::varchar::numeric(9, 4) AS "LaborHours", (xpath('@LotSize'::text, mfginstructions))[1]::varchar::int AS "LotSize", (xpath('/step/text()'::text, step))[1]::varchar(1024) AS "Step", rowguid, modifieddate FROM (SELECT locations.productmodelid, locations.name, locations.rowguid, locations.modifieddate, locations.instructions, locations.mfginstructions, unnest(xpath('step'::text, locations.mfginstructions)) AS step FROM (SELECT productmodel.productmodelid, productmodel.name, productmodel.rowguid, productmodel.modifieddate, productmodel.instructions, unnest(xpath('/ns:root/ns:Location'::text, productmodel.instructions, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/ProductModelManuInstructions}}'::text[])) AS mfginstructions FROM production.productmodel) locations) pm;

CREATE VIEW "pu"."pod" AS
SELECT purchaseorderdetailid AS id, purchaseorderid, purchaseorderdetailid, duedate, orderqty, productid, unitprice, receivedqty, rejectedqty, modifieddate FROM purchasing.purchaseorderdetail;

CREATE VIEW "pu"."poh" AS
SELECT purchaseorderid AS id, purchaseorderid, revisionnumber, status, employeeid, vendorid, shipmethodid, orderdate, shipdate, subtotal, taxamt, freight, modifieddate FROM purchasing.purchaseorderheader;

CREATE VIEW "pu"."pv" AS
SELECT productid AS id, productid, businessentityid, averageleadtime, standardprice, lastreceiptcost, lastreceiptdate, minorderqty, maxorderqty, onorderqty, unitmeasurecode, modifieddate FROM purchasing.productvendor;

CREATE VIEW "pu"."sm" AS
SELECT shipmethodid AS id, shipmethodid, name, shipbase, shiprate, rowguid, modifieddate FROM purchasing.shipmethod;

CREATE VIEW "pu"."v" AS
SELECT businessentityid AS id, businessentityid, accountnumber, name, creditrating, preferredvendorstatus, activeflag, purchasingwebserviceurl, modifieddate FROM purchasing.vendor;

CREATE VIEW "purchasing"."vvendorwithaddresses" AS
SELECT v.businessentityid, v.name, at.name AS addresstype, a.addressline1, a.addressline2, a.city, sp.name AS stateprovincename, a.postalcode, cr.name AS countryregionname FROM purchasing.vendor v JOIN person.businessentityaddress bea ON bea.businessentityid = v.businessentityid JOIN person.address a ON a.addressid = bea.addressid JOIN person.stateprovince sp ON sp.stateprovinceid = a.stateprovinceid JOIN person.countryregion cr ON cr.countryregioncode::text = sp.countryregioncode::text JOIN person.addresstype at ON at.addresstypeid = bea.addresstypeid;

CREATE VIEW "purchasing"."vvendorwithcontacts" AS
SELECT v.businessentityid, v.name, ct.name AS contacttype, p.title, p.firstname, p.middlename, p.lastname, p.suffix, pp.phonenumber, pnt.name AS phonenumbertype, ea.emailaddress, p.emailpromotion FROM purchasing.vendor v JOIN person.businessentitycontact bec ON bec.businessentityid = v.businessentityid JOIN person.contacttype ct ON ct.contacttypeid = bec.contacttypeid JOIN person.person p ON p.businessentityid = bec.personid LEFT JOIN person.emailaddress ea ON ea.businessentityid = p.businessentityid LEFT JOIN person.personphone pp ON pp.businessentityid = p.businessentityid LEFT JOIN person.phonenumbertype pnt ON pnt.phonenumbertypeid = pp.phonenumbertypeid;

CREATE VIEW "sa"."c" AS
SELECT customerid AS id, customerid, personid, storeid, territoryid, rowguid, modifieddate FROM sales.customer;

CREATE VIEW "sa"."cc" AS
SELECT creditcardid AS id, creditcardid, cardtype, cardnumber, expmonth, expyear, modifieddate FROM sales.creditcard;

CREATE VIEW "sa"."cr" AS
SELECT currencyrateid, currencyratedate, fromcurrencycode, tocurrencycode, averagerate, endofdayrate, modifieddate FROM sales.currencyrate;

CREATE VIEW "sa"."crc" AS
SELECT countryregioncode, currencycode, modifieddate FROM sales.countryregioncurrency;

CREATE VIEW "sa"."cu" AS
SELECT currencycode AS id, currencycode, name, modifieddate FROM sales.currency;

CREATE VIEW "sa"."pcc" AS
SELECT businessentityid AS id, businessentityid, creditcardid, modifieddate FROM sales.personcreditcard;

CREATE VIEW "sa"."s" AS
SELECT businessentityid AS id, businessentityid, name, salespersonid, demographics, rowguid, modifieddate FROM sales.store;

CREATE VIEW "sa"."sci" AS
SELECT shoppingcartitemid AS id, shoppingcartitemid, shoppingcartid, quantity, productid, datecreated, modifieddate FROM sales.shoppingcartitem;

CREATE VIEW "sa"."so" AS
SELECT specialofferid AS id, specialofferid, description, discountpct, type, category, startdate, enddate, minqty, maxqty, rowguid, modifieddate FROM sales.specialoffer;

CREATE VIEW "sa"."sod" AS
SELECT salesorderdetailid AS id, salesorderid, salesorderdetailid, carriertrackingnumber, orderqty, productid, specialofferid, unitprice, unitpricediscount, rowguid, modifieddate FROM sales.salesorderdetail;

CREATE VIEW "sa"."soh" AS
SELECT salesorderid AS id, salesorderid, revisionnumber, orderdate, duedate, shipdate, status, onlineorderflag, purchaseordernumber, accountnumber, customerid, salespersonid, territoryid, billtoaddressid, shiptoaddressid, shipmethodid, creditcardid, creditcardapprovalcode, currencyrateid, subtotal, taxamt, freight, totaldue, comment, rowguid, modifieddate FROM sales.salesorderheader;

CREATE VIEW "sa"."sohsr" AS
SELECT salesorderid, salesreasonid, modifieddate FROM sales.salesorderheadersalesreason;

CREATE VIEW "sa"."sop" AS
SELECT specialofferid AS id, specialofferid, productid, rowguid, modifieddate FROM sales.specialofferproduct;

CREATE VIEW "sa"."sp" AS
SELECT businessentityid AS id, businessentityid, territoryid, salesquota, bonus, commissionpct, salesytd, saleslastyear, rowguid, modifieddate FROM sales.salesperson;

CREATE VIEW "sa"."spqh" AS
SELECT businessentityid AS id, businessentityid, quotadate, salesquota, rowguid, modifieddate FROM sales.salespersonquotahistory;

CREATE VIEW "sa"."sr" AS
SELECT salesreasonid AS id, salesreasonid, name, reasontype, modifieddate FROM sales.salesreason;

CREATE VIEW "sa"."st" AS
SELECT territoryid AS id, territoryid, name, countryregioncode, "group", salesytd, saleslastyear, costytd, costlastyear, rowguid, modifieddate FROM sales.salesterritory;

CREATE VIEW "sa"."sth" AS
SELECT territoryid AS id, businessentityid, territoryid, startdate, enddate, rowguid, modifieddate FROM sales.salesterritoryhistory;

CREATE VIEW "sa"."tr" AS
SELECT salestaxrateid AS id, salestaxrateid, stateprovinceid, taxtype, taxrate, name, rowguid, modifieddate FROM sales.salestaxrate;

CREATE VIEW "sales"."vindividualcustomer" AS
SELECT p.businessentityid, p.title, p.firstname, p.middlename, p.lastname, p.suffix, pp.phonenumber, pnt.name AS phonenumbertype, ea.emailaddress, p.emailpromotion, at.name AS addresstype, a.addressline1, a.addressline2, a.city, sp.name AS stateprovincename, a.postalcode, cr.name AS countryregionname, p.demographics FROM person.person p JOIN person.businessentityaddress bea ON bea.businessentityid = p.businessentityid JOIN person.address a ON a.addressid = bea.addressid JOIN person.stateprovince sp ON sp.stateprovinceid = a.stateprovinceid JOIN person.countryregion cr ON cr.countryregioncode::text = sp.countryregioncode::text JOIN person.addresstype at ON at.addresstypeid = bea.addresstypeid JOIN sales.customer c ON c.personid = p.businessentityid LEFT JOIN person.emailaddress ea ON ea.businessentityid = p.businessentityid LEFT JOIN person.personphone pp ON pp.businessentityid = p.businessentityid LEFT JOIN person.phonenumbertype pnt ON pnt.phonenumbertypeid = pp.phonenumbertypeid WHERE c.storeid IS NULL;

CREATE VIEW "sales"."vpersondemographics" AS
SELECT businessentityid, (xpath('n:TotalPurchaseYTD/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::money AS totalpurchaseytd, (xpath('n:DateFirstPurchase/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::date AS datefirstpurchase, (xpath('n:BirthDate/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::date AS birthdate, (xpath('n:MaritalStatus/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar(1) AS maritalstatus, (xpath('n:YearlyIncome/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar(30) AS yearlyincome, (xpath('n:Gender/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar(1) AS gender, (xpath('n:TotalChildren/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::int AS totalchildren, (xpath('n:NumberChildrenAtHome/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::int AS numberchildrenathome, (xpath('n:Education/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar(30) AS education, (xpath('n:Occupation/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar(30) AS occupation, (xpath('n:HomeOwnerFlag/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::boolean AS homeownerflag, (xpath('n:NumberCarsOwned/text()'::text, demographics, '{{n,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/IndividualSurvey}}'::text[]))[1]::varchar::int AS numbercarsowned FROM person.person WHERE demographics IS NOT NULL;

CREATE VIEW "sales"."vsalesperson" AS
SELECT s.businessentityid, p.title, p.firstname, p.middlename, p.lastname, p.suffix, e.jobtitle, pp.phonenumber, pnt.name AS phonenumbertype, ea.emailaddress, p.emailpromotion, a.addressline1, a.addressline2, a.city, sp.name AS stateprovincename, a.postalcode, cr.name AS countryregionname, st.name AS territoryname, st."group" AS territorygroup, s.salesquota, s.salesytd, s.saleslastyear FROM sales.salesperson s JOIN humanresources.employee e ON e.businessentityid = s.businessentityid JOIN person.person p ON p.businessentityid = s.businessentityid JOIN person.businessentityaddress bea ON bea.businessentityid = s.businessentityid JOIN person.address a ON a.addressid = bea.addressid JOIN person.stateprovince sp ON sp.stateprovinceid = a.stateprovinceid JOIN person.countryregion cr ON cr.countryregioncode::text = sp.countryregioncode::text LEFT JOIN sales.salesterritory st ON st.territoryid = s.territoryid LEFT JOIN person.emailaddress ea ON ea.businessentityid = p.businessentityid LEFT JOIN person.personphone pp ON pp.businessentityid = p.businessentityid LEFT JOIN person.phonenumbertype pnt ON pnt.phonenumbertypeid = pp.phonenumbertypeid;

CREATE VIEW "sales"."vsalespersonsalesbyfiscalyears" AS
SELECT "SalesPersonID", "FullName", "JobTitle", "SalesTerritory", "2012", "2013", "2014" FROM public.crosstab('SELECT
    SalesPersonID
    ,FullName
    ,JobTitle
    ,SalesTerritory
    ,FiscalYear
    ,SalesTotal
FROM Sales.vSalesPersonSalesByFiscalYearsData
ORDER BY 2,4'::text, 'SELECT unnest(''{2012,2013,2014}''::text[])'::text) salestotal ("SalesPersonID" int, "FullName" text, "JobTitle" text, "SalesTerritory" text, "2012" numeric(12, 4), "2013" numeric(12, 4), "2014" numeric(12, 4));

CREATE VIEW "sales"."vsalespersonsalesbyfiscalyearsdata" AS
SELECT salespersonid, fullname, jobtitle, salesterritory, sum(subtotal) AS salestotal, fiscalyear FROM (SELECT soh.salespersonid, ((p.firstname::text || ' '::text) || COALESCE(p.middlename::text || ' '::text, ''::text)) || p.lastname::text AS fullname, e.jobtitle, st.name AS salesterritory, soh.subtotal, extract ('year' FROM soh.orderdate + '6 mons'::interval) AS fiscalyear FROM sales.salesperson sp JOIN sales.salesorderheader soh ON sp.businessentityid = soh.salespersonid JOIN sales.salesterritory st ON sp.territoryid = st.territoryid JOIN humanresources.employee e ON soh.salespersonid = e.businessentityid JOIN person.person p ON p.businessentityid = sp.businessentityid) granular GROUP BY salespersonid, fullname, jobtitle, salesterritory, fiscalyear;

CREATE VIEW "sales"."vstorewithaddresses" AS
SELECT s.businessentityid, s.name, at.name AS addresstype, a.addressline1, a.addressline2, a.city, sp.name AS stateprovincename, a.postalcode, cr.name AS countryregionname FROM sales.store s JOIN person.businessentityaddress bea ON bea.businessentityid = s.businessentityid JOIN person.address a ON a.addressid = bea.addressid JOIN person.stateprovince sp ON sp.stateprovinceid = a.stateprovinceid JOIN person.countryregion cr ON cr.countryregioncode::text = sp.countryregioncode::text JOIN person.addresstype at ON at.addresstypeid = bea.addresstypeid;

CREATE VIEW "sales"."vstorewithcontacts" AS
SELECT s.businessentityid, s.name, ct.name AS contacttype, p.title, p.firstname, p.middlename, p.lastname, p.suffix, pp.phonenumber, pnt.name AS phonenumbertype, ea.emailaddress, p.emailpromotion FROM sales.store s JOIN person.businessentitycontact bec ON bec.businessentityid = s.businessentityid JOIN person.contacttype ct ON ct.contacttypeid = bec.contacttypeid JOIN person.person p ON p.businessentityid = bec.personid LEFT JOIN person.emailaddress ea ON ea.businessentityid = p.businessentityid LEFT JOIN person.personphone pp ON pp.businessentityid = p.businessentityid LEFT JOIN person.phonenumbertype pnt ON pnt.phonenumbertypeid = pp.phonenumbertypeid;

CREATE VIEW "sales"."vstorewithdemographics" AS
SELECT businessentityid, name, unnest(xpath('/ns:StoreSurvey/ns:AnnualSales/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar::money AS "AnnualSales", unnest(xpath('/ns:StoreSurvey/ns:AnnualRevenue/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar::money AS "AnnualRevenue", unnest(xpath('/ns:StoreSurvey/ns:BankName/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar(50) AS "BankName", unnest(xpath('/ns:StoreSurvey/ns:BusinessType/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar(5) AS "BusinessType", unnest(xpath('/ns:StoreSurvey/ns:YearOpened/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar::int AS "YearOpened", unnest(xpath('/ns:StoreSurvey/ns:Specialty/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar(50) AS "Specialty", unnest(xpath('/ns:StoreSurvey/ns:SquareFeet/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar::int AS "SquareFeet", unnest(xpath('/ns:StoreSurvey/ns:Brands/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar(30) AS "Brands", unnest(xpath('/ns:StoreSurvey/ns:Internet/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar(30) AS "Internet", unnest(xpath('/ns:StoreSurvey/ns:NumberEmployees/text()'::text, demographics, '{{ns,http://schemas.microsoft.com/sqlserver/2004/07/adventure-works/StoreSurvey}}'::text[]))::varchar::int AS "NumberEmployees" FROM sales.store;

CREATE UNIQUE INDEX "ix_vstateprovincecountryregion" ON "person"."vstateprovincecountryregion" (
	"stateprovinceid",
	"countryregioncode"
);

CREATE UNIQUE INDEX "ix_vproductanddescription" ON "production"."vproductanddescription" (
	"cultureid",
	"productid"
);

COMMENT ON SCHEMA "humanresources" IS 'Contains objects related to employees and departments.';
COMMENT ON SCHEMA "person" IS 'Contains objects related to names and addresses of customers, vendors, and employees';
COMMENT ON SCHEMA "production" IS 'Contains objects related to products, inventory, and manufacturing.';
COMMENT ON SCHEMA "purchasing" IS 'Contains objects related to vendors and purchase orders.';
COMMENT ON SCHEMA "sales" IS 'Contains objects related to customers, sales orders, and sales territories.';
COMMENT ON TABLE "humanresources"."department" IS 'Lookup table containing the departments within the Adventure Works Cycles company.';
COMMENT ON COLUMN "humanresources"."department"."departmentid" IS 'Primary key for Department records.';
COMMENT ON COLUMN "humanresources"."department"."name" IS 'Name of the department.';
COMMENT ON COLUMN "humanresources"."department"."groupname" IS 'Name of the group to which the department belongs.';
COMMENT ON TABLE "humanresources"."employee" IS 'Employee information such as salary, department, and title.';
COMMENT ON COLUMN "humanresources"."employee"."businessentityid" IS 'Primary key for Employee records.  Foreign key to BusinessEntity.BusinessEntityID.';
COMMENT ON COLUMN "humanresources"."employee"."nationalidnumber" IS 'Unique national identification number such as a social security number.';
COMMENT ON COLUMN "humanresources"."employee"."loginid" IS 'Network login.';
COMMENT ON COLUMN "humanresources"."employee"."jobtitle" IS 'Work title such as Buyer or Sales Representative.';
COMMENT ON COLUMN "humanresources"."employee"."birthdate" IS 'Date of birth.';
COMMENT ON COLUMN "humanresources"."employee"."maritalstatus" IS 'M = Married, S = Single';
COMMENT ON COLUMN "humanresources"."employee"."gender" IS 'M = Male, F = Female';
COMMENT ON COLUMN "humanresources"."employee"."hiredate" IS 'Employee hired on this date.';
COMMENT ON COLUMN "humanresources"."employee"."salariedflag" IS 'Job classification. 0 = Hourly, not exempt from collective bargaining. 1 = Salaried, exempt from collective bargaining.';
COMMENT ON COLUMN "humanresources"."employee"."vacationhours" IS 'Number of available vacation hours.';
COMMENT ON COLUMN "humanresources"."employee"."sickleavehours" IS 'Number of available sick leave hours.';
COMMENT ON COLUMN "humanresources"."employee"."currentflag" IS '0 = Inactive, 1 = Active';
COMMENT ON COLUMN "humanresources"."employee"."organizationnode" IS 'Where the employee is located in corporate hierarchy.';
COMMENT ON TABLE "humanresources"."employeedepartmenthistory" IS 'Employee department transfers.';
COMMENT ON COLUMN "humanresources"."employeedepartmenthistory"."businessentityid" IS 'Employee identification number. Foreign key to Employee.BusinessEntityID.';
COMMENT ON COLUMN "humanresources"."employeedepartmenthistory"."departmentid" IS 'Department in which the employee worked including currently. Foreign key to Department.DepartmentID.';
COMMENT ON COLUMN "humanresources"."employeedepartmenthistory"."shiftid" IS 'Identifies which 8-hour shift the employee works. Foreign key to Shift.Shift.ID.';
COMMENT ON COLUMN "humanresources"."employeedepartmenthistory"."startdate" IS 'Date the employee started work in the department.';
COMMENT ON COLUMN "humanresources"."employeedepartmenthistory"."enddate" IS 'Date the employee left the department. NULL = Current department.';
COMMENT ON TABLE "humanresources"."employeepayhistory" IS 'Employee pay history.';
COMMENT ON COLUMN "humanresources"."employeepayhistory"."businessentityid" IS 'Employee identification number. Foreign key to Employee.BusinessEntityID.';
COMMENT ON COLUMN "humanresources"."employeepayhistory"."ratechangedate" IS 'Date the change in pay is effective';
COMMENT ON COLUMN "humanresources"."employeepayhistory"."rate" IS 'Salary hourly rate.';
COMMENT ON COLUMN "humanresources"."employeepayhistory"."payfrequency" IS '1 = Salary received monthly, 2 = Salary received biweekly';
COMMENT ON TABLE "humanresources"."jobcandidate" IS 'RÃ©sumÃ©s submitted to Human Resources by job applicants.';
COMMENT ON COLUMN "humanresources"."jobcandidate"."jobcandidateid" IS 'Primary key for JobCandidate records.';
COMMENT ON COLUMN "humanresources"."jobcandidate"."businessentityid" IS 'Employee identification number if applicant was hired. Foreign key to Employee.BusinessEntityID.';
COMMENT ON COLUMN "humanresources"."jobcandidate"."resume" IS 'RÃ©sumÃ© in XML format.';
COMMENT ON TABLE "humanresources"."shift" IS 'Work shift lookup table.';
COMMENT ON COLUMN "humanresources"."shift"."shiftid" IS 'Primary key for Shift records.';
COMMENT ON COLUMN "humanresources"."shift"."name" IS 'Shift description.';
COMMENT ON COLUMN "humanresources"."shift"."starttime" IS 'Shift start time.';
COMMENT ON COLUMN "humanresources"."shift"."endtime" IS 'Shift end time.';
COMMENT ON TABLE "person"."address" IS 'Street address information for customers, employees, and vendors.';
COMMENT ON COLUMN "person"."address"."addressid" IS 'Primary key for Address records.';
COMMENT ON COLUMN "person"."address"."addressline1" IS 'First street address line.';
COMMENT ON COLUMN "person"."address"."addressline2" IS 'Second street address line.';
COMMENT ON COLUMN "person"."address"."city" IS 'Name of the city.';
COMMENT ON COLUMN "person"."address"."stateprovinceid" IS 'Unique identification number for the state or province. Foreign key to StateProvince table.';
COMMENT ON COLUMN "person"."address"."postalcode" IS 'Postal code for the street address.';
COMMENT ON COLUMN "person"."address"."spatiallocation" IS 'Latitude and longitude of this address.';
COMMENT ON TABLE "person"."businessentityaddress" IS 'Cross-reference table mapping customers, vendors, and employees to their addresses.';
COMMENT ON COLUMN "person"."businessentityaddress"."businessentityid" IS 'Primary key. Foreign key to BusinessEntity.BusinessEntityID.';
COMMENT ON COLUMN "person"."businessentityaddress"."addressid" IS 'Primary key. Foreign key to Address.AddressID.';
COMMENT ON COLUMN "person"."businessentityaddress"."addresstypeid" IS 'Primary key. Foreign key to AddressType.AddressTypeID.';
COMMENT ON TABLE "person"."countryregion" IS 'Lookup table containing the ISO standard codes for countries and regions.';
COMMENT ON COLUMN "person"."countryregion"."countryregioncode" IS 'ISO standard code for countries and regions.';
COMMENT ON COLUMN "person"."countryregion"."name" IS 'Country or region name.';
COMMENT ON TABLE "person"."emailaddress" IS 'Where to send a person email.';
COMMENT ON COLUMN "person"."emailaddress"."businessentityid" IS 'Primary key. Person associated with this email address.  Foreign key to Person.BusinessEntityID';
COMMENT ON COLUMN "person"."emailaddress"."emailaddressid" IS 'Primary key. ID of this email address.';
COMMENT ON COLUMN "person"."emailaddress"."emailaddress" IS 'E-mail address for the person.';
COMMENT ON TABLE "person"."person" IS 'Human beings involved with AdventureWorks: employees, customer contacts, and vendor contacts.';
COMMENT ON COLUMN "person"."person"."businessentityid" IS 'Primary key for Person records.';
COMMENT ON COLUMN "person"."person"."persontype" IS 'Primary type of person: SC = Store Contact, IN = Individual (retail) customer, SP = Sales person, EM = Employee (non-sales), VC = Vendor contact, GC = General contact';
COMMENT ON COLUMN "person"."person"."namestyle" IS '0 = The data in FirstName and LastName are stored in western style (first name, last name) order.  1 = Eastern style (last name, first name) order.';
COMMENT ON COLUMN "person"."person"."title" IS 'A courtesy title. For example, Mr. or Ms.';
COMMENT ON COLUMN "person"."person"."firstname" IS 'First name of the person.';
COMMENT ON COLUMN "person"."person"."middlename" IS 'Middle name or middle initial of the person.';
COMMENT ON COLUMN "person"."person"."lastname" IS 'Last name of the person.';
COMMENT ON COLUMN "person"."person"."suffix" IS 'Surname suffix. For example, Sr. or Jr.';
COMMENT ON COLUMN "person"."person"."emailpromotion" IS '0 = Contact does not wish to receive e-mail promotions, 1 = Contact does wish to receive e-mail promotions from AdventureWorks, 2 = Contact does wish to receive e-mail promotions from AdventureWorks and selected partners.';
COMMENT ON COLUMN "person"."person"."additionalcontactinfo" IS 'Additional contact information about the person stored in xml format.';
COMMENT ON COLUMN "person"."person"."demographics" IS 'Personal information such as hobbies, and income collected from online shoppers. Used for sales analysis.';
COMMENT ON TABLE "person"."personphone" IS 'Telephone number and type of a person.';
COMMENT ON COLUMN "person"."personphone"."businessentityid" IS 'Business entity identification number. Foreign key to Person.BusinessEntityID.';
COMMENT ON COLUMN "person"."personphone"."phonenumber" IS 'Telephone number identification number.';
COMMENT ON COLUMN "person"."personphone"."phonenumbertypeid" IS 'Kind of phone number. Foreign key to PhoneNumberType.PhoneNumberTypeID.';
COMMENT ON TABLE "person"."phonenumbertype" IS 'Type of phone number of a person.';
COMMENT ON COLUMN "person"."phonenumbertype"."phonenumbertypeid" IS 'Primary key for telephone number type records.';
COMMENT ON COLUMN "person"."phonenumbertype"."name" IS 'Name of the telephone number type';
COMMENT ON TABLE "person"."stateprovince" IS 'State and province lookup table.';
COMMENT ON COLUMN "person"."stateprovince"."stateprovinceid" IS 'Primary key for StateProvince records.';
COMMENT ON COLUMN "person"."stateprovince"."stateprovincecode" IS 'ISO standard state or province code.';
COMMENT ON COLUMN "person"."stateprovince"."countryregioncode" IS 'ISO standard country or region code. Foreign key to CountryRegion.CountryRegionCode.';
COMMENT ON COLUMN "person"."stateprovince"."isonlystateprovinceflag" IS '0 = StateProvinceCode exists. 1 = StateProvinceCode unavailable, using CountryRegionCode.';
COMMENT ON COLUMN "person"."stateprovince"."name" IS 'State or province description.';
COMMENT ON COLUMN "person"."stateprovince"."territoryid" IS 'ID of the territory in which the state or province is located. Foreign key to SalesTerritory.SalesTerritoryID.';
COMMENT ON TABLE "person"."addresstype" IS 'Types of addresses stored in the Address table.';
COMMENT ON COLUMN "person"."addresstype"."addresstypeid" IS 'Primary key for AddressType records.';
COMMENT ON COLUMN "person"."addresstype"."name" IS 'Address type description. For example, Billing, Home, or Shipping.';
COMMENT ON TABLE "person"."businessentity" IS 'Source of the ID that connects vendors, customers, and employees with address and contact information.';
COMMENT ON COLUMN "person"."businessentity"."businessentityid" IS 'Primary key for all customers, vendors, and employees.';
COMMENT ON TABLE "person"."businessentitycontact" IS 'Cross-reference table mapping stores, vendors, and employees to people';
COMMENT ON COLUMN "person"."businessentitycontact"."businessentityid" IS 'Primary key. Foreign key to BusinessEntity.BusinessEntityID.';
COMMENT ON COLUMN "person"."businessentitycontact"."personid" IS 'Primary key. Foreign key to Person.BusinessEntityID.';
COMMENT ON COLUMN "person"."businessentitycontact"."contacttypeid" IS 'Primary key.  Foreign key to ContactType.ContactTypeID.';
COMMENT ON TABLE "person"."contacttype" IS 'Lookup table containing the types of business entity contacts.';
COMMENT ON COLUMN "person"."contacttype"."contacttypeid" IS 'Primary key for ContactType records.';
COMMENT ON COLUMN "person"."contacttype"."name" IS 'Contact type description.';
COMMENT ON TABLE "person"."password" IS 'One way hashed authentication information';
COMMENT ON COLUMN "person"."password"."passwordhash" IS 'Password for the e-mail account.';
COMMENT ON COLUMN "person"."password"."passwordsalt" IS 'Random value concatenated with the password string before the password is hashed.';
COMMENT ON TABLE "production"."billofmaterials" IS 'Items required to make bicycles and bicycle subassemblies. It identifies the heirarchical relationship between a parent product and its components.';
COMMENT ON COLUMN "production"."billofmaterials"."billofmaterialsid" IS 'Primary key for BillOfMaterials records.';
COMMENT ON COLUMN "production"."billofmaterials"."productassemblyid" IS 'Parent product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."billofmaterials"."componentid" IS 'Component identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."billofmaterials"."startdate" IS 'Date the component started being used in the assembly item.';
COMMENT ON COLUMN "production"."billofmaterials"."enddate" IS 'Date the component stopped being used in the assembly item.';
COMMENT ON COLUMN "production"."billofmaterials"."unitmeasurecode" IS 'Standard code identifying the unit of measure for the quantity.';
COMMENT ON COLUMN "production"."billofmaterials"."bomlevel" IS 'Indicates the depth the component is from its parent (AssemblyID).';
COMMENT ON COLUMN "production"."billofmaterials"."perassemblyqty" IS 'Quantity of the component needed to create the assembly.';
COMMENT ON TABLE "production"."culture" IS 'Lookup table containing the languages in which some AdventureWorks data is stored.';
COMMENT ON COLUMN "production"."culture"."cultureid" IS 'Primary key for Culture records.';
COMMENT ON COLUMN "production"."culture"."name" IS 'Culture description.';
COMMENT ON TABLE "production"."document" IS 'Product maintenance documents.';
COMMENT ON COLUMN "production"."document"."title" IS 'Title of the document.';
COMMENT ON COLUMN "production"."document"."owner" IS 'Employee who controls the document.  Foreign key to Employee.BusinessEntityID';
COMMENT ON COLUMN "production"."document"."folderflag" IS '0 = This is a folder, 1 = This is a document.';
COMMENT ON COLUMN "production"."document"."filename" IS 'File name of the document';
COMMENT ON COLUMN "production"."document"."fileextension" IS 'File extension indicating the document type. For example, .doc or .txt.';
COMMENT ON COLUMN "production"."document"."revision" IS 'Revision number of the document.';
COMMENT ON COLUMN "production"."document"."changenumber" IS 'Engineering change approval number.';
COMMENT ON COLUMN "production"."document"."status" IS '1 = Pending approval, 2 = Approved, 3 = Obsolete';
COMMENT ON COLUMN "production"."document"."documentsummary" IS 'Document abstract.';
COMMENT ON COLUMN "production"."document"."document" IS 'Complete document.';
COMMENT ON COLUMN "production"."document"."rowguid" IS 'ROWGUIDCOL number uniquely identifying the record. Required for FileStream.';
COMMENT ON COLUMN "production"."document"."documentnode" IS 'Primary key for Document records.';
COMMENT ON TABLE "production"."illustration" IS 'Bicycle assembly diagrams.';
COMMENT ON COLUMN "production"."illustration"."illustrationid" IS 'Primary key for Illustration records.';
COMMENT ON COLUMN "production"."illustration"."diagram" IS 'Illustrations used in manufacturing instructions. Stored as XML.';
COMMENT ON TABLE "production"."location" IS 'Product inventory and manufacturing locations.';
COMMENT ON COLUMN "production"."location"."locationid" IS 'Primary key for Location records.';
COMMENT ON COLUMN "production"."location"."name" IS 'Location description.';
COMMENT ON COLUMN "production"."location"."costrate" IS 'Standard hourly cost of the manufacturing location.';
COMMENT ON COLUMN "production"."location"."availability" IS 'Work capacity (in hours) of the manufacturing location.';
COMMENT ON TABLE "production"."product" IS 'Products sold or used in the manfacturing of sold products.';
COMMENT ON COLUMN "production"."product"."productid" IS 'Primary key for Product records.';
COMMENT ON COLUMN "production"."product"."name" IS 'Name of the product.';
COMMENT ON COLUMN "production"."product"."productnumber" IS 'Unique product identification number.';
COMMENT ON COLUMN "production"."product"."makeflag" IS '0 = Product is purchased, 1 = Product is manufactured in-house.';
COMMENT ON COLUMN "production"."product"."finishedgoodsflag" IS '0 = Product is not a salable item. 1 = Product is salable.';
COMMENT ON COLUMN "production"."product"."color" IS 'Product color.';
COMMENT ON COLUMN "production"."product"."safetystocklevel" IS 'Minimum inventory quantity.';
COMMENT ON COLUMN "production"."product"."reorderpoint" IS 'Inventory level that triggers a purchase order or work order.';
COMMENT ON COLUMN "production"."product"."standardcost" IS 'Standard cost of the product.';
COMMENT ON COLUMN "production"."product"."listprice" IS 'Selling price.';
COMMENT ON COLUMN "production"."product"."size" IS 'Product size.';
COMMENT ON COLUMN "production"."product"."sizeunitmeasurecode" IS 'Unit of measure for Size column.';
COMMENT ON COLUMN "production"."product"."weightunitmeasurecode" IS 'Unit of measure for Weight column.';
COMMENT ON COLUMN "production"."product"."weight" IS 'Product weight.';
COMMENT ON COLUMN "production"."product"."daystomanufacture" IS 'Number of days required to manufacture the product.';
COMMENT ON COLUMN "production"."product"."productline" IS 'R = Road, M = Mountain, T = Touring, S = Standard';
COMMENT ON COLUMN "production"."product"."class" IS 'H = High, M = Medium, L = Low';
COMMENT ON COLUMN "production"."product"."style" IS 'W = Womens, M = Mens, U = Universal';
COMMENT ON COLUMN "production"."product"."productsubcategoryid" IS 'Product is a member of this product subcategory. Foreign key to ProductSubCategory.ProductSubCategoryID.';
COMMENT ON COLUMN "production"."product"."productmodelid" IS 'Product is a member of this product model. Foreign key to ProductModel.ProductModelID.';
COMMENT ON COLUMN "production"."product"."sellstartdate" IS 'Date the product was available for sale.';
COMMENT ON COLUMN "production"."product"."sellenddate" IS 'Date the product was no longer available for sale.';
COMMENT ON COLUMN "production"."product"."discontinueddate" IS 'Date the product was discontinued.';
COMMENT ON TABLE "production"."productcategory" IS 'High-level product categorization.';
COMMENT ON COLUMN "production"."productcategory"."productcategoryid" IS 'Primary key for ProductCategory records.';
COMMENT ON COLUMN "production"."productcategory"."name" IS 'Category description.';
COMMENT ON TABLE "production"."productcosthistory" IS 'Changes in the cost of a product over time.';
COMMENT ON COLUMN "production"."productcosthistory"."productid" IS 'Product identification number. Foreign key to Product.ProductID';
COMMENT ON COLUMN "production"."productcosthistory"."startdate" IS 'Product cost start date.';
COMMENT ON COLUMN "production"."productcosthistory"."enddate" IS 'Product cost end date.';
COMMENT ON COLUMN "production"."productcosthistory"."standardcost" IS 'Standard cost of the product.';
COMMENT ON TABLE "production"."productdescription" IS 'Product descriptions in several languages.';
COMMENT ON COLUMN "production"."productdescription"."productdescriptionid" IS 'Primary key for ProductDescription records.';
COMMENT ON COLUMN "production"."productdescription"."description" IS 'Description of the product.';
COMMENT ON TABLE "production"."productdocument" IS 'Cross-reference table mapping products to related product documents.';
COMMENT ON COLUMN "production"."productdocument"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."productdocument"."documentnode" IS 'Document identification number. Foreign key to Document.DocumentNode.';
COMMENT ON TABLE "production"."productinventory" IS 'Product inventory information.';
COMMENT ON COLUMN "production"."productinventory"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."productinventory"."locationid" IS 'Inventory location identification number. Foreign key to Location.LocationID.';
COMMENT ON COLUMN "production"."productinventory"."shelf" IS 'Storage compartment within an inventory location.';
COMMENT ON COLUMN "production"."productinventory"."bin" IS 'Storage container on a shelf in an inventory location.';
COMMENT ON COLUMN "production"."productinventory"."quantity" IS 'Quantity of products in the inventory location.';
COMMENT ON TABLE "production"."productlistpricehistory" IS 'Changes in the list price of a product over time.';
COMMENT ON COLUMN "production"."productlistpricehistory"."productid" IS 'Product identification number. Foreign key to Product.ProductID';
COMMENT ON COLUMN "production"."productlistpricehistory"."startdate" IS 'List price start date.';
COMMENT ON COLUMN "production"."productlistpricehistory"."enddate" IS 'List price end date';
COMMENT ON COLUMN "production"."productlistpricehistory"."listprice" IS 'Product list price.';
COMMENT ON TABLE "production"."productmodel" IS 'Product model classification.';
COMMENT ON COLUMN "production"."productmodel"."productmodelid" IS 'Primary key for ProductModel records.';
COMMENT ON COLUMN "production"."productmodel"."name" IS 'Product model description.';
COMMENT ON COLUMN "production"."productmodel"."catalogdescription" IS 'Detailed product catalog information in xml format.';
COMMENT ON COLUMN "production"."productmodel"."instructions" IS 'Manufacturing instructions in xml format.';
COMMENT ON TABLE "production"."productmodelillustration" IS 'Cross-reference table mapping product models and illustrations.';
COMMENT ON COLUMN "production"."productmodelillustration"."productmodelid" IS 'Primary key. Foreign key to ProductModel.ProductModelID.';
COMMENT ON COLUMN "production"."productmodelillustration"."illustrationid" IS 'Primary key. Foreign key to Illustration.IllustrationID.';
COMMENT ON TABLE "production"."productmodelproductdescriptionculture" IS 'Cross-reference table mapping product descriptions and the language the description is written in.';
COMMENT ON COLUMN "production"."productmodelproductdescriptionculture"."productmodelid" IS 'Primary key. Foreign key to ProductModel.ProductModelID.';
COMMENT ON COLUMN "production"."productmodelproductdescriptionculture"."productdescriptionid" IS 'Primary key. Foreign key to ProductDescription.ProductDescriptionID.';
COMMENT ON COLUMN "production"."productmodelproductdescriptionculture"."cultureid" IS 'Culture identification number. Foreign key to Culture.CultureID.';
COMMENT ON TABLE "production"."productphoto" IS 'Product images.';
COMMENT ON COLUMN "production"."productphoto"."productphotoid" IS 'Primary key for ProductPhoto records.';
COMMENT ON COLUMN "production"."productphoto"."thumbnailphoto" IS 'Small image of the product.';
COMMENT ON COLUMN "production"."productphoto"."thumbnailphotofilename" IS 'Small image file name.';
COMMENT ON COLUMN "production"."productphoto"."largephoto" IS 'Large image of the product.';
COMMENT ON COLUMN "production"."productphoto"."largephotofilename" IS 'Large image file name.';
COMMENT ON TABLE "production"."productproductphoto" IS 'Cross-reference table mapping products and product photos.';
COMMENT ON COLUMN "production"."productproductphoto"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."productproductphoto"."productphotoid" IS 'Product photo identification number. Foreign key to ProductPhoto.ProductPhotoID.';
COMMENT ON COLUMN "production"."productproductphoto"."primary" IS '0 = Photo is not the principal image. 1 = Photo is the principal image.';
COMMENT ON TABLE "production"."productreview" IS 'Customer reviews of products they have purchased.';
COMMENT ON COLUMN "production"."productreview"."productreviewid" IS 'Primary key for ProductReview records.';
COMMENT ON COLUMN "production"."productreview"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."productreview"."reviewername" IS 'Name of the reviewer.';
COMMENT ON COLUMN "production"."productreview"."reviewdate" IS 'Date review was submitted.';
COMMENT ON COLUMN "production"."productreview"."emailaddress" IS 'Reviewer''s e-mail address.';
COMMENT ON COLUMN "production"."productreview"."rating" IS 'Product rating given by the reviewer. Scale is 1 to 5 with 5 as the highest rating.';
COMMENT ON COLUMN "production"."productreview"."comments" IS 'Reviewer''s comments';
COMMENT ON TABLE "production"."productsubcategory" IS 'Product subcategories. See ProductCategory table.';
COMMENT ON COLUMN "production"."productsubcategory"."productsubcategoryid" IS 'Primary key for ProductSubcategory records.';
COMMENT ON COLUMN "production"."productsubcategory"."productcategoryid" IS 'Product category identification number. Foreign key to ProductCategory.ProductCategoryID.';
COMMENT ON COLUMN "production"."productsubcategory"."name" IS 'Subcategory description.';
COMMENT ON TABLE "production"."scrapreason" IS 'Manufacturing failure reasons lookup table.';
COMMENT ON COLUMN "production"."scrapreason"."scrapreasonid" IS 'Primary key for ScrapReason records.';
COMMENT ON COLUMN "production"."scrapreason"."name" IS 'Failure description.';
COMMENT ON TABLE "production"."transactionhistory" IS 'Record of each purchase order, sales order, or work order transaction year to date.';
COMMENT ON COLUMN "production"."transactionhistory"."transactionid" IS 'Primary key for TransactionHistory records.';
COMMENT ON COLUMN "production"."transactionhistory"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."transactionhistory"."referenceorderid" IS 'Purchase order, sales order, or work order identification number.';
COMMENT ON COLUMN "production"."transactionhistory"."referenceorderlineid" IS 'Line number associated with the purchase order, sales order, or work order.';
COMMENT ON COLUMN "production"."transactionhistory"."transactiondate" IS 'Date and time of the transaction.';
COMMENT ON COLUMN "production"."transactionhistory"."transactiontype" IS 'W = WorkOrder, S = SalesOrder, P = PurchaseOrder';
COMMENT ON COLUMN "production"."transactionhistory"."quantity" IS 'Product quantity.';
COMMENT ON COLUMN "production"."transactionhistory"."actualcost" IS 'Product cost.';
COMMENT ON TABLE "production"."transactionhistoryarchive" IS 'Transactions for previous years.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."transactionid" IS 'Primary key for TransactionHistoryArchive records.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."referenceorderid" IS 'Purchase order, sales order, or work order identification number.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."referenceorderlineid" IS 'Line number associated with the purchase order, sales order, or work order.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."transactiondate" IS 'Date and time of the transaction.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."transactiontype" IS 'W = Work Order, S = Sales Order, P = Purchase Order';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."quantity" IS 'Product quantity.';
COMMENT ON COLUMN "production"."transactionhistoryarchive"."actualcost" IS 'Product cost.';
COMMENT ON TABLE "production"."unitmeasure" IS 'Unit of measure lookup table.';
COMMENT ON COLUMN "production"."unitmeasure"."unitmeasurecode" IS 'Primary key.';
COMMENT ON COLUMN "production"."unitmeasure"."name" IS 'Unit of measure description.';
COMMENT ON TABLE "production"."workorder" IS 'Manufacturing work orders.';
COMMENT ON COLUMN "production"."workorder"."workorderid" IS 'Primary key for WorkOrder records.';
COMMENT ON COLUMN "production"."workorder"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."workorder"."orderqty" IS 'Product quantity to build.';
COMMENT ON COLUMN "production"."workorder"."scrappedqty" IS 'Quantity that failed inspection.';
COMMENT ON COLUMN "production"."workorder"."startdate" IS 'Work order start date.';
COMMENT ON COLUMN "production"."workorder"."enddate" IS 'Work order end date.';
COMMENT ON COLUMN "production"."workorder"."duedate" IS 'Work order due date.';
COMMENT ON COLUMN "production"."workorder"."scrapreasonid" IS 'Reason for inspection failure.';
COMMENT ON TABLE "production"."workorderrouting" IS 'Work order details.';
COMMENT ON COLUMN "production"."workorderrouting"."workorderid" IS 'Primary key. Foreign key to WorkOrder.WorkOrderID.';
COMMENT ON COLUMN "production"."workorderrouting"."productid" IS 'Primary key. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "production"."workorderrouting"."operationsequence" IS 'Primary key. Indicates the manufacturing process sequence.';
COMMENT ON COLUMN "production"."workorderrouting"."locationid" IS 'Manufacturing location where the part is processed. Foreign key to Location.LocationID.';
COMMENT ON COLUMN "production"."workorderrouting"."scheduledstartdate" IS 'Planned manufacturing start date.';
COMMENT ON COLUMN "production"."workorderrouting"."scheduledenddate" IS 'Planned manufacturing end date.';
COMMENT ON COLUMN "production"."workorderrouting"."actualstartdate" IS 'Actual start date.';
COMMENT ON COLUMN "production"."workorderrouting"."actualenddate" IS 'Actual end date.';
COMMENT ON COLUMN "production"."workorderrouting"."actualresourcehrs" IS 'Number of manufacturing hours used.';
COMMENT ON COLUMN "production"."workorderrouting"."plannedcost" IS 'Estimated manufacturing cost.';
COMMENT ON COLUMN "production"."workorderrouting"."actualcost" IS 'Actual manufacturing cost.';
COMMENT ON TABLE "purchasing"."purchaseorderdetail" IS 'Individual products associated with a specific purchase order. See PurchaseOrderHeader.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."purchaseorderid" IS 'Primary key. Foreign key to PurchaseOrderHeader.PurchaseOrderID.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."purchaseorderdetailid" IS 'Primary key. One line number per purchased product.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."duedate" IS 'Date the product is expected to be received.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."orderqty" IS 'Quantity ordered.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."unitprice" IS 'Vendor''s selling price of a single product.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."receivedqty" IS 'Quantity actually received from the vendor.';
COMMENT ON COLUMN "purchasing"."purchaseorderdetail"."rejectedqty" IS 'Quantity rejected during inspection.';
COMMENT ON TABLE "purchasing"."purchaseorderheader" IS 'General purchase order information. See PurchaseOrderDetail.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."purchaseorderid" IS 'Primary key.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."revisionnumber" IS 'Incremental number to track changes to the purchase order over time.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."status" IS 'Order current status. 1 = Pending; 2 = Approved; 3 = Rejected; 4 = Complete';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."employeeid" IS 'Employee who created the purchase order. Foreign key to Employee.BusinessEntityID.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."vendorid" IS 'Vendor with whom the purchase order is placed. Foreign key to Vendor.BusinessEntityID.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."shipmethodid" IS 'Shipping method. Foreign key to ShipMethod.ShipMethodID.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."orderdate" IS 'Purchase order creation date.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."shipdate" IS 'Estimated shipment date from the vendor.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."subtotal" IS 'Purchase order subtotal. Computed as SUM(PurchaseOrderDetail.LineTotal)for the appropriate PurchaseOrderID.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."taxamt" IS 'Tax amount.';
COMMENT ON COLUMN "purchasing"."purchaseorderheader"."freight" IS 'Shipping cost.';
COMMENT ON TABLE "purchasing"."productvendor" IS 'Cross-reference table mapping vendors with the products they supply.';
COMMENT ON COLUMN "purchasing"."productvendor"."productid" IS 'Primary key. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "purchasing"."productvendor"."businessentityid" IS 'Primary key. Foreign key to Vendor.BusinessEntityID.';
COMMENT ON COLUMN "purchasing"."productvendor"."averageleadtime" IS 'The average span of time (in days) between placing an order with the vendor and receiving the purchased product.';
COMMENT ON COLUMN "purchasing"."productvendor"."standardprice" IS 'The vendor''s usual selling price.';
COMMENT ON COLUMN "purchasing"."productvendor"."lastreceiptcost" IS 'The selling price when last purchased.';
COMMENT ON COLUMN "purchasing"."productvendor"."lastreceiptdate" IS 'Date the product was last received by the vendor.';
COMMENT ON COLUMN "purchasing"."productvendor"."minorderqty" IS 'The maximum quantity that should be ordered.';
COMMENT ON COLUMN "purchasing"."productvendor"."maxorderqty" IS 'The minimum quantity that should be ordered.';
COMMENT ON COLUMN "purchasing"."productvendor"."onorderqty" IS 'The quantity currently on order.';
COMMENT ON COLUMN "purchasing"."productvendor"."unitmeasurecode" IS 'The product''s unit of measure.';
COMMENT ON TABLE "purchasing"."shipmethod" IS 'Shipping company lookup table.';
COMMENT ON COLUMN "purchasing"."shipmethod"."shipmethodid" IS 'Primary key for ShipMethod records.';
COMMENT ON COLUMN "purchasing"."shipmethod"."name" IS 'Shipping company name.';
COMMENT ON COLUMN "purchasing"."shipmethod"."shipbase" IS 'Minimum shipping charge.';
COMMENT ON COLUMN "purchasing"."shipmethod"."shiprate" IS 'Shipping charge per pound.';
COMMENT ON TABLE "purchasing"."vendor" IS 'Companies from whom Adventure Works Cycles purchases parts or other goods.';
COMMENT ON COLUMN "purchasing"."vendor"."businessentityid" IS 'Primary key for Vendor records.  Foreign key to BusinessEntity.BusinessEntityID';
COMMENT ON COLUMN "purchasing"."vendor"."accountnumber" IS 'Vendor account (identification) number.';
COMMENT ON COLUMN "purchasing"."vendor"."name" IS 'Company name.';
COMMENT ON COLUMN "purchasing"."vendor"."creditrating" IS '1 = Superior, 2 = Excellent, 3 = Above average, 4 = Average, 5 = Below average';
COMMENT ON COLUMN "purchasing"."vendor"."preferredvendorstatus" IS '0 = Do not use if another vendor is available. 1 = Preferred over other vendors supplying the same product.';
COMMENT ON COLUMN "purchasing"."vendor"."activeflag" IS '0 = Vendor no longer used. 1 = Vendor is actively used.';
COMMENT ON COLUMN "purchasing"."vendor"."purchasingwebserviceurl" IS 'Vendor URL.';
COMMENT ON TABLE "sales"."customer" IS 'Current customer information. Also see the Person and Store tables.';
COMMENT ON COLUMN "sales"."customer"."customerid" IS 'Primary key.';
COMMENT ON COLUMN "sales"."customer"."personid" IS 'Foreign key to Person.BusinessEntityID';
COMMENT ON COLUMN "sales"."customer"."storeid" IS 'Foreign key to Store.BusinessEntityID';
COMMENT ON COLUMN "sales"."customer"."territoryid" IS 'ID of the territory in which the customer is located. Foreign key to SalesTerritory.SalesTerritoryID.';
COMMENT ON TABLE "sales"."creditcard" IS 'Customer credit card information.';
COMMENT ON COLUMN "sales"."creditcard"."creditcardid" IS 'Primary key for CreditCard records.';
COMMENT ON COLUMN "sales"."creditcard"."cardtype" IS 'Credit card name.';
COMMENT ON COLUMN "sales"."creditcard"."cardnumber" IS 'Credit card number.';
COMMENT ON COLUMN "sales"."creditcard"."expmonth" IS 'Credit card expiration month.';
COMMENT ON COLUMN "sales"."creditcard"."expyear" IS 'Credit card expiration year.';
COMMENT ON TABLE "sales"."currencyrate" IS 'Currency exchange rates.';
COMMENT ON COLUMN "sales"."currencyrate"."currencyrateid" IS 'Primary key for CurrencyRate records.';
COMMENT ON COLUMN "sales"."currencyrate"."currencyratedate" IS 'Date and time the exchange rate was obtained.';
COMMENT ON COLUMN "sales"."currencyrate"."fromcurrencycode" IS 'Exchange rate was converted from this currency code.';
COMMENT ON COLUMN "sales"."currencyrate"."tocurrencycode" IS 'Exchange rate was converted to this currency code.';
COMMENT ON COLUMN "sales"."currencyrate"."averagerate" IS 'Average exchange rate for the day.';
COMMENT ON COLUMN "sales"."currencyrate"."endofdayrate" IS 'Final exchange rate for the day.';
COMMENT ON TABLE "sales"."countryregioncurrency" IS 'Cross-reference table mapping ISO currency codes to a country or region.';
COMMENT ON COLUMN "sales"."countryregioncurrency"."countryregioncode" IS 'ISO code for countries and regions. Foreign key to CountryRegion.CountryRegionCode.';
COMMENT ON COLUMN "sales"."countryregioncurrency"."currencycode" IS 'ISO standard currency code. Foreign key to Currency.CurrencyCode.';
COMMENT ON TABLE "sales"."currency" IS 'Lookup table containing standard ISO currencies.';
COMMENT ON COLUMN "sales"."currency"."currencycode" IS 'The ISO code for the Currency.';
COMMENT ON COLUMN "sales"."currency"."name" IS 'Currency name.';
COMMENT ON TABLE "sales"."personcreditcard" IS 'Cross-reference table mapping people to their credit card information in the CreditCard table.';
COMMENT ON COLUMN "sales"."personcreditcard"."businessentityid" IS 'Business entity identification number. Foreign key to Person.BusinessEntityID.';
COMMENT ON COLUMN "sales"."personcreditcard"."creditcardid" IS 'Credit card identification number. Foreign key to CreditCard.CreditCardID.';
COMMENT ON TABLE "sales"."store" IS 'Customers (resellers) of Adventure Works products.';
COMMENT ON COLUMN "sales"."store"."businessentityid" IS 'Primary key. Foreign key to Customer.BusinessEntityID.';
COMMENT ON COLUMN "sales"."store"."name" IS 'Name of the store.';
COMMENT ON COLUMN "sales"."store"."salespersonid" IS 'ID of the sales person assigned to the customer. Foreign key to SalesPerson.BusinessEntityID.';
COMMENT ON COLUMN "sales"."store"."demographics" IS 'Demographic informationg about the store such as the number of employees, annual sales and store type.';
COMMENT ON TABLE "sales"."shoppingcartitem" IS 'Contains online customer orders until the order is submitted or cancelled.';
COMMENT ON COLUMN "sales"."shoppingcartitem"."shoppingcartitemid" IS 'Primary key for ShoppingCartItem records.';
COMMENT ON COLUMN "sales"."shoppingcartitem"."shoppingcartid" IS 'Shopping cart identification number.';
COMMENT ON COLUMN "sales"."shoppingcartitem"."quantity" IS 'Product quantity ordered.';
COMMENT ON COLUMN "sales"."shoppingcartitem"."productid" IS 'Product ordered. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "sales"."shoppingcartitem"."datecreated" IS 'Date the time the record was created.';
COMMENT ON TABLE "sales"."specialoffer" IS 'Sale discounts lookup table.';
COMMENT ON COLUMN "sales"."specialoffer"."specialofferid" IS 'Primary key for SpecialOffer records.';
COMMENT ON COLUMN "sales"."specialoffer"."description" IS 'Discount description.';
COMMENT ON COLUMN "sales"."specialoffer"."discountpct" IS 'Discount precentage.';
COMMENT ON COLUMN "sales"."specialoffer"."type" IS 'Discount type category.';
COMMENT ON COLUMN "sales"."specialoffer"."category" IS 'Group the discount applies to such as Reseller or Customer.';
COMMENT ON COLUMN "sales"."specialoffer"."startdate" IS 'Discount start date.';
COMMENT ON COLUMN "sales"."specialoffer"."enddate" IS 'Discount end date.';
COMMENT ON COLUMN "sales"."specialoffer"."minqty" IS 'Minimum discount percent allowed.';
COMMENT ON COLUMN "sales"."specialoffer"."maxqty" IS 'Maximum discount percent allowed.';
COMMENT ON TABLE "sales"."salesorderdetail" IS 'Individual products associated with a specific sales order. See SalesOrderHeader.';
COMMENT ON COLUMN "sales"."salesorderdetail"."salesorderid" IS 'Primary key. Foreign key to SalesOrderHeader.SalesOrderID.';
COMMENT ON COLUMN "sales"."salesorderdetail"."salesorderdetailid" IS 'Primary key. One incremental unique number per product sold.';
COMMENT ON COLUMN "sales"."salesorderdetail"."carriertrackingnumber" IS 'Shipment tracking number supplied by the shipper.';
COMMENT ON COLUMN "sales"."salesorderdetail"."orderqty" IS 'Quantity ordered per product.';
COMMENT ON COLUMN "sales"."salesorderdetail"."productid" IS 'Product sold to customer. Foreign key to Product.ProductID.';
COMMENT ON COLUMN "sales"."salesorderdetail"."specialofferid" IS 'Promotional code. Foreign key to SpecialOffer.SpecialOfferID.';
COMMENT ON COLUMN "sales"."salesorderdetail"."unitprice" IS 'Selling price of a single product.';
COMMENT ON COLUMN "sales"."salesorderdetail"."unitpricediscount" IS 'Discount amount.';
COMMENT ON TABLE "sales"."salesorderheader" IS 'General sales order information.';
COMMENT ON COLUMN "sales"."salesorderheader"."salesorderid" IS 'Primary key.';
COMMENT ON COLUMN "sales"."salesorderheader"."revisionnumber" IS 'Incremental number to track changes to the sales order over time.';
COMMENT ON COLUMN "sales"."salesorderheader"."orderdate" IS 'Dates the sales order was created.';
COMMENT ON COLUMN "sales"."salesorderheader"."duedate" IS 'Date the order is due to the customer.';
COMMENT ON COLUMN "sales"."salesorderheader"."shipdate" IS 'Date the order was shipped to the customer.';
COMMENT ON COLUMN "sales"."salesorderheader"."status" IS 'Order current status. 1 = In process; 2 = Approved; 3 = Backordered; 4 = Rejected; 5 = Shipped; 6 = Cancelled';
COMMENT ON COLUMN "sales"."salesorderheader"."onlineorderflag" IS '0 = Order placed by sales person. 1 = Order placed online by customer.';
COMMENT ON COLUMN "sales"."salesorderheader"."purchaseordernumber" IS 'Customer purchase order number reference.';
COMMENT ON COLUMN "sales"."salesorderheader"."accountnumber" IS 'Financial accounting number reference.';
COMMENT ON COLUMN "sales"."salesorderheader"."customerid" IS 'Customer identification number. Foreign key to Customer.BusinessEntityID.';
COMMENT ON COLUMN "sales"."salesorderheader"."salespersonid" IS 'Sales person who created the sales order. Foreign key to SalesPerson.BusinessEntityID.';
COMMENT ON COLUMN "sales"."salesorderheader"."territoryid" IS 'Territory in which the sale was made. Foreign key to SalesTerritory.SalesTerritoryID.';
COMMENT ON COLUMN "sales"."salesorderheader"."billtoaddressid" IS 'Customer billing address. Foreign key to Address.AddressID.';
COMMENT ON COLUMN "sales"."salesorderheader"."shiptoaddressid" IS 'Customer shipping address. Foreign key to Address.AddressID.';
COMMENT ON COLUMN "sales"."salesorderheader"."shipmethodid" IS 'Shipping method. Foreign key to ShipMethod.ShipMethodID.';
COMMENT ON COLUMN "sales"."salesorderheader"."creditcardid" IS 'Credit card identification number. Foreign key to CreditCard.CreditCardID.';
COMMENT ON COLUMN "sales"."salesorderheader"."creditcardapprovalcode" IS 'Approval code provided by the credit card company.';
COMMENT ON COLUMN "sales"."salesorderheader"."currencyrateid" IS 'Currency exchange rate used. Foreign key to CurrencyRate.CurrencyRateID.';
COMMENT ON COLUMN "sales"."salesorderheader"."subtotal" IS 'Sales subtotal. Computed as SUM(SalesOrderDetail.LineTotal)for the appropriate SalesOrderID.';
COMMENT ON COLUMN "sales"."salesorderheader"."taxamt" IS 'Tax amount.';
COMMENT ON COLUMN "sales"."salesorderheader"."freight" IS 'Shipping cost.';
COMMENT ON COLUMN "sales"."salesorderheader"."totaldue" IS 'Total due from customer. Computed as Subtotal + TaxAmt + Freight.';
COMMENT ON COLUMN "sales"."salesorderheader"."comment" IS 'Sales representative comments.';
COMMENT ON TABLE "sales"."salesorderheadersalesreason" IS 'Cross-reference table mapping sales orders to sales reason codes.';
COMMENT ON COLUMN "sales"."salesorderheadersalesreason"."salesorderid" IS 'Primary key. Foreign key to SalesOrderHeader.SalesOrderID.';
COMMENT ON COLUMN "sales"."salesorderheadersalesreason"."salesreasonid" IS 'Primary key. Foreign key to SalesReason.SalesReasonID.';
COMMENT ON TABLE "sales"."specialofferproduct" IS 'Cross-reference table mapping products to special offer discounts.';
COMMENT ON COLUMN "sales"."specialofferproduct"."specialofferid" IS 'Primary key for SpecialOfferProduct records.';
COMMENT ON COLUMN "sales"."specialofferproduct"."productid" IS 'Product identification number. Foreign key to Product.ProductID.';
COMMENT ON TABLE "sales"."salesperson" IS 'Sales representative current information.';
COMMENT ON COLUMN "sales"."salesperson"."businessentityid" IS 'Primary key for SalesPerson records. Foreign key to Employee.BusinessEntityID';
COMMENT ON COLUMN "sales"."salesperson"."territoryid" IS 'Territory currently assigned to. Foreign key to SalesTerritory.SalesTerritoryID.';
COMMENT ON COLUMN "sales"."salesperson"."salesquota" IS 'Projected yearly sales.';
COMMENT ON COLUMN "sales"."salesperson"."bonus" IS 'Bonus due if quota is met.';
COMMENT ON COLUMN "sales"."salesperson"."commissionpct" IS 'Commision percent received per sale.';
COMMENT ON COLUMN "sales"."salesperson"."salesytd" IS 'Sales total year to date.';
COMMENT ON COLUMN "sales"."salesperson"."saleslastyear" IS 'Sales total of previous year.';
COMMENT ON TABLE "sales"."salespersonquotahistory" IS 'Sales performance tracking.';
COMMENT ON COLUMN "sales"."salespersonquotahistory"."businessentityid" IS 'Sales person identification number. Foreign key to SalesPerson.BusinessEntityID.';
COMMENT ON COLUMN "sales"."salespersonquotahistory"."quotadate" IS 'Sales quota date.';
COMMENT ON COLUMN "sales"."salespersonquotahistory"."salesquota" IS 'Sales quota amount.';
COMMENT ON TABLE "sales"."salesreason" IS 'Lookup table of customer purchase reasons.';
COMMENT ON COLUMN "sales"."salesreason"."salesreasonid" IS 'Primary key for SalesReason records.';
COMMENT ON COLUMN "sales"."salesreason"."name" IS 'Sales reason description.';
COMMENT ON COLUMN "sales"."salesreason"."reasontype" IS 'Category the sales reason belongs to.';
COMMENT ON TABLE "sales"."salesterritory" IS 'Sales territory lookup table.';
COMMENT ON COLUMN "sales"."salesterritory"."territoryid" IS 'Primary key for SalesTerritory records.';
COMMENT ON COLUMN "sales"."salesterritory"."name" IS 'Sales territory description';
COMMENT ON COLUMN "sales"."salesterritory"."countryregioncode" IS 'ISO standard country or region code. Foreign key to CountryRegion.CountryRegionCode.';
COMMENT ON COLUMN "sales"."salesterritory"."group" IS 'Geographic area to which the sales territory belong.';
COMMENT ON COLUMN "sales"."salesterritory"."salesytd" IS 'Sales in the territory year to date.';
COMMENT ON COLUMN "sales"."salesterritory"."saleslastyear" IS 'Sales in the territory the previous year.';
COMMENT ON COLUMN "sales"."salesterritory"."costytd" IS 'Business costs in the territory year to date.';
COMMENT ON COLUMN "sales"."salesterritory"."costlastyear" IS 'Business costs in the territory the previous year.';
COMMENT ON TABLE "sales"."salesterritoryhistory" IS 'Sales representative transfers to other sales territories.';
COMMENT ON COLUMN "sales"."salesterritoryhistory"."businessentityid" IS 'Primary key. The sales rep.  Foreign key to SalesPerson.BusinessEntityID.';
COMMENT ON COLUMN "sales"."salesterritoryhistory"."territoryid" IS 'Primary key. Territory identification number. Foreign key to SalesTerritory.SalesTerritoryID.';
COMMENT ON COLUMN "sales"."salesterritoryhistory"."startdate" IS 'Primary key. Date the sales representive started work in the territory.';
COMMENT ON COLUMN "sales"."salesterritoryhistory"."enddate" IS 'Date the sales representative left work in the territory.';
COMMENT ON TABLE "sales"."salestaxrate" IS 'Tax rate lookup table.';
COMMENT ON COLUMN "sales"."salestaxrate"."salestaxrateid" IS 'Primary key for SalesTaxRate records.';
COMMENT ON COLUMN "sales"."salestaxrate"."stateprovinceid" IS 'State, province, or country/region the sales tax applies to.';
COMMENT ON COLUMN "sales"."salestaxrate"."taxtype" IS '1 = Tax applied to retail transactions, 2 = Tax applied to wholesale transactions, 3 = Tax applied to all sales (retail and wholesale) transactions.';
COMMENT ON COLUMN "sales"."salestaxrate"."taxrate" IS 'Tax rate amount.';
COMMENT ON COLUMN "sales"."salestaxrate"."name" IS 'Tax rate description.';
