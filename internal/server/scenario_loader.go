package server

import (
	"context"
	"fmt"

	"github.com/like19970403/TRPG-Simulation/internal/realtime"
)

// scenarioLoaderAdapter implements realtime.ScenarioLoader by combining
// SessionRepository and ScenarioRepository to load scenario content for a session.
type scenarioLoaderAdapter struct {
	sessionRepo  SessionRepository
	scenarioRepo ScenarioRepository
}

// LoadScenarioForSession loads and parses the scenario content for a given session.
func (a *scenarioLoaderAdapter) LoadScenarioForSession(ctx context.Context, sessionID string) (*realtime.ScenarioContent, error) {
	gs, err := a.sessionRepo.GetByID(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("scenario loader: get session: %w", err)
	}

	sc, err := a.scenarioRepo.GetByID(ctx, gs.ScenarioID)
	if err != nil {
		return nil, fmt.Errorf("scenario loader: get scenario: %w", err)
	}

	content, err := realtime.ParseScenarioContent(sc.Content)
	if err != nil {
		return nil, fmt.Errorf("scenario loader: parse content: %w", err)
	}

	return content, nil
}
