package app

import "testing"

func TestLimitTaskListCapsResponse(t *testing.T) {
	tasks := make([]*Task, maxTaskListResponseTasks+25)
	for i := range tasks {
		tasks[i] = &Task{ID: newID("forest")}
	}
	got := limitTaskList(tasks)
	if len(got) != maxTaskListResponseTasks {
		t.Fatalf("len = %d, want %d", len(got), maxTaskListResponseTasks)
	}
	if got[0] != tasks[0] || got[len(got)-1] != tasks[maxTaskListResponseTasks-1] {
		t.Fatal("limitTaskList did not preserve leading tasks")
	}
}

func TestLimitTaskListKeepsSmallList(t *testing.T) {
	tasks := []*Task{{ID: "a"}, {ID: "b"}}
	got := limitTaskList(tasks)
	if len(got) != len(tasks) {
		t.Fatalf("len = %d, want %d", len(got), len(tasks))
	}
}
