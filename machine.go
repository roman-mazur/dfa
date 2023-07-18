package dfa

import (
	"reflect"
	"runtime"
	"time"
)

// Machine can be used to run a deterministic state automaton expressed in a form of state functions (see StateFn).
// Its Run method is typically used to launch a new go routine that executed the state functions.
// For example:
//
//	var m dfa.Machine
//	go m.Run(initState, myFSM, output, nil)
type Machine[T any, Out any] struct {
	// An optional state function transformer.
	// For instance, it can be used to implement logging of all the encountered threads.
	Transformer func(fn StateFn[T]) StateFn[T]
}

// Run starts executing the state functions beginning from the provided one.
// out and statsOut channels can be nil.
// When out is not nil, it's closed by this method before returning, after all the state machine reaches the terminal state.
// When statsOut is not nil, it's used to emit information about the encountered states (see StateStats).
func (m *Machine[T, Out]) Run(start StateFn[T], x T, out chan<- Out, statsOut chan<- StateStats) {
	tf := m.Transformer
	if tf == nil {
		tf = identity[T]
	}

	stats := make(map[string]StateStats, 10)

	startTime := time.Now()
	for state := start; state != nil; {
		name := stateFuncName(state)
		ss := stats[name]
		ss.Name = name
		ss.EntryCount++
		if ss.EntryCount == 1 {
			ss.TimeToFirstEntry = time.Since(startTime)
		}
		ss.emit(statsOut)

		stateStartTime := time.Now()
		state = tf(state)(x)
		ss.TotalTimeSpent += time.Since(stateStartTime)

		ss.emit(statsOut)
		stats[name] = ss
	}

	if statsOut != nil {
		close(statsOut)
	}
	if out != nil {
		close(out)
	}
}

// StateFn performs actions associated with a state represented by the function
// and returns the next state to transition to. Returning nil means reaching the terminal state finishing the execution
// that was started by Machine.Run.
type StateFn[T any] func(T) StateFn[T]

// StateStats encapsulates the statistics accumulated about the particular state.
type StateStats struct {
	Name             string        // State function name. Retrieved using the reflect package.
	EntryCount       uint          // Number of times the state was reached.
	TimeToFirstEntry time.Duration // Amount of time passed since the start of Machine.Run until entering the state.
	TotalTimeSpent   time.Duration // Cumulative amount of time spent in the corresponding state function.
}

func (ss StateStats) emit(out chan<- StateStats) {
	if out != nil {
		out <- ss
	}
}

func stateFuncName(f any) string {
	v := reflect.ValueOf(f)
	pkgPathLen := len(v.Type().PkgPath())
	fName := runtime.FuncForPC(v.Pointer()).Name()
	return fName[pkgPathLen+1:]
}

func identity[T any](fn StateFn[T]) StateFn[T] {
	return fn
}
