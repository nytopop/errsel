package errsel

import (
	"fmt"
	"testing"
	"time"

	"github.com/pkg/errors"
)

var ns = []int{4, 8, 16, 32, 128}

func BenchmarkTrampoline(b *testing.B) {
	for _, n := range ns {
		b.Run("N: "+fmt.Sprint(n), func(b *testing.B) {
			b.StopTimer()
			err := errors.New("hello")
			for i := 0; i < n; i++ {
				err = errors.Wrap(err, "world")
				err = errors.Wrap(err, "hello")
			}

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				causes := CausesOf(err)
				_ = causes
			}
		})
	}
}

func causesOf(err error, opts ...TraverseOption) []error {
	cfg := new(traverseConfig)
	for _, f := range opts {
		f(cfg)
	}

	errs := []error{err}
	if c, ok := err.(causer); ok {
		// recurse to find additional causes
		// we don't pass traverse options to recursive calls
		errs = append(errs, causesOf(c.Cause())...)
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

func BenchmarkRecursive(b *testing.B) {
	for _, n := range ns {
		b.Run("N: "+fmt.Sprint(n), func(b *testing.B) {
			b.StopTimer()
			err := errors.New("hello")
			for i := 0; i < n; i++ {
				err = errors.Wrap(err, "world")
				err = errors.Wrap(err, "hello")
			}

			b.StartTimer()
			for i := 0; i < b.N; i++ {
				causes := causesOf(err)
				_ = causes
			}
		})
	}
}

func BenchmarkOr(b *testing.B) {
	for _, n := range ns {
		var css []Selector
		err := errors.New("hello")
		for i := 0; i < n; i++ {
			cs := Anonymous()
			css = append(css, cs)
			err = cs.Wrapc(err)
		}

		selector := Or(css...)

		b.Run("N: "+fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v, ok := selector.Query(err)
				_, _ = v, ok
			}
		})
	}
}

func BenchmarkOrC(b *testing.B) {
	for _, n := range ns {
		var css []Selector
		err := errors.New("hello")
		for i := 0; i < n; i++ {
			cs := Anonymous()
			css = append(css, cs)
			err = cs.Wrapc(err)
		}

		selector := OrC(css...)

		b.Run("N: "+fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v, ok := selector.Query(err)
				_, _ = v, ok
			}
		})
	}
}

func slow(_ error) {
	time.Sleep(5 * time.Millisecond)
}

func BenchmarkOrCSlowIO(b *testing.B) {
	for _, n := range ns {
		var css []Selector
		err := errors.New("hello")
		for i := 0; i < n; i++ {
			cs := Anonymous()
			css = append(css, Call(slow, cs))
			err = cs.Wrapc(err)
		}

		selector := OrC(css...)

		b.Run("N: "+fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v, ok := selector.Query(err)
				_, _ = v, ok
			}
		})
	}
}

func BenchmarkOrSlowIO(b *testing.B) {
	for _, n := range ns {
		var css []Selector
		err := errors.New("hello")
		for i := 0; i < n; i++ {
			cs := Anonymous()
			css = append(css, Call(slow, cs))
			err = cs.Wrapc(err)
		}

		selector := Or(css...)

		b.Run("N: "+fmt.Sprint(n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				v, ok := selector.Query(err)
				_, _ = v, ok
			}
		})
	}
}
