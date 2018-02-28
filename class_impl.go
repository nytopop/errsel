package errsel

type class struct {
	named  bool
	name   string
	shadow bool
}

// Anonymous returns an anonymous class.
//
// When used as a selector, it will match only against itself. Making
// a copy will effectively create a new anonymous class distinct from
// the original.
//
// Due to its dependence on an address comparison, it should probably
// not cross package boundaries.
func Anonymous() Class {
	c := &class{}
	return ToClass(LifterFunc(c.lift), Classes(c.in))
}

// Named returns a named class.
//
// When used as a selector, it will match against any other named
// class with exactly the same name.
func Named(name string) Class {
	c := &class{
		named: true,
		name:  name,
	}
	return ToClass(LifterFunc(c.lift), Classes(c.in))
}

// AnonymousShadow returns an anonymous, shadowing class. Wrapping
// an error with it will hide any deeper class definitions in the
// error's context chain. This can be useful if you need to logically
// segment internal and external errors.
//
// When used as a selector, it will match only against itself. The same
// copying restrictions as Anonymous apply.
func AnonymousShadow() Class {
	c := &class{
		shadow: true,
	}
	return ToClass(LifterFunc(c.lift), Classes(c.in))
}

// NamedShadow returns a named, shadowing class. Wrapping an error
// with it will hide any deeper class definitions in the error's
// context chain. This can be useful if you need to logically segment
// internal and external errors across package boundaries.
//
// When used as a selector, it will match against any other named
// class with exactly the same name.
func NamedShadow(name string) Class {
	c := &class{
		named:  true,
		name:   name,
		shadow: true,
	}
	return ToClass(LifterFunc(c.lift), Classes(c.in))
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
	if err == nil {
		return nil
	}

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
	return c.err.Error()
}

func (c *classErr) Cause() error {
	return c.err
}
