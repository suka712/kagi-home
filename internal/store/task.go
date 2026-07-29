package store

import (
	"fmt"
	"time"

	"hp/internal/model"
)

// CreateTask assigns an ID, timestamps, and a filename to t, then writes it
// to the vault. t is mutated in place with the assigned ID/Path.
func (s *Store) CreateTask(t *model.Task) error {
	now := time.Now()
	t.ID = newID()
	t.Created = now
	t.Updated = now
	if t.Status == "" {
		t.Status = model.StatusTodo
	}
	if t.Priority == 0 {
		t.Priority = 3
	}
	if t.Source == "" {
		t.Source = model.SourceManual
	}
	path, err := uniquePath(s.tasksDir(), slugify(t.Title))
	if err != nil {
		return err
	}
	t.Path = path
	return writeFrontmatterFile(path, t, t.Body)
}

// UpdateTask rewrites t's file with its current in-memory contents, bumping
// Updated. t.Path must already be set (e.g. from GetTask/ListTasks).
func (s *Store) UpdateTask(t *model.Task) error {
	if t.Path == "" {
		return fmt.Errorf("task %s has no path", t.ID)
	}
	t.Updated = time.Now()
	return writeFrontmatterFile(t.Path, t, t.Body)
}

// ListTasks returns all tasks in the vault, sorted by priority then title.
func (s *Store) ListTasks() ([]*model.Task, error) {
	paths, err := listMarkdownFiles(s.tasksDir())
	if err != nil {
		return nil, err
	}
	tasks := make([]*model.Task, 0, len(paths))
	for _, p := range paths {
		var t model.Task
		body, err := readFrontmatterFile(p, &t)
		if err != nil {
			return nil, err
		}
		t.Body = body
		t.Path = p
		tasks = append(tasks, &t)
	}
	sortTasks(tasks)
	return tasks, nil
}

func sortTasks(tasks []*model.Task) {
	for i := 1; i < len(tasks); i++ {
		for j := i; j > 0; j-- {
			a, b := tasks[j-1], tasks[j]
			if a.Priority < b.Priority || (a.Priority == b.Priority && a.Title <= b.Title) {
				break
			}
			tasks[j-1], tasks[j] = tasks[j], tasks[j-1]
		}
	}
}

// GetTask finds a task by exact ID or unique ID prefix.
func (s *Store) GetTask(idOrPrefix string) (*model.Task, error) {
	tasks, err := s.ListTasks()
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(tasks))
	byID := make(map[string]*model.Task, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
		byID[t.ID] = t
	}
	id, err := resolveID(ids, idOrPrefix)
	if err != nil {
		return nil, err
	}
	return byID[id], nil
}

// TaskFilter narrows ListTasks results. Zero values mean "no filter" for
// that field.
type TaskFilter struct {
	GoalID   string
	Status   model.Status
	Priority int
}

func FilterTasks(tasks []*model.Task, f TaskFilter) []*model.Task {
	var out []*model.Task
	for _, t := range tasks {
		if f.GoalID != "" && t.GoalID != f.GoalID {
			continue
		}
		if f.Status != "" && t.Status != f.Status {
			continue
		}
		if f.Priority != 0 && t.Priority != f.Priority {
			continue
		}
		out = append(out, t)
	}
	return out
}

// Children returns the direct subtasks of parentID.
func Children(tasks []*model.Task, parentID string) []*model.Task {
	var out []*model.Task
	for _, t := range tasks {
		if t.ParentID == parentID {
			out = append(out, t)
		}
	}
	return out
}
