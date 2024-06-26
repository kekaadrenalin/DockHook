package errors

type HttpError struct {
	StatusCode int
	Message    string
	Err        error
}

func (m *HttpError) Error() string {
	if m.Message != "" {
		return m.Message
	}

	return m.Err.Error()
}
