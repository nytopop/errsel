package errsel

import (
	"github.com/pkg/errors"
)

// Class is an interface for things that are both lifters and selectors.
//
// The minimum required implementation for a class is a lift function and
// a selector. From those, a class can be automatically derived.
//
//    var _ Class = ToClass(LifterFunc(liftFunction), selector)
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

func Bind(f, g Class) Class {
	// TODO: this should be optimized...
	//       we should be able to fuse their traversal functions
	//       into a single pass over the context chain, instead
	//       of traversing for every bound class
	//
	// TODO: type annotations on a standard func(error) bool
	return ToClass(f.Bind(g), And(f, g))
}

func Binds(f Class, gs ...Class) Class {
	for _, g := range gs {
		f = Bind(f, g)
	}
	return f
}

func BindL(f, g Class) Class {
	return ToClass(f.Bind(g), AndL(f, g))
}

func BindsL(f Class, gs ...Class) Class {
	for _, g := range gs {
		f = BindL(f, g)
	}
	return f
}

// Lifter can lift errors into another scope. It can also perform some common
// operations on errors such as wrapping, attaching stack traces, and building
// new errors.
//
// Lift is the minimum complete definition; every other method can be derived
// automatically upon converting an appropriate Lift function to a LifterFunc.
//
//    func nothing(err error) error {
//        // do whatever you like here
//        return err
//    }
//
//    var _ Lifter = LifterFunc(nothing)
type Lifter interface {
	// Lift lifts an error into a new scope.
	Lift(err error) error
	Bind(lft Lifter) Lifter

	New(msg string) error
	Errorf(format string, args ...interface{}) error

	WithStack(err error) error
	WithMessage(err error, msg string) error

	Wrap(err error, msg string) error
	Wrapf(err error, format string, args ...interface{}) error
}

// LifterFunc lifts an error to another scope.
type LifterFunc func(error) error

// Lift an error into a new scope. If err is nil, Lift always returns
// nil.
func (f LifterFunc) Lift(err error) error {
	if err == nil {
		return nil
	}
	return f(err)
}

// Bind binds the lifterfunc to the provided lifter, and returns a
// new lifter that will lift errors into both.
func (f LifterFunc) Bind(lft Lifter) Lifter {
	// aka
	// bind f lft = lft >>= f
	return LifterFunc(func(err error) error {
		return f(lft.Lift(err))
	})
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
