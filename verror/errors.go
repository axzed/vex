package verror

type VError struct {
	err     error
	ErrFunc ErrorFunc
}

func Default() *VError {
	return &VError{}
}

func (e *VError) Error() string {
	return e.err.Error()
}

// Put the error in VError
func (e *VError) Put(err error) {
	e.check(err)
	e.err = err
}

// check the error and set
func (e *VError) check(err error) {
	if err != nil {
		e.err = err
		panic(e)
	}
}

type ErrorFunc func(vError *VError)

func (e *VError) Result(errorFunc ErrorFunc) {
	e.ErrFunc = errorFunc
}

// ExecResult to handle error
// Explose to user to set this
func (e *VError) ExecResult() {
	e.ErrFunc(e)
}
