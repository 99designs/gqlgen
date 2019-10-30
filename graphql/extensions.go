package graphql

import "context"

func GetExtensions(ctx context.Context) map[string]interface{} {
	ext := GetRequestContext(ctx).Extensions
	if ext == nil {
		return map[string]interface{}{}
	}

	return ext
}
