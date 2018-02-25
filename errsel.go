// Package errsel builds on the concept of error causers in
// github.com/pkg/errors by extending errors with a particular
// set of classes.
//
// This package also provides an efficient, flexible method for
// querying error conditions in the form of selectors. Error
// classes from this package are themselves selectors, and provide
// an abstract, structured approach to errors that is fully
// compatible with common go error handling idioms.
//
// In essence, classes are a mechanism of abstraction over error
// types, hopefully one more pleasant to work with than a complex
// series of boolean expressions and branch statements.
//
// For example, the following (which may be duplicated in many places):
//
//    func something() {
//        err := someJankyFunction()
//        if err != nil && errors.Cause(err) != sql.ErrNoRows {
//            if errors.Cause(err) == sql.ErrConflict || errors.Cause(err) == sql.ErrTransactionClosed {
//               // actually handle the error
//            }
//        }
//    }
//
// Becomes this, which can be composed and reused anywhere:
//
//    var thatCommonErr = And(
//       Not(Error(sql.ErrNoRows)),
//       Or(
//         Error(sql.ErrConflict),
//         Error(sql.ErrTransactionClosed),
//       ),
//    )
//
//    func something() {
//        if err := someJankyFunction(); err != nil {
//            if thatCommonErr.In(err) {
//                // handle the error
//            }
//            // more selectors, or just bail
//        }
//    }
//
// Further, selectors automatically traverse the entire error chain if
// it was ever wrapped, and also include intermediate errors in their
// search. It's rather easy to forget an invocation of errors.Cause()
// on an error in these checks (leading to bugs), so the selector tends
// to be a more robust solution. Full chain search in selectors also
// means that even wrapped sentinel errors like:
//
//     var ErrKindaBad = errors.New("it's kinda bad")
//     var ErrVeryBad = errors.Wrap(ErrKindaBad, "it was kinda bad, now it's very bad")
//
// can be inspected with a trivial query like:
//
//     var isErrVeryBad = Error(ErrVeryBad)
//
// Trying to inspect 'sideways' errors like the above would require manual
// traversal of the error chain or (more commonly) some janky searching
// of the error's string representation. Please note that errsel also
// supports janky string searches via Grep(), if you must use one.
//
// Error classes extend selectors with a method to abstract away entire
// categories of errors by creating annotations within an error's context
// chain. While this could be accomplished with a series of normal selectors
// and carefully crafted error constructors, classes make it easier and
// more flexible.
//
//    var numErr = Anonymous()
//
//    func checkN(n int) error {
//       if n < 0 {
//           return numErr.New("negative")
//       }
//       if n > 5000 {
//           return numErr.New("n is probably too large")
//       }
//       return nil
//    }
package errsel

import (
	"reflect"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

type causer interface {
	Cause() error
}

// CausesOf returns a slice containing every intermediate error in
// an error's context chain.
//
// If any traverse options are provided, they will use wrapped error
// counts, not classes. For example, Lens(2) will lens 2 errors into
// the context chain, not 2 classes.
func CausesOf(err error, opts ...TraverseOption) []error {
	cfg := new(traverseConfig)
	for _, f := range opts {
		f(cfg)
	}

	errs := []error{err}
	if c, ok := err.(causer); ok {
		// recurse to find additional causes
		// we don't pass traverse options to recursive calls
		errs = append(errs, CausesOf(c.Cause())...)
	}

	// apply lensing
	if cfg.lens < uint(len(errs)) {
		errs = errs[cfg.lens:]
	} else {
		errs = errs[len(errs)-1:]
	}

	// apply depth cutoff
	if cfg.depth != 0 && uint(len(errs)) > cfg.depth {
		errs = errs[:cfg.depth+1]
	}

	return errs
}

var (
	_ error   = new(classErr)
	_ causer  = new(classErr)
	_ classer = new(classErr)
)

type classErr struct {
	class *Class
	err   error
}

// Error calls Error on the underlying error.
func (c *classErr) Error() string {
	return c.err.Error()
}

// Cause returns the cause of the underlying error, or the error
// itself if there is none.
func (c *classErr) Cause() error {
	if cc, ok := c.err.(causer); ok {
		return cc.Cause()
	}
	return c
}

// Class returns the underlying class.
func (c *classErr) Class() *Class {
	return c.class
}

var _ Selector = new(Class)

// Class represents an error class. The zero value is a valid
// configuration.
//
// When used as a selector, all classes will respect provided
// traverse options as if they were provided to ClassesOf.
type Class struct {
	named  bool
	name   string
	shadow bool
}

// Anonymous returns an anonymous class (the zero value of Class).
//
// When used as a selector, it will match only against itself. Making
// a copy will effectively create a new anonymous class distinct from
// the original.
//
// Due to its dependence on an address comparison, it should probably
// not cross package boundaries.
func Anonymous() *Class {
	return &Class{}
}

// Named returns a named class.
//
// When used as a selector, it will match against any other named
// class with exactly the same name.
func Named(name string) *Class {
	return &Class{
		named: true,
		name:  name,
	}
}

// AnonymousShadow returns an anonymous, shadowing class. Wrapping
// an error with it will hide any deeper class definitions in the
// error's context chain. This can be useful if you need to logically
// segment internal and external errors.
//
// When used as a selector, it will match only against itself. The same
// copying restrictions as Anonymous apply.
func AnonymousShadow() *Class {
	return &Class{
		shadow: true,
	}
}

// NamedShadow returns a named, shadowing class. Wrapping an error
// with it will hide any deeper class definitions in the error's
// context chain. This can be useful if you need to logically segment
// internal and external errors across package boundaries.
//
// When used as a selector, it will match against any other named
// class with exactly the same name.
func NamedShadow(name string) *Class {
	return &Class{
		named:  true,
		name:   name,
		shadow: true,
	}
}

type classer interface {
	Class() *Class
}

// ClassesOf returns a slice containing every class in the provided
// error's context chain.
//
// If any traverse options are provided, they will use class counts
// and not total wrapped errors. For example, Lens(4) will lens 4
// classes into the context chain, which may consist of more than
// 4 errors.
//
// This function will respect class shadowing; use a Lens if you want
// to skip past a shadowing class. Do this with care, as a shadowing
// class was likely used for a reason.
func ClassesOf(err error, opts ...TraverseOption) []*Class {
	cfg := new(traverseConfig)
	for _, f := range opts {
		f(cfg)
	}

	var classes []*Class
	for _, e := range CausesOf(err) {
		if c, ok := e.(classer); ok {
			classes = append(classes, c.Class())
		}
	}

	// apply lensing
	if cfg.lens < uint(len(classes)) {
		classes = classes[cfg.lens:]
	} else {
		classes = classes[len(classes)-1:]
	}

	// apply depth cutoff
	if cfg.depth != 0 && uint(len(classes)) > cfg.depth {
		classes = classes[:cfg.depth+1]
	}

	// apply shadowing
	for i, cs := range classes {
		if cs.shadow {
			classes = classes[:i+1]
			break
		}
	}

	return classes
}

// In checks if an error has an instance of the class.
func (e *Class) In(err error, opts ...TraverseOption) bool {
	// retrieve complete chain and match
	classes := ClassesOf(err, opts...)
	for _, cs := range classes {
		if cs == e {
			return true
		}

		if cs.named && e.named {
			if cs.name == e.name {
				return true
			}
		}
	}

	return false
}

// Wrapc wraps the provided error with a class.
func (e *Class) Wrapc(err error) error {
	return &classErr{
		class: e,
		err:   err,
	}
}

func (e *Class) Errorf(format string, args ...interface{}) error {
	return e.Wrapc(errors.Errorf(format, args...))
}

func (e *Class) New(message string) error {
	return e.Wrapc(errors.New(message))
}

func (e *Class) WithMessage(err error, message string) error {
	return e.Wrapc(errors.WithMessage(err, message))
}

func (e *Class) WithStack(err error) error {
	return e.Wrapc(errors.WithStack(err))
}

func (e *Class) Wrap(err error, message string) error {
	return e.Wrapc(errors.Wrap(err, message))
}

func (e *Class) Wrapf(err error, format string, args ...interface{}) error {
	return e.Wrapc(errors.Wrapf(err, format, args...))
}

type traverseConfig struct {
	lens  uint
	depth uint
}

type TraverseOption func(*traverseConfig)

// Lens sets lensing depth to k elements.
func Lens(k uint) TraverseOption {
	return TraverseOption(func(c *traverseConfig) {
		c.lens = k
	})
}

// Depth sets maximum traversal depth to d elements.
func Depth(d uint) TraverseOption {
	return TraverseOption(func(c *traverseConfig) {
		c.depth = d
	})
}

// Selector provides an interface for composing error control flow.
type Selector interface {
	In(err error, opts ...TraverseOption) bool
}

var _ Selector = new(SelectorFunc)

// SelectorFunc implements Selector.
type SelectorFunc func(err error, opts ...TraverseOption) bool

// In calls the underlying function.
func (f SelectorFunc) In(err error, opts ...TraverseOption) bool {
	return f(err, opts...)
}

// And returns a selector that will only match if all input selectors
// match.
func And(selectors ...Selector) Selector {
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		for _, sel := range selectors {
			if !sel.In(err, opts...) {
				return false
			}
		}
		return true
	})
}

// AndC behaves like And, except that input selectors will be executed
// concurrently.
func AndC(selectors ...Selector) Selector {
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		var eg errgroup.Group
		for _, sel := range selectors {
			eg.Go(func() error {
				if !sel.In(err, opts...) {
					return new(classErr)
				}
				return nil
			})
		}

		return eg.Wait() == nil
	})
}

// Or returns a selector that will match if any of the input selectors
// match.
func Or(selectors ...Selector) Selector {
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		for _, sel := range selectors {
			if sel.In(err, opts...) {
				return true
			}
		}
		return false
	})
}

// OrC behaves like Or, except that input selectors will be executed
// concurrently.
func OrC(selectors ...Selector) Selector {
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		var eg errgroup.Group
		for _, sel := range selectors {
			eg.Go(func() error {
				if sel.In(err, opts...) {
					return new(classErr)
				}
				return nil
			})
		}
		return eg.Wait() != nil
	})
}

// Not inverts the input selector.
func Not(selector Selector) Selector {
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		return !selector.In(err, opts...)
	})
}

// Error returns a selector that will match if the provided error
// occurs anywhere in an error's context chain.
//
// If any traverse options are provided, they will behave as if passed
// to CausesOf.
func Error(err error) Selector {
	return SelectorFunc(func(e error, opts ...TraverseOption) bool {
		for _, c := range CausesOf(e, opts...) {
			if c == e {
				return true
			}
		}
		return false
	})
}

// Type returns a selector that will match if the provided type
// occurs anywhere in an error's context chain.
//
// It's not super useful yet and uses reflect internally, so it
// should probably be avoided until improved or deprecated.
//
// If any traverse options are provided, they will behave as if
// passed to CausesOf.
//
// TODO: There is currently no way to extract the actual value
// of the error. that's an oversight :\
//
// hmm. what about we just write directly to the interface?
//
func Type(t interface{}) Selector {
	T := reflect.TypeOf(t)
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		for _, c := range CausesOf(err, opts...) {
			if reflect.TypeOf(c) == T {
				return true
			}
		}
		return false
	})
}

// Grep returns a selector that will match if the provided string
// is a substring of an error's concatenated Error() output.
func Grep(str string) Selector {
	return SelectorFunc(func(err error, _ ...TraverseOption) bool {
		idx := strings.Index(err.Error(), str)
		return idx != -1
	})
}

// Call returns a selector that will call the provided function if the
// provided selector matches. The result of the match is passed through
// without modification, so it can be used at any level within a selector
// definition.
//
// This can be useful if a certain error condition needs to be handled
// a certain way in all cases; using a Call selector means it becomes
// impossible to forget to do this. For example, a Call might be used
// to execute a log function.
func Call(f func(error), selector Selector) Selector {
	return SelectorFunc(func(err error, opts ...TraverseOption) bool {
		matched := selector.In(err, opts...)
		if matched {
			f(err)
		}
		return matched
	})
}

/* quick & dirty example within a function
   it only becomes really powerful if used
   in a wider scope
func t() {
	logerr := func(err error) {
		fmt.Println(err)
	}

	someKindaErr := Anonymous()
	someSelector := And(someKindaErr, Grep("sql"))
	otherSelector := Call(logerr, someSelector)

	err := someKindaErr.New("oh man, mysql is acting up again")
	// or
	err = someKindaErr.Wrap(sql.ErrNoRows, "uh oh")

	if someKindaErr.In(err) {
		// handle it manually
	}

	if someSelector.In(err) {
		// still handling it manually
		// it seems to work pretty okay
	}

	// level 2
	if otherSelector.In(err) {
		// error is already logged

		// we can still handle control flow for
		// the local function here
	}
}
*/
