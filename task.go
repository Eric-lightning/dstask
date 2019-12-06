package dstask

// main task data structures

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

type SubTask struct {
	Summary  string
	Resolved bool
}

type Task struct {
	// not stored in file -- rather filename and directory
	UUID   string `yaml:"-"`
	Status string `yaml:"-"`
	// is new or has changed. Need to write to disk.
	WritePending bool `yaml:"-"`

	// ephemeral, used to address tasks quickly. Non-resolved only.
	ID int `yaml:",omitempty"`

	// concise representation of task
	Summary string
	// more detail, or information to remember to complete the task
	Notes   string
	Tags    []string
	Project string
	// see const.go for PRIORITY_ strings
	Priority    string
	DelegatedTo string
	Subtasks    []SubTask
	// uuids of tasks that this task depends on
	// blocked status can be derived.
	// TODO possible filter: :blocked. Also, :overdue
	Dependencies []string

	Created  time.Time
	Resolved time.Time
	Due      time.Time

	// only valid for recurring tasks, and resolved recurring tasks. Tasks
	// created by a recurring task should have this removed or will fail
	// validation. syntax: cron.
	Schedule string `yaml:"omitempty"`
	// task this was derived from. At the moment this means a recurring task
	// but could also be a template task in future
	Parent   string `yaml:"omitempty"`
}

func (task Task) String() string {
	if task.ID > 0 {
		return fmt.Sprintf("%v: %s", task.ID, task.Summary)
	} else {
		return task.Summary
	}
}

// used for applying a context to a new task
func (cmdLine *CmdLine) MergeContext(context CmdLine) {
	for _, tag := range context.Tags {
		if !StrSliceContains(cmdLine.Tags, tag) {
			cmdLine.Tags = append(cmdLine.Tags, tag)
		}
	}

	for _, tag := range context.AntiTags {
		if !StrSliceContains(cmdLine.AntiTags, tag) {
			cmdLine.AntiTags = append(cmdLine.AntiTags, tag)
		}
	}

	// TODO same for antitags
	if context.Project != "" {
		if cmdLine.Project != "" && cmdLine.Project != context.Project {
			ExitFail("Could not apply context, project conflict")
		} else {
			cmdLine.Project = context.Project
		}
	}

	if context.Priority != "" {
		if cmdLine.Priority != "" {
			ExitFail("Could not apply context, priority conflict")
		} else {
			cmdLine.Priority = context.Priority
		}
	}
}

func (task *Task) MatchesFilter(cmdLine CmdLine) bool {
	for _, id := range cmdLine.IDs {
		if id == task.ID {
			return true
		}
	}

	// IDs were specified but no match
	if len(cmdLine.IDs) > 0 {
		return false
	}

	for _, tag := range cmdLine.Tags {
		if !StrSliceContains(task.Tags, tag) {
			return false
		}
	}

	for _, tag := range cmdLine.AntiTags {
		if StrSliceContains(task.Tags, tag) {
			return false
		}
	}

	if StrSliceContains(cmdLine.AntiProjects, task.Project) {
		return false
	}

	if cmdLine.Project != "" && task.Project != cmdLine.Project {
		return false
	}

	if cmdLine.Priority != "" && task.Priority != cmdLine.Priority {
		return false
	}

	if cmdLine.Text != "" && !strings.Contains(strings.ToLower(task.Summary+task.Notes), strings.ToLower(cmdLine.Text)) {
		return false
	}

	return true
}

func (task *Task) Normalise() {
	task.Project = strings.ToLower(task.Project)

	// tags must be lowercase
	for i, tag := range task.Tags {
		task.Tags[i] = strings.ToLower(tag)
	}

	// tags must be sorted
	sort.Strings(task.Tags)

	// tags must be unique
	task.Tags = DeduplicateStrings(task.Tags)

	if task.Status == STATUS_RESOLVED {
		// resolved task should not have ID as it's meaningless
		task.ID = 0
	}

	if task.Priority == "" {
		task.Priority = PRIORITY_NORMAL
	}
}

// normalise the task before validating!
func (task *Task) Validate() error {
	if !IsValidUUID4String(task.UUID) {
		return errors.New("Invalid task UUID4")
	}

	if !IsValidStatus(task.Status) {
		return errors.New("Invalid status specified on task")
	}

	if !IsValidPriority(task.Priority) {
		return errors.New("Invalid priority specified")
	}

	for _, uuid := range task.Dependencies {
		if !IsValidUUID4String(uuid) {
			return errors.New("Invalid dependency UUID4")
		}
	}

	return nil
}

// provides Summary + Last note if available
func (task *Task) LongSummary() string {
	noteLines := strings.Split(task.Notes, "\n")
	lastNote := noteLines[len(noteLines)-1]

	if len(lastNote) > 0 {
		return task.Summary + " " + NOTE_MODE_KEYWORD + " " + lastNote
	} else {
		return task.Summary
	}
}
