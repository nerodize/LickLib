package models

import (
	"database/sql/driver"
	"fmt"
)

type Difficulty string

const (
	DifficultyEasy    Difficulty = "EASY"
	DifficultyMedium  Difficulty = "MEDIUM"
	DifficultyHard    Difficulty = "HARD"
	DifficultyGoggins Difficulty = "GOGGINS"
)

func (d *Difficulty) Scan(value any) error {
	if value == nil {
		*d = ""
		return nil
	}
	s, ok := value.(string)
	if !ok {
		return fmt.Errorf("difficulty scan: not a String")
	}
	*d = Difficulty(s)
	return nil
}

func (d Difficulty) Value() (driver.Value, error) {
	if d == "" {
		return nil, nil
	}
	return string(d), nil
}

func PtrDifficulty(v Difficulty) *Difficulty { return &v }
