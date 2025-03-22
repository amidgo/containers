package miniorunner

import miniocontainer "github.com/amidgo/containers/minio"

var reusable = miniocontainer.NewReusable(RunContainer(nil))

func Reusable() *miniocontainer.Reusable {
	return reusable
}
