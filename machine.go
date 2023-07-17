package dfa

import (
	"reflect"
	"runtime"
	"time"
)

type Machine[T any, Out any] struct {
	Transformer func(fn StateFn[T]) StateFn[T]
}

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

	close(out)
	if statsOut != nil {
		close(statsOut)
	}
}

type StateFn[T any] func(T) StateFn[T]

type StateStats struct {
	Name             string
	EntryCount       uint
	TimeToFirstEntry time.Duration
	TotalTimeSpent   time.Duration
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
