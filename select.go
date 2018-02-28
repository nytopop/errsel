package errsel

import (
	"reflect"
	"strings"
	"sync"
)

// Selector is an interface for matching errors.
//
// A function f of the signature:
//
//    var f func(error) bool
//
// Can be converted into a selector by wrapping it with an appropriate
// traversal type:
//
//    var s Selector
//    // calls f with original error
//    s = Root(f)
//    // calls f with every intermediate error
//    s = Causes(f)
//    // calls f with every intermediate annotated error
//    s = Classes(f)
//
// A function g of the signature:
//
//    var g func(error) (bool, error)
//
// Can be converted directly into a selector by wrapping it in a
// SelectorFunc (preserving traversal behavior of g):
//
//    var s Selector
//    s = SelectorFunc(g)
//
type Selector interface {
	Traverse(err error) (bool, error)
	In(err error) bool
	Is(err error) error
}

var _ Selector = new(SelectorFunc)

type SelectorFunc func(error) (bool, error)

func (f SelectorFunc) Traverse(err error) (bool, error) {
	return f(err)
}

func (f SelectorFunc) In(err error) bool {
	ok, _ := f(err)
	return ok
}

func (f SelectorFunc) Is(err error) error {
	_, er := f(err)
	return er
}

// And returns a selector that will only match if all input selectors
// match. It will always return the error it was called with on a match,
// and nil otherwise.
func And(ss ...Selector) Selector {
	return Root(func(err error) bool {
		accum := true
		for _, s := range ss {
			ok, _ := s.Traverse(err)
			accum = accum && ok
		}
		return accum
	})
}

// AndC behaves like And, except that input selectors will be evaluated
// concurrently.
func AndC(ss ...Selector) Selector {
	return Root(func(err error) bool {
		var (
			accum = true
			mu    sync.Mutex
			wg    sync.WaitGroup
		)
		for _, s := range ss {
			wg.Add(1)
			go func(s Selector) {
				ok, _ := s.Traverse(err)
				mu.Lock()
				accum = accum && ok
				mu.Unlock()
				wg.Done()
			}(s)
		}

		wg.Wait()
		return accum
	})
}

// Or returns a selector that will match if any of the input selectors
// match. It will always return the error it was called with on a match,
// and nil otherwise.
func Or(ss ...Selector) Selector {
	return Root(func(err error) bool {
		var accum bool
		for _, s := range ss {
			ok, _ := s.Traverse(err)
			accum = accum || ok
		}
		return accum
	})
}

// OrC behaves like Or, except that input selectors will be evaluated
// concurrently.
func OrC(ss ...Selector) Selector {
	return Root(func(err error) bool {
		var (
			accum bool
			mu    sync.Mutex
			wg    sync.WaitGroup
		)
		for _, s := range ss {
			wg.Add(1)
			go func(s Selector) {
				ok, _ := s.Traverse(err)
				mu.Lock()
				accum = accum || ok
				mu.Unlock()
				wg.Done()
			}(s)
		}
		wg.Wait()
		return accum
	})
}

// Not returns a selector that will invert the input selector's result.
func Not(s Selector) Selector {
	// given: f(err) bool, error
	// we want to return an f(err) bool, error
	// that inverts the bool
	// we don't want to mess with the error output
	return Root(func(err error) bool {
		return !s.In(err)
	})
}

// Error returns a selector that will match if the provided error occurs
// anywhere in an error's context chain.
//
// Any provided traverse options will scope to causes.
func Error(err error, opts ...TraverseOption) Selector {
	return Causes(func(er error) bool {
		if err == er {
			return true
		}
		return false
	}, opts...)
}

// Type returns a selector that will match if the provided type occurs
// anywhere in an error's context chain.
//
// Any provided traverse options will scope to causes.
func Type(t interface{}, opts ...TraverseOption) Selector {
	T := reflect.TypeOf(t)
	return Causes(func(err error) bool {
		if reflect.TypeOf(err) == T {
			return true
		}
		return false
	}, opts...)
}

// Grep returns a selector that will match if the provided string is a
// substring in an error's concatenated Error() output.
func Grep(str string) Selector {
	return Root(func(err error) bool {
		idx := strings.Index(err.Error(), str)
		return idx != -1
	})
}

// Call returns a selector that will call the provided function if the
// provided selector matches.
//
// This can be useful if a certain error condition needs to be handled
// a certain way in all cases; using a Call selector means it becomes
// impossible to forget to do this. For example, a Call might be used
// to execute a log function.
func Call(f func(error), s Selector) Selector {
	return Root(func(err error) bool {
		ok, er := s.Traverse(err)
		if ok {
			f(er)
		}
		return ok
	})
}

// Once is an idempotent alternative to Call. For any (n > 0) times that
// the returned selector has matched, the provided f is guaranteed to have
// executed exactly once.
func Once(f func(error), s Selector) Selector {
	var once sync.Once
	return Root(func(err error) bool {
		ok, er := s.Traverse(err)
		if ok {
			once.Do(func() {
				f(er)
			})
		}
		return ok
	})
}

// Mask returns a selector that masks the provided selector with a SelectorFunc
// wrapper. This can be useful if you want to export a class for use as a
// selector, while disallowing the creation of new error instances.
func Mask(s Selector) Selector {
	return SelectorFunc(s.Traverse)
}
