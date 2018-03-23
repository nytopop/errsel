package errsel

type traverseConfig struct {
	lens  uint
	depth uint
}

func applyTraverseOpts(opts ...TraverseOption) *traverseConfig {
	cfg := new(traverseConfig)
	for _, f := range opts {
		f(cfg)
	}
	return cfg
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

type root func(error) bool

// Root returns a selector that will apply f to an error. If it returns
// true, it will return true and the error f was called with. Otherwise,
// it will return false and nil.
func Root(f func(error) bool) Selector {
	return SelectorFunc(root(f).traverse)
}

func (f root) traverse(err error) (bool, error) {
	if f(err) {
		return true, err
	}
	return false, nil
}

type causes struct {
	f   func(error) bool
	cfg *traverseConfig
}

// Causes returns a selector that will apply f to every intermediate cause
// of an error. The first time f returns true, it will return true and the
// intermediate error that f was called with. Otherwise, it will return false
// and nil.
//
// Traversal of intermediates will be done using an efficient, in-place
// trampoline algorithm with as few allocations as possible.
func Causes(f func(error) bool, opts ...TraverseOption) Selector {
	return SelectorFunc(causes{
		f:   f,
		cfg: applyTraverseOpts(opts...),
	}.traverse)
}

type causer interface {
	Cause() error
}

func (t causes) traverse(err error) (bool, error) {
	cursor := err
	for lens := t.cfg.lens; lens > 0; lens-- {
		if c, ok := err.(causer); ok {
			cursor = c.Cause()
			continue
		}
		break
	}

	for depth := uint(0); depth < t.cfg.depth || t.cfg.depth == 0; depth++ {
		e := cursor
		if t.f(e) {
			return true, e
		}

		c, ok := e.(causer)
		if !ok {
			return false, nil
		}

		cursor = c.Cause()
	}

	return false, nil
}

type classes struct {
	f   func(error) bool
	cfg *traverseConfig
}

// Classes returns a selector that will apply f to every intermediate error
// that has been annotated with a class. The first time f returns true, it
// will return true and the intermediate error that f was called with.
// Otherwise, it will return false and nil.
//
// It will respect class shadowing. A lens can be used to skip past shadowing
// classes, if such behavior is required.
//
// Traversal of intermediates will be done using an efficient, in-place
// trampoline algorithm with as few allocations as possible.
func Classes(f func(error) bool, opts ...TraverseOption) Selector {
	return SelectorFunc(classes{
		f:   f,
		cfg: applyTraverseOpts(opts...),
	}.traverse)
}

func (t classes) traverse(err error) (bool, error) {
	cursor, lensCursor := err, err
	for lens := t.cfg.lens; lens > 0; lens-- {
		if _, ok := lensCursor.(*classErr); ok {
			cursor = lensCursor
		}

		if c, ok := err.(causer); ok {
			lensCursor = c.Cause()
			continue
		}
		break
	}

	var depth uint
	for depth < t.cfg.depth || t.cfg.depth == 0 {
		e := cursor
		if c, ok := e.(*classErr); ok {
			if t.f(e) {
				return true, e
			}

			if c.cls.shadow {
				return false, nil
			}
		}

		c, ok := e.(causer)
		if !ok {
			return false, nil
		}

		cursor = c.Cause()
		depth++
	}

	return false, nil
}
