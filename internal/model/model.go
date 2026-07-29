// Package model defines the core domain types stored in the vault.
package model

import "time"

type Status string

const (
	StatusTodo       Status = "todo"
	StatusInProgress Status = "in_progress"
	StatusBlocked    Status = "blocked"
	StatusDone       Status = "done"
	StatusCancelled  Status = "cancelled"
)

type Source string

const (
	SourceManual Source = "manual"
	SourceAI     Source = "ai"
)

type GoalStatus string

const (
	GoalActive   GoalStatus = "active"
	GoalPaused   GoalStatus = "paused"
	GoalAchieved GoalStatus = "achieved"
	GoalDropped  GoalStatus = "dropped"
)

// Task is a unit of work, optionally tied to a goal and/or a parent task
// (making it a subtask). Stored as one markdown file with YAML frontmatter
// per task.
type Task struct {
	ID       string    `yaml:"id"`
	Title    string    `yaml:"title"`
	GoalID   string    `yaml:"goal_id,omitempty"`
	ParentID string    `yaml:"parent_id,omitempty"`
	Priority int       `yaml:"priority"`
	Status   Status    `yaml:"status"`
	Due      string    `yaml:"due,omitempty"`
	Tags     []string  `yaml:"tags,omitempty"`
	Created  time.Time `yaml:"created"`
	Updated  time.Time `yaml:"updated"`
	Source   Source    `yaml:"source"`

	Body string `yaml:"-"`
	Path string `yaml:"-"`
}

// Goal is a long-horizon outcome the user is working toward.
type Goal struct {
	ID         string     `yaml:"id"`
	Title      string     `yaml:"title"`
	Status     GoalStatus `yaml:"status"`
	TargetDate string     `yaml:"target_date,omitempty"`
	Created    time.Time  `yaml:"created"`

	Body string `yaml:"-"`
	Path string `yaml:"-"`
}

// ContextDoc is background information (bio, CV, constraints, Q&A log)
// fed to the AI when planning or prioritizing.
type ContextDoc struct {
	Title   string    `yaml:"title"`
	Created time.Time `yaml:"created"`

	Body string `yaml:"-"`
	Path string `yaml:"-"`
}

const (
	PriorityHighest = 1
	PriorityLowest  = 5
)
