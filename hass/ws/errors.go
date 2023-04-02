package ws

type ClientError struct {
	Source string
	Err    error
}

func (e *ClientError) Error() string {
	return e.Source + ": " + e.Err.Error()
}
func (e *ClientError) Unwrap() error { return e.Err }

func NewClientError(source string, err error) ClientError {
	return ClientError{
		Source: source,
		Err:    err,
	}
}
