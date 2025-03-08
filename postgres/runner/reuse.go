package postgresrunner

import pgcnt "github.com/amidgo/containers/postgres"

var reusable = pgcnt.NewReusable(RunContainer(nil))

func Reusable() *pgcnt.Reusable {
	return reusable
}
