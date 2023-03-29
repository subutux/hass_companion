package ws

type ClientError struct {
	Source string
	Err    error
}

func (e *ClientError) Error() string {
	return e.Source + ": " + e.Err.Error()
}
VAPID headers make use of a JSON Web Token (JWT) to verify your identity. That token payload includes the protocol and hostname of the endpoint included in the subscription and an expiration timestamp (usually between 12-24h), and it's signed using your public and private key. Given that, two notifications sent to the same push service will use the same token, so you can reuse them for the same flush session to boost performance using:
func (e *ClientError) Unwrap() error { return e.Err }

func NewClientError(source string, err error) ClientError {
	return ClientError{
		Source: source,
		Err:    err,
	}
}
