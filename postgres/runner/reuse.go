package postgrescontainerrunner

import postgrescontainer "github.com/amidgo/containers/postgres"

var (
	reusable = postgrescontainer.NewReusable(RunContainer(nil))
)

func Reusable() *postgrescontainer.Reusable {
	return reusable
}
