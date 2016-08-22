package throttler

import (
	"testing"
	"time"
)

var (
	resultIncreased = result{
		Now:                    sinceZero(1234 * time.Millisecond),
		RateChange:             increasedRate,
		lastRateChange:         sinceZero(1 * time.Millisecond),
		OldState:               stateIncreaseRate,
		TestedState:            stateIncreaseRate,
		NewState:               stateIncreaseRate,
		OldRate:                100,
		NewRate:                100,
		Reason:                 "increased the rate",
		CurrentRate:            99,
		GoodOrBad:              goodRate,
		MemorySkipReason:       "",
		HighestGood:            95,
		LowestBad:              0,
		LagRecordNow:           lagRecord(sinceZero(1234*time.Millisecond), 101, 1),
		LagRecordBefore:        replicationLagRecord{},
		MasterRate:             99,
		GuessedSlaveRate:       0,
		GuessedSlaveBacklogOld: 0,
		GuessedSlaveBacklogNew: 0,
	}
	resultDecreased = result{
		Now:                    sinceZero(5000 * time.Millisecond),
		RateChange:             decreasedRate,
		lastRateChange:         sinceZero(1234 * time.Millisecond),
		OldState:               stateIncreaseRate,
		TestedState:            stateDecreaseAndGuessRate,
		NewState:               stateDecreaseAndGuessRate,
		OldRate:                200,
		NewRate:                100,
		Reason:                 "decreased the rate",
		CurrentRate:            200,
		GoodOrBad:              badRate,
		MemorySkipReason:       "",
		HighestGood:            95,
		LowestBad:              200,
		LagRecordNow:           lagRecord(sinceZero(5000*time.Millisecond), 101, 2),
		LagRecordBefore:        lagRecord(sinceZero(1234*time.Millisecond), 101, 1),
		MasterRate:             200,
		GuessedSlaveRate:       150,
		GuessedSlaveBacklogOld: 10,
		GuessedSlaveBacklogNew: 20,
	}
	resultEmergency = result{
		Now:                    sinceZero(10123 * time.Millisecond),
		RateChange:             decreasedRate,
		lastRateChange:         sinceZero(5000 * time.Millisecond),
		OldState:               stateDecreaseAndGuessRate,
		TestedState:            stateEmergency,
		NewState:               stateEmergency,
		OldRate:                100,
		NewRate:                50,
		Reason:                 "emergency state decreased the rate",
		CurrentRate:            100,
		GoodOrBad:              badRate,
		MemorySkipReason:       "",
		HighestGood:            95,
		LowestBad:              100,
		LagRecordNow:           lagRecord(sinceZero(10123*time.Millisecond), 101, 23),
		LagRecordBefore:        lagRecord(sinceZero(5000*time.Millisecond), 101, 2),
		MasterRate:             0,
		GuessedSlaveRate:       0,
		GuessedSlaveBacklogOld: 0,
		GuessedSlaveBacklogNew: 0,
	}
)

func TestResultString(t *testing.T) {
	testcases := []struct {
		r    result
		want string
	}{
		{
			resultIncreased,
			`rate was: increased from: 100 to: 100
alias: cell1-0000000101 lag: 1s
last change: 1.2s rate: 99 good/bad? good skipped b/c:  good/bad: 95/0
state (old/tested/new): I/I/I 
lag before: n/a (n/a ago) rates (master/slave): 99/0 backlog (old/new): 0/0
reason: increased the rate`,
		},
		{
			resultDecreased,
			`rate was: decreased from: 200 to: 100
alias: cell1-0000000101 lag: 2s
last change: 3.8s rate: 200 good/bad? bad skipped b/c:  good/bad: 95/200
state (old/tested/new): I/D/D 
lag before: 1s (3.8s ago) rates (master/slave): 200/150 backlog (old/new): 10/20
reason: decreased the rate`,
		},
		{
			resultEmergency,
			`rate was: decreased from: 100 to: 50
alias: cell1-0000000101 lag: 23s
last change: 5.1s rate: 100 good/bad? bad skipped b/c:  good/bad: 95/100
state (old/tested/new): D/E/E 
lag before: 2s (5.1s ago) rates (master/slave): 0/0 backlog (old/new): 0/0
reason: emergency state decreased the rate`,
		},
	}

	for _, tc := range testcases {
		got := tc.r.String()
		if got != tc.want {
			t.Fatalf("record.String() = %v, want = %v for full record: %#v", got, tc.want, tc.r)
		}
	}
}
