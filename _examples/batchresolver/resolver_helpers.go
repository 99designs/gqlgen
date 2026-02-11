package batchresolver

import (
	"fmt"

	"github.com/vektah/gqlparser/v2/ast"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func resolveProfile(r *Resolver, idx int) (*Profile, error) {
	if r.profileGqlErrPathIdxs != nil {
		if _, ok := r.profileGqlErrPathIdxs[idx]; ok {
			return nil, &gqlerror.Error{
				Message: fmt.Sprintf("profile gqlerror path set at index %d", idx),
				Path:    ast.Path{ast.PathName("custom"), ast.PathIndex(idx)},
			}
		}
	}
	if r.profileGqlErrNoPathIdxs != nil {
		if _, ok := r.profileGqlErrNoPathIdxs[idx]; ok {
			return nil, gqlerror.Errorf("profile gqlerror path nil at index %d", idx)
		}
	}
	if r.profileErrListIdxs != nil {
		if _, ok := r.profileErrListIdxs[idx]; ok {
			return nil, gqlerror.List{
				gqlerror.Errorf("profile list error 1 at index %d", idx),
				gqlerror.Errorf("profile list error 2 at index %d", idx),
			}
		}
	}
	if r.profileErrWithValueIdxs != nil {
		if _, ok := r.profileErrWithValueIdxs[idx]; ok {
			var value *Profile
			if idx >= 0 && idx < len(r.profiles) {
				value = r.profiles[idx]
			}
			return value, fmt.Errorf("profile error with value at index %d", idx)
		}
	}
	if idx == r.profileErrIdx {
		return nil, fmt.Errorf("profile error at index %d", idx)
	}
	if idx < 0 || idx >= len(r.profiles) {
		return nil, fmt.Errorf("profile not set at index %d", idx)
	}
	return r.profiles[idx], nil
}
