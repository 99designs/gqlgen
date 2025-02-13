package models

import "github.com/john-markham/gqlgen/integration/server/remote_api"

type Viewer struct {
	User *remote_api.User
}
