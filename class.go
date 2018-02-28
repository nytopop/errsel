package errsel

import (
	"github.com/pkg/errors"
)

// Class is an interface for things that are both lifters and selectors.
type Class interface {
	Lifter
	Selector
}

var _ Class = new(errClass)

type errClass struct {
	Lifter
	Selector
}

func ToClass(lft Lifter, sel Selector) Class {
	return &errClass{
		Lifter:   lft,
		Selector: sel,
	}
}

func FromClass(cls Class) (Lifter, Selector) {
	return LifterFunc(cls.Lift), SelectorFunc(cls.Traverse)
}

// Lifter can lift errors into another scope. It can also perform some common
// operations on errors such as wrapping, attaching stack traces, and building
// new errors.
//
// Lift is the minimum complete definition; every other method can be derived
// automatically upon converting an appropriate Lift function to a LifterFunc.
type Lifter interface {
	// Lift lifts an error into a new scope.
	Lift(err error) error

	New(msg string) error
	Errorf(format string, args ...interface{}) error

	WithStack(err error) error
	WithMessage(err error, msg string) error

	Wrap(err error, msg string) error
	Wrapf(err error, format string, args ...interface{}) error
}

// LifterFunc lifts an error to another scope.
type LifterFunc func(error) error

func (f LifterFunc) Lift(err error) error {
	return f(err)
}

func (f LifterFunc) New(msg string) error {
	return f(errors.New(msg))
}

func (f LifterFunc) Errorf(format string, args ...interface{}) error {
	return f(errors.Errorf(format, args...))
}

func (f LifterFunc) WithStack(err error) error {
	return f(errors.WithStack(err))
}

func (f LifterFunc) WithMessage(err error, msg string) error {
	return f(errors.WithMessage(err, msg))
}

func (f LifterFunc) Wrap(err error, msg string) error {
	return f(errors.Wrap(err, msg))
}

func (f LifterFunc) Wrapf(err error, format string, args ...interface{}) error {
	return f(errors.Wrapf(err, format, args...))
}
