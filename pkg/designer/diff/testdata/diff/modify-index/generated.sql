DROP INDEX "ix_notifications_userId";

CREATE UNIQUE INDEX "ix_notifications_userId" ON "notifications" (
	"userId"
)
	WHERE statusId IN (1, 2);

