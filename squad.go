package rules

import (
	"errors"
)

type SquadRuleset struct {
	StandardRuleset

	SquadMap map[string]string

	// These are intentionally designed so that they default to a standard game.
	AllowBodyCollisions bool
	SharedElimination   bool
	SharedHealth        bool
	SharedLength        bool
}

func (r *SquadRuleset) Name() string { return GameTypeSquad }

func (r SquadRuleset) Pipeline() (*Pipeline, error) {
	return NewPipeline(
		"snake.movement.standard",
		"health.reduce.standard",
		"hazard.damage.standard",
		"snake.eatfood.standard",
		"food.spawn.standard",
		"snake.eliminate.standard",
		"snake.collision.squad",
		"snake.share.squad",
		"gameover.squad",
	)
}

func (r *SquadRuleset) CreateNextBoardState(prevState *BoardState, moves []SnakeMove) (*BoardState, error) {
	p, err := r.Pipeline()
	if err != nil {
		return nil, err
	}
	_, nextState, err := p.Execute(prevState, r.Settings(), moves)

	return nextState, err
}

func areSnakesOnSameSquad(squadMap map[string]string, snake *Snake, other *Snake) bool {
	return areSnakeIDsOnSameSquad(squadMap, snake.ID, other.ID)
}

func areSnakeIDsOnSameSquad(squadMap map[string]string, snakeID string, otherID string) bool {
	return squadMap[snakeID] == squadMap[otherID]
}

func ResurrectSnakesSquad(b *BoardState, settings Settings, moves []SnakeMove) (bool, error) {
	if IsInitialisation(b, settings, moves) {
		return false, nil
	}
	if !settings.SquadSettings.AllowBodyCollisions {
		return false, nil
	}

	for i := 0; i < len(b.Snakes); i++ {
		snake := &b.Snakes[i]
		if snake.EliminatedCause == EliminatedByCollision {
			if snake.EliminatedBy == "" {
				return false, errors.New("snake eliminated by collision and eliminatedby is not set")
			}
			if snake.ID != snake.EliminatedBy && areSnakeIDsOnSameSquad(settings.SquadSettings.squadMap, snake.ID, snake.EliminatedBy) {
				snake.EliminatedCause = NotEliminated
				snake.EliminatedBy = ""
			}
		}
	}

	return false, nil
}

func ShareAttributesSquad(b *BoardState, settings Settings, moves []SnakeMove) (bool, error) {
	if IsInitialisation(b, settings, moves) {
		return false, nil
	}
	squadSettings := settings.SquadSettings

	if !(squadSettings.SharedElimination || squadSettings.SharedLength || squadSettings.SharedHealth) {
		return false, nil
	}

	for i := 0; i < len(b.Snakes); i++ {
		snake := &b.Snakes[i]
		if snake.EliminatedCause != NotEliminated {
			continue
		}

		for j := 0; j < len(b.Snakes); j++ {
			other := &b.Snakes[j]
			if areSnakesOnSameSquad(squadSettings.squadMap, snake, other) {
				if squadSettings.SharedHealth {
					if snake.Health < other.Health {
						snake.Health = other.Health
					}
				}
				if squadSettings.SharedLength {
					if len(snake.Body) == 0 || len(other.Body) == 0 {
						return false, errors.New("found snake of zero length")
					}
					for len(snake.Body) < len(other.Body) {
						growSnake(snake)
					}
				}
				if squadSettings.SharedElimination {
					if snake.EliminatedCause == NotEliminated && other.EliminatedCause != NotEliminated {
						snake.EliminatedCause = EliminatedBySquad
						// We intentionally do not set snake.EliminatedBy because there might be multiple culprits.
						snake.EliminatedBy = ""
					}
				}
			}
		}
	}

	return false, nil
}

func (r *SquadRuleset) IsGameOver(b *BoardState) (bool, error) {
	return r.callStageFunc(GameOverSquad, b, []SnakeMove{})
}

func GameOverSquad(b *BoardState, settings Settings, moves []SnakeMove) (bool, error) {
	snakesRemaining := []*Snake{}
	for i := 0; i < len(b.Snakes); i++ {
		if b.Snakes[i].EliminatedCause == NotEliminated {
			snakesRemaining = append(snakesRemaining, &b.Snakes[i])
		}
	}

	for i := 0; i < len(snakesRemaining); i++ {
		if !areSnakesOnSameSquad(settings.SquadSettings.squadMap, snakesRemaining[i], snakesRemaining[0]) {
			// There are multiple squads remaining
			return false, nil
		}
	}
	// no snakes or single squad remaining
	return true, nil
}

func (r SquadRuleset) Settings() Settings {
	s := r.StandardRuleset.Settings()
	s.SquadSettings = SquadSettings{
		squadMap:            r.SquadMap,
		AllowBodyCollisions: r.AllowBodyCollisions,
		SharedElimination:   r.SharedElimination,
		SharedHealth:        r.SharedHealth,
		SharedLength:        r.SharedLength,
	}
	return s
}

// Adaptor for integrating stages into SquadRuleset
func (r *SquadRuleset) callStageFunc(stage StageFunc, boardState *BoardState, moves []SnakeMove) (bool, error) {
	return stage(boardState, r.Settings(), moves)
}
