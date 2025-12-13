package httprpc

// StatusError marks an error with an HTTP status code for encoding.
// It can be returned by handlers or used internally (e.g. decode failures).
type StatusError struct {
	Status int
	Err    error
}

func (e StatusError) Error() string {
	if e.Err == nil {
		return ""
	}
	return e.Err.Error()
}

func (e StatusError) Unwrap() error { return e.Err }

