package models

import "github.com/apito-cms/gqlgen/integration/server/remote_api"

type Viewer struct {
	User *remote_api.User
}
