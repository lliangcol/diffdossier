package results

import (
	"testing"
	"time"

	"github.com/lliangcol/diffdossier/internal/planner"
)

func TestResultIndexIsImmutableAndRequiresEveryPerspective(t *testing.T) {
	plan := planner.Plan{Tasks: []planner.Task{{
		ID: "task-a", RequiredPasses: 2, Perspectives: []string{"correctness", "failure-recovery"},
	}}}
	index := Index{}
	one := fixtureResult()
	one.Reviewer.PassID = "one"
	one.Reviewer.Perspective = "correctness"
	var err error
	index, err = Append(index, one, Validation{Accepted: true, Completed: true}, ResultPath(one.TaskID, one.Reviewer.PassID), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if ReviewComplete(index, plan) {
		t.Fatal("one of two perspectives must not complete review")
	}
	if err := VerifyRecord(index.Records[0], one, Validation{Accepted: true, Completed: true}); err != nil {
		t.Fatal(err)
	}
	tampered := index.Records[0]
	tampered.Completed = false
	if err := VerifyRecord(tampered, one, Validation{Accepted: true, Completed: true}); err == nil {
		t.Fatal("tampered result index record was accepted")
	}
	if _, err := Append(index, one, Validation{Accepted: true, Completed: true}, "duplicate", time.Now()); err == nil {
		t.Fatal("duplicate pass_id was accepted")
	}
	two := one
	two.Reviewer.PassID = "two"
	two.Reviewer.Perspective = "failure-recovery"
	two.Reviewer.ContextIsolation = "fresh manual review two"
	index, err = Append(index, two, Validation{Accepted: true, Completed: true}, ResultPath(two.TaskID, two.Reviewer.PassID), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !ReviewComplete(index, plan) {
		t.Fatal("all required perspectives should complete review")
	}
}
