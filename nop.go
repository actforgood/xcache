// Copyright 2022 Bogdan Constantinescu.
// Use of this source code is governed by an MIT-style
// license that can be found in the LICENSE file or at
// https://github.com/actforgood/xcache/LICENSE.

package xcache

import (
	"context"
	"time"
)

// Nop is a no-operation Cache which does nothing.
// It simply ignores saves and returns ErrNotFound.
type Nop struct{}

// Save does nothing.
func (Nop) Save(context.Context, string, []byte, time.Duration) error {
	return nil
}

// Load returns ErrNotFound.
func (Nop) Load(context.Context, string) ([]byte, error) {
	return nil, ErrNotFound
}

// TTL returns negative TTL.
func (Nop) TTL(context.Context, string) (time.Duration, error) {
	return -1, nil
}

// Stats returns empty Stats object.
func (Nop) Stats(context.Context) (Stats, error) {
	return Stats{}, nil
}
