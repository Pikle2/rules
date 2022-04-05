package rules

type SoloRuleset struct {
	StandardRuleset
}

func (r *SoloRuleset) Name() string { return GameTypeSolo }

func (r SoloRuleset) Pipeline() (*Pipeline, error) {
	return NewPipeline(
		"movement.standard",
		"reducehealth.standard",
		"hazarddamage.standard",
		"eatfood.standard",
		"placefood.standard",
		"eliminatesnake.standard",
		"gameover.solo",
	)
}

func (r *SoloRuleset) IsGameOver(b *BoardState) (bool, error) {
	return r.callStageFunc(GameOverSolo, b, []SnakeMove{})
}

func GameOverSolo(b *BoardState, settings Settings, moves []SnakeMove) (bool, error) {
	for i := 0; i < len(b.Snakes); i++ {
		if b.Snakes[i].EliminatedCause == NotEliminated {
			return false, nil
		}
	}
	return true, nil
}
