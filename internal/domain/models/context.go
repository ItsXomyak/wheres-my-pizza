package models

// Context key for request_id (unexported to avoid collisions)
type ctxKey struct{}

var requestIDKey = &ctxKey{}

func GetRequestIDKey() *ctxKey {
	return requestIDKey
}
