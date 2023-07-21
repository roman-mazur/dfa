package dfa

import (
	"fmt"
)

type Turnstile struct {
	MinCoins uint

	m Machine[*Turnstile, Action]
	c uint

	coins   chan uint
	pushes  chan struct{}
	stop    chan struct{}
	actions chan Action
}

func (t *Turnstile) On() <-chan Action {
	t.actions = make(chan Action)
	t.coins = make(chan uint)
	t.pushes = make(chan struct{})
	t.stop = make(chan struct{})

	go t.m.Run(locked, t, t.actions, nil)

	return t.actions
}

func (t *Turnstile) Off() {
	close(t.stop)
}

func (t *Turnstile) PutCoins(amount uint) {
	t.coins <- amount
}

func (t *Turnstile) Push() {
	t.pushes <- struct{}{}
}

type Action string

const (
	ActionLock   Action = "lock"
	ActionUnlock Action = "unlock"
	ActionBlock  Action = "block"
)

func ExampleMachine() {
	t := Turnstile{MinCoins: 10}
	actions := t.On()
	done := make(chan struct{})
	go func() {
		for a := range actions {
			fmt.Println(a)
		}
		close(done)
	}()

	t.Push() // Blocked, no coins.
	t.PutCoins(5)
	t.Push() // Blocked, not enough coins.
	t.PutCoins(5)
	t.Push() // Works.
	t.PutCoins(15)
	t.Push() // Works.
	t.Push() // Blocked.

	t.Off()

	<-done

	// Output:
	// block
	// block
	// unlock
	// lock
	// unlock
	// lock
	// block
}

func locked(t *Turnstile) StateFn[*Turnstile] {
	select {
	case a := <-t.coins:
		t.c += a
		if t.c >= t.MinCoins {
			t.actions <- ActionUnlock
			return unlocked
		}
		return locked

	case <-t.pushes:
		t.actions <- ActionBlock
		return locked

	case <-t.stop:
		return nil
	}
}

func unlocked(t *Turnstile) StateFn[*Turnstile] {
	t.c = 0

	select {
	case <-t.pushes:
		t.actions <- ActionLock
		return locked

	case <-t.stop:
		return nil
	}
}
