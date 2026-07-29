package store

import (
	"fmt"
	"time"

	"hp/internal/model"
)

func (s *Store) CreateGoal(g *model.Goal) error {
	g.ID = newID()
	g.Created = time.Now()
	if g.Status == "" {
		g.Status = model.GoalActive
	}
	path, err := uniquePath(s.goalsDir(), slugify(g.Title))
	if err != nil {
		return err
	}
	g.Path = path
	return writeFrontmatterFile(path, g, g.Body)
}

func (s *Store) UpdateGoal(g *model.Goal) error {
	if g.Path == "" {
		return fmt.Errorf("goal %s has no path", g.ID)
	}
	return writeFrontmatterFile(g.Path, g, g.Body)
}

func (s *Store) ListGoals() ([]*model.Goal, error) {
	paths, err := listMarkdownFiles(s.goalsDir())
	if err != nil {
		return nil, err
	}
	goals := make([]*model.Goal, 0, len(paths))
	for _, p := range paths {
		var g model.Goal
		body, err := readFrontmatterFile(p, &g)
		if err != nil {
			return nil, err
		}
		g.Body = body
		g.Path = p
		goals = append(goals, &g)
	}
	return goals, nil
}

func (s *Store) GetGoal(idOrPrefix string) (*model.Goal, error) {
	goals, err := s.ListGoals()
	if err != nil {
		return nil, err
	}
	ids := make([]string, len(goals))
	byID := make(map[string]*model.Goal, len(goals))
	for i, g := range goals {
		ids[i] = g.ID
		byID[g.ID] = g
	}
	id, err := resolveID(ids, idOrPrefix)
	if err != nil {
		return nil, err
	}
	return byID[id], nil
}
