package models

import "github.com/99designs/gqlgen/integration/server/remote_api"

type Viewer struct {
	User *remote_api.User
}
