package errsel

import (
	"fmt"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var badStuff = Anonymous()
var okayStuff = Anonymous()
var goodStuff = Anonymous()

var stuff = Binds(goodStuff, okayStuff, badStuff)

var ErrCause = badStuff.Lift(errors.New("asdfjknaksjdfn"))
var ErrInter = okayStuff.Lift(errors.Wrap(ErrCause, "kfmkljmlmk"))
var ErrSomeErr = goodStuff.Lift(errors.Wrap(errors.Wrap(ErrInter, "fasd"), "lkjfasdkjfj"))

func isCause(err error) bool {
	return errors.Cause(err) == ErrCause
}
func isCauseP(err error) bool {
	return err == ErrCause
}

/*
func isInter(err error) bool {
	return err == ErrInter
}
func isTip(err error) bool {
	return err == ErrSomeErr
}
*/

func TestMoreT(t *testing.T) {
	// generate a router of these lol

	database := Named("database")
	conflict := Bind(database, NamedShadow("conflict"))

	err := conflict.New("oh no err btrees")

	ok, er := database.Traverse(err)
	assert.True(t, ok)
	assert.Equal(t, err, er)

	ok, er = conflict.Traverse(err)
	assert.True(t, ok)
	assert.Equal(t, err, er)

	err = database.New("what about just this? And should just magic it")
	ok, er = database.Traverse(err)
	assert.True(t, ok)
	assert.Equal(t, err, er)

	ok, er = conflict.Traverse(err)
	assert.False(t, ok)
	assert.Equal(t, nil, er)

	t.Log(conflict.New("test, conflict selector works!"))
}

func TestTheT(t *testing.T) {
	a, b, c := Named("internal"), Named("input"), Named("database")
	fmt.Println(a.New("test"))
	fmt.Println(a.Bind(b).New("test"))
	fmt.Println(a.Bind(b).Bind(c).New("test"))
}

var stuffSel = And(goodStuff, okayStuff, badStuff)

func BenchmarkClassManualBindTraverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = stuffSel.Traverse(ErrSomeErr)
	}
}

func BenchmarkClassManualBindAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = And(goodStuff, okayStuff, badStuff)
	}
}

func BenchmarkClassManualBindTraverseAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = And(goodStuff, okayStuff, badStuff).Traverse(ErrSomeErr)
	}
}

func BenchmarkClassAutoBindTraverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = stuff.Traverse(ErrSomeErr)
	}
}

func BenchmarkClassAutoBindAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Binds(goodStuff, okayStuff, badStuff)
	}
}

func BenchmarkClassAutoBindTraverseAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = Binds(goodStuff, okayStuff, badStuff).Traverse(ErrSomeErr)
	}
}

// nested auto
var autoBindNested = Binds(stuff, stuff, stuff, stuff)

func BenchmarkClassNestedAutoBindTraverse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = autoBindNested.Traverse(ErrSomeErr)
	}
}
func BenchmarkClassNestedAutoBindAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		stuf := Binds(goodStuff, okayStuff, badStuff)
		_ = Binds(stuf, stuf, stuf, stuf)
	}
}
func BenchmarkClassNestedAutoBindTraverseAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		stuf := Binds(goodStuff, okayStuff, badStuff)
		_, _ = Binds(stuf, stuf, stuf, stuf).Traverse(ErrSomeErr)
	}
}

// nested auto
var autoBindNestedL = BindsL(stuff, stuff, stuff, stuff)

func BenchmarkClassNestedAutoBindTraverseL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = autoBindNestedL.Traverse(ErrSomeErr)
	}
}
func BenchmarkClassNestedAutoBindAllocL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		stuf := BindsL(goodStuff, okayStuff, badStuff)
		_ = BindsL(stuf, stuf, stuf, stuf)
	}
}
func BenchmarkClassNestedAutoBindTraverseAllocL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		stuf := BindsL(goodStuff, okayStuff, badStuff)
		_, _ = BindsL(stuf, stuf, stuf, stuf).Traverse(ErrSomeErr)
	}
}

func BenchmarkClassTraverse(b *testing.B) {
	sels := []Selector{badStuff, okayStuff, goodStuff}
	for _, sel := range sels {
		b.Run("layers", func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				_, _ = sel.Traverse(ErrSomeErr)
			}
		})
	}
}

func BenchmarkClassIn(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = badStuff.In(ErrSomeErr)
		//x := goodStuff.In(ErrSomeErr)
	}
}

var causesM = Causes(isCauseP)

func BenchmarkCausesM(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// oh, fuck yeah
		// we're getting extremely similar perf,
		// but errsel is calling the function on at least 10 errors
		_ = causesM.In(ErrSomeErr)
	}
}

func BenchmarkCausesMAlloc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Causes(isCauseP).In(ErrSomeErr)
	}
}

func BenchmarkCausesE(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = isCause(ErrSomeErr)
		//_ = isTip(ErrSomeErr)
	}
}
