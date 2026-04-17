package eventbus

import "errors"

var (
	ErrTransient = errors.New("eventbus: transient processing error")
	ErrPermanent = errors.New("eventbus: permanent processing error")
)

type wrappedError struct {
	kind error
	err  error
}

func (w wrappedError) Error() string {
	if w.err == nil {
		return w.kind.Error()
	}
	return w.err.Error()
}

func (w wrappedError) Unwrap() error {
	return w.kind
}

func Transient(err error) error {
	return wrappedError{kind: ErrTransient, err: err}
}

func Permanent(err error) error {
	return wrappedError{kind: ErrPermanent, err: err}
}

func IsTransient(err error) bool {
	return errors.Is(err, ErrTransient)
}

func IsPermanent(err error) bool {
	return errors.Is(err, ErrPermanent)
}
