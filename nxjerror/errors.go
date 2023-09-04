package nxjerror

type NxjError struct {
	err    error
	ErrFuc ErrorFunc
}

func Default() *NxjError {
	return &NxjError{}
}

func (e *NxjError) Error() string {
	return e.err.Error()
}

func (e *NxjError) Put(err error) {
	e.check(err)
	e.err = err
}

func (e *NxjError) check(err error) {
	if err != nil {
		e.err = err
		panic(e)
	}
}

type ErrorFunc func(nxjError *NxjError)

func (e *NxjError) Result(errorFunc ErrorFunc) {
	e.ErrFuc = errorFunc
}

func (e *NxjError) ExecResult() {
	e.ErrFuc(e)
}
