package batchresolver

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	users                   []*User
	profiles                []*Profile
	profileErrIdx           int
	profileErrWithValueIdxs map[int]struct{}
	profileErrListIdxs      map[int]struct{}
	profileGqlErrNoPathIdxs map[int]struct{}
	profileGqlErrPathIdxs   map[int]struct{}
	profileWrongLen         bool
	batchErrsWrongLen       bool
	batchErrsLen            int
	batchResultsWrongLen    bool
	batchResultsLen         int
	batchErrListIdxs        map[int]struct{}
}

func (r *Resolver) userIndex(obj *User) int {
	if obj == nil {
		return -1
	}
	for i := range r.users {
		if r.users[i] == obj {
			return i
		}
	}
	return -1
}
