package firewx

import (
	"bytes"
	"encoding/json"
	"fmt"
)

// Opt holds a value that may be absent.
//
// Missing data is the normal case, not an error case: RAWS drop hours
// routinely, consumer weather stations drop them more often, and a sensor can
// fail independently of the rest of the station. The alternative of signalling
// absence with NaN is a trap here, because NaN propagates silently through the
// finite-difference loop in the Nelson dead fuel moisture model and produces
// output that looks plausible but is not.
//
// Opt marshals to JSON as either the bare value or null, so persisted
// observations and model state round-trip through a jsonb column without a
// wrapper object.
type Opt[T any] struct {
	val T
	ok  bool
}

// Some returns an Opt holding v.
func Some[T any](v T) Opt[T] { return Opt[T]{val: v, ok: true} }

// None returns an empty Opt.
func None[T any]() Opt[T] { return Opt[T]{} }

// Get returns the value and whether it is present.
func (o Opt[T]) Get() (T, bool) { return o.val, o.ok }

// Valid reports whether a value is present.
func (o Opt[T]) Valid() bool { return o.ok }

// Or returns the value if present, otherwise def.
func (o Opt[T]) Or(def T) T {
	if o.ok {
		return o.val
	}
	return def
}

// Must returns the value, panicking if it is absent. Intended for tests and
// for call sites that have already checked Valid.
func (o Opt[T]) Must() T {
	if !o.ok {
		var zero T
		panic(fmt.Sprintf("firewx: Opt[%T] is empty", zero))
	}
	return o.val
}

// MapOpt applies fn to the value of o if present. It is a free function rather
// than a method because Go does not permit additional type parameters on
// methods.
func MapOpt[T, U any](o Opt[T], fn func(T) U) Opt[U] {
	if !o.ok {
		return None[U]()
	}
	return Some(fn(o.val))
}

// MarshalJSON encodes a present value as itself and an absent value as null.
func (o Opt[T]) MarshalJSON() ([]byte, error) {
	if !o.ok {
		return []byte("null"), nil
	}
	return json.Marshal(o.val)
}

// UnmarshalJSON decodes null as an absent value and any other JSON as a present
// value.
func (o *Opt[T]) UnmarshalJSON(b []byte) error {
	if bytes.Equal(bytes.TrimSpace(b), []byte("null")) {
		var zero T
		o.val, o.ok = zero, false
		return nil
	}
	if err := json.Unmarshal(b, &o.val); err != nil {
		return err
	}
	o.ok = true
	return nil
}
