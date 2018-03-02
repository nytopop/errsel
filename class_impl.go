package errsel

type class struct {
	named  bool
	name   string
	shadow bool
}

// Anonymous returns an anonymous class.
//
// When used as a selector, it will match only against itself.
//
// Due to its dependence on an address comparison, it should probably
// not cross package boundaries.
func Anonymous() Class {
	return (&class{}).toClass()
}

// Named returns a named class.
//
// When used as a selector, it will match against any other named
// class with exactly the same name.
func Named(name string) Class {
	return (&class{
		named: true,
		name:  name,
	}).toClass()
}

// AnonymousShadow returns an anonymous, shadowing class. Wrapping
// an error with it will hide any deeper class definitions in the
// error's context chain. This can be useful if you need to logically
// segment internal and external errors.
//
// When used as a selector, it will match only against itself.
func AnonymousShadow() Class {
	return (&class{
		shadow: true,
	}).toClass()
}

// NamedShadow returns a named, shadowing class. Wrapping an error
// with it will hide any deeper class definitions in the error's
// context chain. This can be useful if you need to logically segment
// internal and external errors across package boundaries.
//
// When used as a selector, it will match against any other named
// class with exactly the same name.
func NamedShadow(name string) Class {
	return (&class{
		named:  true,
		name:   name,
		shadow: true,
	}).toClass()
}

func (e *class) toClass() Class {
	return ToClass(LifterFunc(e.lift), Classes(e.in))
}

func (e *class) in(err error) bool {
	if c, ok := err.(*classErr); ok {
		if c.cls == e {
			return true
		}

		if c.cls.named && e.named {
			if c.cls.name == e.name {
				return true
			}
		}
	}
	return false
}

func (e *class) lift(err error) error {
	return &classErr{
		cls: e,
		err: err,
	}
}

type classErr struct {
	cls *class
	err error
}

func (c *classErr) Error() string {
	var shad string
	if c.cls.shadow {
		shad = "#"
	}

	if c.cls.named {
		return c.cls.name + shad + "{ " + c.err.Error() + " }"
	}

	return c.err.Error()
}

func (c *classErr) Cause() error {
	return c.err
}
