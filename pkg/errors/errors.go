package errors

type HTTPError struct {
	StatusCode int
	Message    string
	Err        error
}

func (m *HTTPError) Error() string {
	if m.Message != "" {
		return m.Message
	}

	return m.Err.Error()
}
