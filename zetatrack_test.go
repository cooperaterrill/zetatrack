package main

import (
	"math"
	"os"
	"reflect"
	"testing"
)

func TestParseProblem(t *testing.T) {
	wantProblem := Problem{10, "*", 20}
	gotProblem := ParseProblem(wantProblem.String())
	if wantProblem != gotProblem {
		t.Errorf("Couldn't parse problem %s", wantProblem.String())
	}
}

func TestParseProblemLargeOperands(t *testing.T) {
	wantProblem := Problem{math.MaxInt/2 - 1, "*", math.MaxInt/2 - 2}
	gotProblem := ParseProblem(wantProblem.String())
	if wantProblem != gotProblem {
		t.Errorf("Coudldn't parse problem %s", wantProblem.String())
	}
}

func TestParseLog(t *testing.T) {
	var problems []Problem
	var times []int64
	gameLength := 60
	problems = append(problems, Problem{10, "*", 20})
	problems = append(problems, Problem{100, "*", 200})
	problems = append(problems, Problem{0, "*", 0})
	problems = append(problems, Problem{90, "-", 1000})
	times = append(times, 500)
	times = append(times, 200000)
	times = append(times, 20)
	//always have last problem unsolved
	times = append(times, -1)
	wantLog := NewLog(problems, times, gameLength)

	gotLog := ParseLog(wantLog.String())
	if wantLog.LogTime.Sub(gotLog.LogTime).Abs().Milliseconds() < 5 {
		t.Errorf("Failed parsing log time: %s and %s", wantLog.LogTime.String(), wantLog.LogTime.String())
	}
	if !reflect.DeepEqual(wantLog.Problems, gotLog.Problems) {
		t.Errorf("Failed parsing log problems: %v and %v", wantLog.Problems, gotLog.Problems)
	}
	if !reflect.DeepEqual(wantLog.Times, gotLog.Times) {
		t.Errorf("Failing parsing log solvetimes: %v and %v", wantLog.Times, gotLog.Times)
	}
	if wantLog.GameLength != gotLog.GameLength {
		t.Errorf("Failed parsing log gamelength: %d and %d", wantLog.GameLength, gotLog.GameLength)
	}

}

func TestLoadZetamacConfig(t *testing.T) {
	var config Config
	config.Load("test/configs/zetamac.txt")
	if !reflect.DeepEqual(config, GetZetamacConfig()) {
		t.Errorf("Failed loading zetamac config: wanted %s, got %s", GetZetamacConfig().String(), config.String())
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	add := AdditionConfig{5, 100, 2, 600}
	sub := SubtractionConfig{4, 90, 30, 60, false}
	mult := MultiplicationConfig{1, 8, 2, 50}
	div := DivisionConfig{6, 2000, 30, 6000, false}
	wantConfig := Config{"custom", add, sub, mult, div, false, true, 69, []string{"*", "-"}}

	os.Remove("test/configs/custom.txt")
	wantConfig.Save("test/configs/custom.txt")
	var gotConfig Config
	gotConfig.Load("test/configs/custom.txt")

	if !reflect.DeepEqual(wantConfig, gotConfig) {
		t.Errorf("Save/Load config failed: wanted %s, got %s", wantConfig.String(), gotConfig.String())
	}
}
