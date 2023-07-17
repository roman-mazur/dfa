package dfa

import (
	"reflect"
	"strconv"
	"testing"
)

func TestMachine_Run(t *testing.T) {
	var transformExample example
	transformExample.m.Transformer = transformExample.countTransformer

	var fullLoop loopExample
	fullLoop.initLoop = true

	for _, tc := range []struct {
		name string
		ex   stateMachineExample
	}{
		{name: "basic", ex: new(example)},
		{name: "transform", ex: &transformExample},
		{name: "loop", ex: new(loopExample)},
		{name: "loop init", ex: &fullLoop},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			<-tc.ex.Start(t)
			tc.ex.Verify(t)
		})
	}
}

type stateMachineExample interface {
	Start(t *testing.T) (done chan struct{})
	Verify(t *testing.T)
}

type example struct {
	m       Machine[*example, struct{}]
	records []string
	cnt     int

	statsData      map[string]StateStats
	statsUpdateCnt int
}

func (e *example) Start(_ *testing.T) chan struct{} {
	e.statsData = make(map[string]StateStats)

	done := make(chan struct{})
	stats := make(chan StateStats)

	finalDone := make(chan struct{})
	go func() {
		for s := range stats {
			e.statsUpdateCnt++
			e.statsData[s.Name] = s
		}
		<-done
		close(finalDone)
	}()
	go e.m.Run(stateOne, e, done, stats)
	return finalDone
}

func (e *example) Verify(t *testing.T) {
	t.Log("Records: ", e.records)
	if e.m.Transformer == nil {
		if !reflect.DeepEqual(e.records, []string{"stateOne", "stateTwo"}) {
			t.Error("unexpected records: ", e.records)
		}
	} else {
		if !reflect.DeepEqual(e.records, []string{"transform1", "stateOne", "transform2", "stateTwo"}) {
			t.Error("unexpected records: ", e.records)
		}
	}
	if e.statsUpdateCnt != 4 {
		t.Error("wrong number of stats updates: ", e.statsUpdateCnt)
	}

	t.Log(e.statsData)
	if len(e.statsData) != 2 {
		t.Error("wrong stats data size", e.statsData)
	} else {
		oneStats := e.statsData["stateOne"]
		if oneStats.Name != "stateOne" {
			t.Error("wrong state name in the stats")
		}
		if oneStats.EntryCount != 1 {
			t.Error("wrong entry count")
		}
		if oneStats.TotalTimeSpent == 0 {
			t.Error("total time not measured")
		}
		if oneStats.TimeToFirstEntry == 0 {
			t.Error("time to the first entry not measured")
		}
	}
}

func (e *example) countTransformer(fn StateFn[*example]) StateFn[*example] {
	e.cnt++
	e.records = append(e.records, "transform"+strconv.Itoa(e.cnt))
	return fn
}

func stateOne(s *example) StateFn[*example] {
	s.records = append(s.records, "stateOne")
	return stateTwo
}

func stateTwo(s *example) StateFn[*example] {
	s.records = append(s.records, "stateTwo")
	return nil
}

type loopExample struct {
	m Machine[*loopExample, struct{}]

	cnt      int
	initLoop bool

	stats map[string]StateStats
}

func (le *loopExample) Start(t *testing.T) chan struct{} {
	done := make(chan struct{})
	statsCh := make(chan StateStats)

	le.stats = make(map[string]StateStats)
	go func() {
		for s := range statsCh {
			t.Log(s)
			le.stats[s.Name] = s
		}
		close(done)
	}()
	le.m.Run(stateInit, le, make(chan struct{}), statsCh)
	return done
}

func (le *loopExample) Verify(t *testing.T) {
	if le.stats["stateLoop"].EntryCount != 3 {
		t.Error("wrong number of entries for stateLoop")
	}

	initEntries := uint(1)
	if le.initLoop {
		initEntries = 3
	}
	if le.stats["stateInit"].EntryCount != initEntries {
		t.Error("wrong number of entries for stateInit")

	}
}

func stateInit(_ *loopExample) StateFn[*loopExample] {
	return stateLoop
}

func stateLoop(le *loopExample) StateFn[*loopExample] {
	le.cnt++
	if le.cnt == 3 {
		return nil
	}
	if le.initLoop {
		return stateInit
	}
	return stateLoop
}
