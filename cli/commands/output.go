package commands

import (
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/BattlesnakeOfficial/rules/client"
)

type GameExporter struct {
	game          client.Game
	snakeRequests []client.SnakeRequest
	winner        SnakeState
	isDraw        bool
}

type result struct {
	winnerID   string `json:"winnerId"`
	winnerName string `json:"winnerName"`
	isDraw     bool   `json:"isDraw"`
}

func (ge *GameExporter) FlushToFile(filepath string, format string) error {
	var formattedOutput []string
	var formattingErr error

	// TODO: Support more formats
	if format == "JSONL" {
		formattedOutput, formattingErr = ge.ConvertToJSON()
	} else {
		log.Fatalf("Invalid output format passed: %s", format)
	}

	if formattingErr != nil {
		return formattingErr
	}

	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, line := range formattedOutput {
		f.WriteString(fmt.Sprintf("%s\n", line))
	}

	return nil
}

func (ge *GameExporter) ConvertToJSON() ([]string, error) {
	output := make([]string, 0)
	serialisedGame, err := json.Marshal(ge.game)
	if err != nil {
		return output, err
	}
	output = append(output, string(serialisedGame))
	for _, board := range ge.snakeRequests {
		serialisedBoard, err := json.Marshal(board)
		if err != nil {
			return output, err
		}
		output = append(output, string(serialisedBoard))
	}
	serialisedResult, err := json.Marshal(result{
		winnerID:   ge.winner.ID,
		winnerName: ge.winner.Name,
		isDraw:     ge.isDraw,
	})
	if err != nil {
		return output, err
	}
	output = append(output, string(serialisedResult))
	return output, nil
}

func (ge *GameExporter) AddSnakeRequest(snakeRequest client.SnakeRequest) {
	if ge.snakeRequests == nil {
		ge.snakeRequests = []client.SnakeRequest{snakeRequest}
	} else {
		ge.snakeRequests = append(ge.snakeRequests, snakeRequest)
	}
}
