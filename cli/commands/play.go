package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/BattlesnakeOfficial/rules"
	"github.com/BattlesnakeOfficial/rules/board"
	"github.com/BattlesnakeOfficial/rules/client"
	"github.com/BattlesnakeOfficial/rules/maps"
	"github.com/google/uuid"
	"github.com/pkg/browser"
	"github.com/spf13/cobra"
)

// Used to store state for each SnakeState while running a local game
type SnakeState struct {
	URL        string
	Name       string
	ID         string
	LastMove   string
	Character  rune
	Color      string
	Head       string
	Tail       string
	Author     string
	Version    string
	Error      error
	StatusCode int
}

type GameState struct {
	// Options
	Width               int
	Height              int
	Names               []string
	URLs                []string
	Timeout             int
	TurnDuration        int
	Sequential          bool
	GameType            string
	MapName             string
	ViewMap             bool
	UseColor            bool
	Seed                int64
	TurnDelay           int
	DebugRequests       bool
	Output              string
	ViewInBrowser       bool
	BoardURL            string
	FoodSpawnChance     int
	MinimumFood         int
	HazardDamagePerTurn int
	ShrinkEveryNTurns   int

	// Internal game state
	settings    map[string]string
	snakeStates map[string]SnakeState
	gameID      string
	httpClient  http.Client
	ruleset     rules.Ruleset
	gameMap     maps.GameMap
}

func NewPlayCommand() *cobra.Command {
	gameState := &GameState{}

	var playCmd = &cobra.Command{
		Use:   "play",
		Short: "Play a game of Battlesnake locally.",
		Long:  "Play a game of Battlesnake locally.",
		Run: func(cmd *cobra.Command, args []string) {
			gameState.Run()
		},
	}

	playCmd.Flags().IntVarP(&gameState.Width, "width", "W", 11, "Width of Board")
	playCmd.Flags().IntVarP(&gameState.Height, "height", "H", 11, "Height of Board")
	playCmd.Flags().StringArrayVarP(&gameState.Names, "name", "n", nil, "Name of Snake")
	playCmd.Flags().StringArrayVarP(&gameState.URLs, "url", "u", nil, "URL of Snake")
	playCmd.Flags().IntVarP(&gameState.Timeout, "timeout", "t", 500, "Request Timeout")
	playCmd.Flags().BoolVarP(&gameState.Sequential, "sequential", "s", false, "Use Sequential Processing")
	playCmd.Flags().StringVarP(&gameState.GameType, "gametype", "g", "standard", "Type of Game Rules")
	playCmd.Flags().StringVarP(&gameState.MapName, "map", "m", "standard", "Game map to use to populate the board")
	playCmd.Flags().BoolVarP(&gameState.ViewMap, "viewmap", "v", false, "View the Map Each Turn")
	playCmd.Flags().BoolVarP(&gameState.UseColor, "color", "c", false, "Use color to draw the map")
	playCmd.Flags().Int64VarP(&gameState.Seed, "seed", "r", time.Now().UTC().UnixNano(), "Random Seed")
	playCmd.Flags().IntVarP(&gameState.TurnDelay, "delay", "d", 0, "Turn Delay in Milliseconds")
	playCmd.Flags().IntVarP(&gameState.TurnDuration, "duration", "D", 0, "Minimum Turn Duration in Milliseconds")
	playCmd.Flags().BoolVar(&gameState.DebugRequests, "debug-requests", false, "Log body of all requests sent")
	playCmd.Flags().StringVarP(&gameState.Output, "output", "o", "", "File path to output game state to. Existing files will be overwritten")
	playCmd.Flags().BoolVar(&gameState.ViewInBrowser, "browser", false, "View the game in the browser using the Battlesnake game board")
	playCmd.Flags().StringVar(&gameState.BoardURL, "board-url", "https://board.battlesnake.com", "Base URL for the game board when using --browser")

	playCmd.Flags().IntVar(&gameState.FoodSpawnChance, "foodSpawnChance", 15, "Percentage chance of spawning a new food every round")
	playCmd.Flags().IntVar(&gameState.MinimumFood, "minimumFood", 1, "Minimum food to keep on the board every turn")
	playCmd.Flags().IntVar(&gameState.HazardDamagePerTurn, "hazardDamagePerTurn", 14, "Health damage a snake will take when ending its turn in a hazard")
	playCmd.Flags().IntVar(&gameState.ShrinkEveryNTurns, "shrinkEveryNTurns", 25, "In Royale mode, the number of turns between generating new hazards")

	playCmd.Flags().SortFlags = false

	return playCmd
}

// Setup a GameState once all the fields have been parsed from the command-line.
func (gameState *GameState) initialize() {
	// Generate game ID
	gameState.gameID = uuid.New().String()

	// Set up HTTP client with request timeout
	if gameState.Timeout == 0 {
		gameState.Timeout = 500
	}
	gameState.httpClient = http.Client{
		Timeout: time.Duration(gameState.Timeout) * time.Millisecond,
	}

	// Load game map
	gameMap, err := maps.GetMap(gameState.MapName)
	if err != nil {
		log.Fatalf("Failed to load game map %#v: %v", gameState.MapName, err)
	}
	gameState.gameMap = gameMap

	// Create settings object
	gameState.settings = map[string]string{
		rules.ParamGameType:            gameState.GameType,
		rules.ParamFoodSpawnChance:     fmt.Sprint(gameState.FoodSpawnChance),
		rules.ParamMinimumFood:         fmt.Sprint(gameState.MinimumFood),
		rules.ParamHazardDamagePerTurn: fmt.Sprint(gameState.HazardDamagePerTurn),
		rules.ParamShrinkEveryNTurns:   fmt.Sprint(gameState.ShrinkEveryNTurns),
	}

	// Build ruleset from settings
	ruleset := rules.NewRulesetBuilder().
		WithSeed(gameState.Seed).
		WithParams(gameState.settings).
		WithSolo(len(gameState.URLs) < 2).
		Ruleset()
	gameState.ruleset = ruleset

	// Initialize snake states as empty until we can ping the snake URLs
	gameState.snakeStates = map[string]SnakeState{}
}

// Setup and run a full game.
func (gameState *GameState) Run() {
	gameState.initialize()

	// Setup local state for snakes
	gameState.snakeStates = gameState.buildSnakesFromOptions()

	rand.Seed(gameState.Seed)

	boardState := gameState.initializeBoardFromArgs()
	exportGame := gameState.Output != ""

	gameExporter := GameExporter{
		game:          gameState.createClientGame(),
		snakeRequests: make([]client.SnakeRequest, 0),
		winner:        SnakeState{},
		isDraw:        false,
	}

	if gameState.ViewMap {
		gameState.printMap(boardState)
	}

	boardGame := board.Game{
		ID:     gameState.gameID,
		Status: "running",
		Width:  gameState.Width,
		Height: gameState.Height,
		Ruleset: map[string]string{
			rules.ParamGameType: gameState.GameType,
		},
		RulesetName: gameState.GameType,
		RulesStages: []string{},
		Map:         gameState.MapName,
	}
	boardServer := board.NewBoardServer(boardGame)

	if gameState.ViewInBrowser {
		serverURL, err := boardServer.Listen()
		if err != nil {
			log.Fatalf("Error starting HTTP server: %v", err)
		}
		log.Printf("Board server listening on %s", serverURL)

		boardURL := fmt.Sprintf(gameState.BoardURL+"?engine=%s&game=%s&autoplay=true", serverURL, gameState.gameID)

		log.Printf("Opening board URL: %s", boardURL)
		if err := browser.OpenURL(boardURL); err != nil {
			log.Printf("Failed to open browser: %v", err)
		}
	}

	if gameState.ViewInBrowser {
		// send turn zero to websocket server
		boardServer.SendEvent(gameState.createGameEvent(board.EVENT_TYPE_FRAME, boardState))
	}

	var endTime time.Time
	for v := false; !v; v, _ = gameState.ruleset.IsGameOver(boardState) {
		if gameState.TurnDuration > 0 {
			endTime = time.Now().Add(time.Duration(gameState.TurnDuration) * time.Millisecond)
		}

		// Export game first, if enabled, so that we save the board on turn zero
		if exportGame {
			// The output file was designed in a way so that (nearly) every entry is equivalent to a valid API request.
			// This is meant to help unlock further development of tools such as replaying a saved game by simply copying each line and sending it as a POST request.
			// There was a design choice to be made here: the difference between SnakeRequest and BoardState is the `you` key.
			// We could choose to either store the SnakeRequest of each snake OR to omit the `you` key OR fill the `you` key with one of the snakes
			// In all cases the API request is technically non-compliant with how the actual API request should be.
			// The third option (filling the `you` key with an arbitrary snake) is the closest to the actual API request that would need the least manipulation to
			// be adjusted to look like an API call for a specific snake in the game.
			for _, snakeState := range gameState.snakeStates {
				snakeRequest := gameState.getRequestBodyForSnake(boardState, snakeState)
				gameExporter.AddSnakeRequest(snakeRequest)
				break
			}
		}

		boardState = gameState.createNextBoardState(boardState)

		if gameState.ViewMap {
			gameState.printMap(boardState)
		} else {
			log.Printf("[%v]: State: %v\n", boardState.Turn, boardState)
		}

		if gameState.TurnDelay > 0 {
			time.Sleep(time.Duration(gameState.TurnDelay) * time.Millisecond)
		}

		if gameState.TurnDuration > 0 {
			time.Sleep(time.Until(endTime))
		}

		if gameState.ViewInBrowser {
			boardServer.SendEvent(gameState.createGameEvent(board.EVENT_TYPE_FRAME, boardState))
		}
	}

	isDraw := true
	if gameState.GameType == "solo" {
		log.Printf("[DONE]: Game completed after %v turns.", boardState.Turn)
		if exportGame {
			// These checks for exportGame are present to avoid vacuuming up RAM when an export is not requred.
			for _, snakeState := range gameState.snakeStates {
				gameExporter.winner = snakeState
				break
			}
		}
	} else {
		var winner SnakeState
		for _, snake := range boardState.Snakes {
			snakeState := gameState.snakeStates[snake.ID]
			if snake.EliminatedCause == rules.NotEliminated {
				isDraw = false
				winner = snakeState
			}
			gameState.sendEndRequest(boardState, snakeState)
		}

		if isDraw {
			log.Printf("[DONE]: Game completed after %v turns. It was a draw.", boardState.Turn)
		} else {
			log.Printf("[DONE]: Game completed after %v turns. %v is the winner.", boardState.Turn, winner.Name)
		}
		if exportGame {
			gameExporter.winner = winner
			gameExporter.isDraw = isDraw
		}
	}

	if gameState.ViewInBrowser {
		boardServer.SendEvent(board.GameEvent{
			EventType: board.EVENT_TYPE_GAME_END,
			Data:      boardGame,
		})
	}

	if exportGame {
		err := gameExporter.FlushToFile(gameState.Output, "JSONL")
		if err != nil {
			log.Printf("[WARN]: Unable to export game. Reason: %v\n", err.Error())
			os.Exit(1)
		}
	}

	if gameState.ViewInBrowser {
		boardServer.Shutdown()
	}
}

func (gameState *GameState) initializeBoardFromArgs() *rules.BoardState {
	snakeIds := []string{}
	for _, snakeState := range gameState.snakeStates {
		snakeIds = append(snakeIds, snakeState.ID)
	}
	boardState, err := maps.SetupBoard(gameState.gameMap.ID(), gameState.ruleset.Settings(), gameState.Width, gameState.Height, snakeIds)
	if err != nil {
		log.Fatalf("Error Initializing Board State: %v", err)
	}
	boardState, err = gameState.ruleset.ModifyInitialBoardState(boardState)
	if err != nil {
		log.Fatalf("Error Initializing Board State: %v", err)
	}

	for _, snakeState := range gameState.snakeStates {
		snakeRequest := gameState.getRequestBodyForSnake(boardState, snakeState)
		requestBody := serialiseSnakeRequest(snakeRequest)
		u, _ := url.ParseRequestURI(snakeState.URL)
		u.Path = path.Join(u.Path, "start")
		if gameState.DebugRequests {
			log.Printf("POST %s: %v", u, string(requestBody))
		}
		_, err = gameState.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(requestBody))
		if err != nil {
			log.Printf("[WARN]: Request to %v failed", u.String())
		}
	}
	return boardState
}

func (gameState *GameState) createNextBoardState(boardState *rules.BoardState) *rules.BoardState {
	var moves []rules.SnakeMove
	if gameState.Sequential {
		for _, snakeState := range gameState.snakeStates {
			for _, snake := range boardState.Snakes {
				if snakeState.ID == snake.ID && snake.EliminatedCause == rules.NotEliminated {
					moves = append(moves, gameState.getMoveForSnake(boardState, snakeState))
				}
			}
		}
	} else {
		var wg sync.WaitGroup
		c := make(chan rules.SnakeMove, len(gameState.snakeStates))

		for _, snakeState := range gameState.snakeStates {
			for _, snake := range boardState.Snakes {
				if snakeState.ID == snake.ID && snake.EliminatedCause == rules.NotEliminated {
					wg.Add(1)
					go func(snakeState SnakeState) {
						defer wg.Done()
						c <- gameState.getMoveForSnake(boardState, snakeState)
					}(snakeState)
				}
			}
		}

		wg.Wait()
		close(c)

		for move := range c {
			moves = append(moves, move)
		}
	}
	for _, move := range moves {
		snakeState := gameState.snakeStates[move.ID]
		snakeState.LastMove = move.Move
		gameState.snakeStates[move.ID] = snakeState
	}
	boardState, err := gameState.ruleset.CreateNextBoardState(boardState, moves)
	if err != nil {
		log.Fatalf("Error producing next board state: %v", err)
	}

	boardState, err = maps.UpdateBoard(gameState.gameMap.ID(), boardState, gameState.ruleset.Settings())
	if err != nil {
		log.Fatalf("Error updating board with game map: %v", err)
	}

	boardState.Turn += 1

	return boardState
}

func (gameState *GameState) getMoveForSnake(boardState *rules.BoardState, snakeState SnakeState) rules.SnakeMove {
	snakeRequest := gameState.getRequestBodyForSnake(boardState, snakeState)
	requestBody := serialiseSnakeRequest(snakeRequest)
	u, _ := url.ParseRequestURI(snakeState.URL)
	u.Path = path.Join(u.Path, "move")
	if gameState.DebugRequests {
		log.Printf("POST %s: %v", u, string(requestBody))
	}
	res, err := gameState.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(requestBody))
	move := snakeState.LastMove
	if err != nil {
		log.Printf("[WARN]: Request to %v failed\n", u.String())
		log.Printf("Body --> %v\n", string(requestBody))
	} else if res.Body != nil {
		defer res.Body.Close()
		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			log.Fatal(readErr)
		} else {
			playerResponse := client.MoveResponse{}
			jsonErr := json.Unmarshal(body, &playerResponse)
			if jsonErr != nil {
				log.Fatal(jsonErr)
			} else {
				move = playerResponse.Move
			}
		}
	}
	return rules.SnakeMove{ID: snakeState.ID, Move: move}
}

func (gameState *GameState) sendEndRequest(boardState *rules.BoardState, snakeState SnakeState) {
	snakeRequest := gameState.getRequestBodyForSnake(boardState, snakeState)
	requestBody := serialiseSnakeRequest(snakeRequest)
	u, _ := url.ParseRequestURI(snakeState.URL)
	u.Path = path.Join(u.Path, "end")
	if gameState.DebugRequests {
		log.Printf("POST %s: %v", u, string(requestBody))
	}
	_, err := gameState.httpClient.Post(u.String(), "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		log.Printf("[WARN]: Request to %v failed", u.String())
	}
}

func (gameState *GameState) getRequestBodyForSnake(boardState *rules.BoardState, snakeState SnakeState) client.SnakeRequest {
	var youSnake rules.Snake
	for _, snk := range boardState.Snakes {
		if snakeState.ID == snk.ID {
			youSnake = snk
			break
		}
	}
	request := client.SnakeRequest{
		Game:  gameState.createClientGame(),
		Turn:  boardState.Turn,
		Board: convertStateToBoard(boardState, gameState.snakeStates),
		You:   convertRulesSnake(youSnake, snakeState),
	}
	return request
}

func (gameState *GameState) createClientGame() client.Game {
	return client.Game{
		ID:      gameState.gameID,
		Timeout: gameState.Timeout,
		Ruleset: client.Ruleset{
			Name:     gameState.ruleset.Name(),
			Version:  "cli", // TODO: Use GitHub Release Version
			Settings: gameState.ruleset.Settings(),
		},
		Map: gameState.gameMap.ID(),
	}
}

func (gameState *GameState) buildSnakesFromOptions() map[string]SnakeState {
	bodyChars := []rune{'■', '⌀', '●', '☻', '◘', '☺', '□', '⍟'}
	var numSnakes int
	snakes := map[string]SnakeState{}
	numNames := len(gameState.Names)
	numURLs := len(gameState.URLs)
	if numNames > numURLs {
		numSnakes = numNames
	} else {
		numSnakes = numURLs
	}
	if numNames != numURLs {
		log.Println("[WARN]: Number of Names and URLs do not match: defaults will be applied to missing values")
	}
	for i := int(0); i < numSnakes; i++ {
		var snakeName string
		var snakeURL string

		id := uuid.New().String()

		if i < numNames {
			snakeName = gameState.Names[i]
		} else {
			log.Printf("[WARN]: Name for URL %v is missing: a default name will be applied\n", gameState.URLs[i])
			snakeName = id
		}

		if i < numURLs {
			u, err := url.ParseRequestURI(gameState.URLs[i])
			if err != nil {
				log.Printf("[WARN]: URL %v is not valid: a default will be applied\n", gameState.URLs[i])
				snakeURL = "https://example.com"
			} else {
				snakeURL = u.String()
			}
		} else {
			log.Printf("[WARN]: URL for Name %v is missing: a default URL will be applied\n", gameState.Names[i])
			snakeURL = "https://example.com"
		}

		snakeState := SnakeState{
			Name: snakeName, URL: snakeURL, ID: id, LastMove: "up", Character: bodyChars[i%8],
		}
		var snakeErr error
		res, err := gameState.httpClient.Get(snakeURL)
		if err != nil {
			log.Printf("[WARN]: Request to %v failed: %v", snakeURL, err)
			snakeErr = err
		} else {
			snakeState.StatusCode = res.StatusCode

			if res.Body != nil {
				defer res.Body.Close()
				body, readErr := ioutil.ReadAll(res.Body)
				if readErr != nil {
					log.Fatal(readErr)
				}

				pingResponse := client.SnakeMetadataResponse{}
				jsonErr := json.Unmarshal(body, &pingResponse)
				if jsonErr != nil {
					snakeErr = jsonErr
					log.Printf("Error reading response from %v: %v", snakeURL, jsonErr)
				} else {
					snakeState.Head = pingResponse.Head
					snakeState.Tail = pingResponse.Tail
					snakeState.Color = pingResponse.Color
					snakeState.Author = pingResponse.Author
					snakeState.Version = pingResponse.Version
				}
			}
		}
		if snakeErr != nil {
			snakeState.Error = snakeErr
		}

		snakes[snakeState.ID] = snakeState
	}
	return snakes
}

func (gameState *GameState) printMap(boardState *rules.BoardState) {
	var o bytes.Buffer
	o.WriteString(fmt.Sprintf("Ruleset: %s, Seed: %d, Turn: %v\n", gameState.GameType, gameState.Seed, boardState.Turn))
	board := make([][]string, boardState.Width)
	for i := range board {
		board[i] = make([]string, boardState.Height)
	}
	for y := int(0); y < boardState.Height; y++ {
		for x := int(0); x < boardState.Width; x++ {
			if gameState.UseColor {
				board[x][y] = TERM_FG_LIGHTGRAY + "□"
			} else {
				board[x][y] = "◦"
			}
		}
	}
	for _, oob := range boardState.Hazards {
		if gameState.UseColor {
			board[oob.X][oob.Y] = TERM_BG_GRAY + " " + TERM_BG_WHITE
		} else {
			board[oob.X][oob.Y] = "░"
		}
	}
	if gameState.UseColor {
		o.WriteString(fmt.Sprintf("Hazards "+TERM_BG_GRAY+" "+TERM_RESET+": %v\n", boardState.Hazards))
	} else {
		o.WriteString(fmt.Sprintf("Hazards ░: %v\n", boardState.Hazards))
	}
	for _, f := range boardState.Food {
		if gameState.UseColor {
			board[f.X][f.Y] = TERM_FG_FOOD + "●"
		} else {
			board[f.X][f.Y] = "⚕"
		}
	}
	if gameState.UseColor {
		o.WriteString(fmt.Sprintf("Food "+TERM_FG_FOOD+TERM_BG_WHITE+"●"+TERM_RESET+": %v\n", boardState.Food))
	} else {
		o.WriteString(fmt.Sprintf("Food ⚕: %v\n", boardState.Food))
	}
	for _, s := range boardState.Snakes {
		red, green, blue := parseSnakeColor(gameState.snakeStates[s.ID].Color)
		for _, b := range s.Body {
			if b.X >= 0 && b.X < boardState.Width && b.Y >= 0 && b.Y < boardState.Height {
				if gameState.UseColor {
					board[b.X][b.Y] = fmt.Sprintf(TERM_FG_RGB+"■", red, green, blue)
				} else {
					board[b.X][b.Y] = string(gameState.snakeStates[s.ID].Character)
				}
			}
		}
		if gameState.UseColor {
			o.WriteString(fmt.Sprintf("%v "+TERM_FG_RGB+TERM_BG_WHITE+"■■■"+TERM_RESET+": %v\n", gameState.snakeStates[s.ID].Name, red, green, blue, s))
		} else {
			o.WriteString(fmt.Sprintf("%v %c: %v\n", gameState.snakeStates[s.ID].Name, gameState.snakeStates[s.ID].Character, s))
		}
	}
	for y := boardState.Height - 1; y >= 0; y-- {
		if gameState.UseColor {
			o.WriteString(TERM_BG_WHITE)
		}
		for x := int(0); x < boardState.Width; x++ {
			o.WriteString(board[x][y])
		}
		if gameState.UseColor {
			o.WriteString(TERM_RESET)
		}
		o.WriteString("\n")
	}
	log.Print(o.String())
}

func (gameState *GameState) createGameEvent(eventType board.GameEventType, boardState *rules.BoardState) board.GameEvent {
	snakes := []board.Snake{}

	for _, snake := range boardState.Snakes {
		snakeState := gameState.snakeStates[snake.ID]

		convertedSnake := board.Snake{
			ID:            snake.ID,
			Name:          snakeState.Name,
			Body:          snake.Body,
			Health:        snake.Health,
			Color:         snakeState.Color,
			HeadType:      snakeState.Head,
			TailType:      snakeState.Tail,
			Author:        snakeState.Author,
			StatusCode:    snakeState.StatusCode,
			IsBot:         false,
			IsEnvironment: false,

			// Not supporting local latency for now - there are better ways to test performance locally
			Latency: "1",
		}
		if snakeState.Error != nil {
			// Instead of trying to keep in sync with the production engine's
			// error detection and messages, just show a generic error and rely
			// on the CLI logs to show what really happened.
			convertedSnake.Error = "0:Error communicating with server"
		} else if snakeState.StatusCode != http.StatusOK {
			convertedSnake.Error = fmt.Sprintf("7:Bad HTTP status code %d", snakeState.StatusCode)
		}
		if snake.EliminatedCause != rules.NotEliminated {
			convertedSnake.Death = &board.Death{
				Cause:        snake.EliminatedCause,
				Turn:         snake.EliminatedOnTurn,
				EliminatedBy: snake.EliminatedBy,
			}
		}
		snakes = append(snakes, convertedSnake)
	}

	gameFrame := board.GameFrame{
		Turn:    boardState.Turn,
		Snakes:  snakes,
		Food:    boardState.Food,
		Hazards: boardState.Hazards,
	}

	return board.GameEvent{
		EventType: eventType,
		Data:      gameFrame,
	}
}

func serialiseSnakeRequest(snakeRequest client.SnakeRequest) []byte {
	requestJSON, err := json.Marshal(snakeRequest)
	if err != nil {
		log.Fatalf("Error marshalling JSON from State: %v", err)
	}
	return requestJSON
}

func convertRulesSnake(snake rules.Snake, snakeState SnakeState) client.Snake {
	return client.Snake{
		ID:      snake.ID,
		Name:    snakeState.Name,
		Health:  snake.Health,
		Body:    client.CoordFromPointArray(snake.Body),
		Latency: "0",
		Head:    client.CoordFromPoint(snake.Body[0]),
		Length:  int(len(snake.Body)),
		Shout:   "",
		Customizations: client.Customizations{
			Head:  snakeState.Head,
			Tail:  snakeState.Tail,
			Color: snakeState.Color,
		},
	}
}

func convertRulesSnakes(snakes []rules.Snake, snakeStates map[string]SnakeState) []client.Snake {
	a := make([]client.Snake, 0)
	for _, snake := range snakes {
		if snake.EliminatedCause == rules.NotEliminated {
			a = append(a, convertRulesSnake(snake, snakeStates[snake.ID]))
		}
	}
	return a
}

func convertStateToBoard(boardState *rules.BoardState, snakeStates map[string]SnakeState) client.Board {
	return client.Board{
		Height:  boardState.Height,
		Width:   boardState.Width,
		Food:    client.CoordFromPointArray(boardState.Food),
		Hazards: client.CoordFromPointArray(boardState.Hazards),
		Snakes:  convertRulesSnakes(boardState.Snakes, snakeStates),
	}
}

// Parses a color string like "#ef03d3" to rgb values from 0 to 255 or returns
// the default gray if any errors occure
func parseSnakeColor(color string) (int64, int64, int64) {
	if len(color) == 7 {
		red, err_r := strconv.ParseInt(color[1:3], 16, 64)
		green, err_g := strconv.ParseInt(color[3:5], 16, 64)
		blue, err_b := strconv.ParseInt(color[5:], 16, 64)
		if err_r == nil && err_g == nil && err_b == nil {
			return red, green, blue
		}
	}
	// Default gray color from Battlesnake board
	return 136, 136, 136
}
