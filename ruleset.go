package rules

type RulesetError string

func (err RulesetError) Error() string { return string(err) }

const (
	MoveUp    = "up"
	MoveDown  = "down"
	MoveRight = "right"
	MoveLeft  = "left"

	BoardSizeSmall  = 7
	BoardSizeMedium = 11
	BoardSizeLarge  = 19

	SnakeMaxHealth = 100
	SnakeStartSize = 3

	// bvanvugt - TODO: Just return formatted strings instead of codes?
	NotEliminated                   = ""
	EliminatedByCollision           = "snake-collision"
	EliminatedBySelfCollision       = "snake-self-collision"
	EliminatedByOutOfHealth         = "out-of-health"
	EliminatedByHeadToHeadCollision = "head-collision"
	EliminatedByOutOfBounds         = "wall-collision"

	// TODO - Error consts
	ErrorTooManySnakes   = RulesetError("too many snakes for fixed start positions")
	ErrorNoRoomForSnake  = RulesetError("not enough space to place snake")
	ErrorNoRoomForFood   = RulesetError("not enough space to place food")
	ErrorNoMoveFound     = RulesetError("move not provided for snake")
	ErrorZeroLengthSnake = RulesetError("snake is length zero")
)

type Point struct {
	X int32 `json:"x"`
	Y int32 `json:"y"`
}

type Snake struct {
	ID               string  `json:"id"`
	Body             []Point `json:"body"`
	Health           int32   `json:"health"`
	EliminatedCause  string  `json:"eliminated_cause"`
	EliminatedOnTurn int32   `json:"eliminated_on_turn"`
	EliminatedBy     string  `json:"eliminated_by"`
}

type SnakeMove struct {
	ID   string `json:"id"`
	Move string `json:"move"`
}

type Ruleset interface {
	Name() string
	ModifyInitialBoardState(initialState *BoardState) (*BoardState, error)
	CreateNextBoardState(prevState *BoardState, moves []SnakeMove) (*BoardState, error)
	IsGameOver(state *BoardState) (bool, error)
}
