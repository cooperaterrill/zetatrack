package main

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	"math/rand/v2"

	"github.com/Knetic/govaluate"
	"golang.org/x/term"
)

type Mode int

const (
	GameMode Mode = iota
	StatsMode
	ConfigMode
)

type Problem struct {
	FirstNum  int
	Operation string
	SecondNum int
}

func (problem Problem) String() string {
	return fmt.Sprintf("%d %s %d", problem.FirstNum, problem.Operation, problem.SecondNum)
}

func ParseProblem(problemString string) Problem {
	parts := strings.Split(problemString, " ")
	firstNum, err := strconv.Atoi(parts[0])
	if err != nil {
		panic(err)
	}
	secondNum, err := strconv.Atoi(parts[2])
	if err != nil {
		panic(err)
	}
	return Problem{FirstNum: firstNum, Operation: parts[1], SecondNum: secondNum}
}

type Log struct {
	Problems   []Problem
	Times      []int64
	LogTime    time.Time
	GameLength int
}

func NewLog(problems []Problem, times []int64, gameLength int) Log {
	return Log{Problems: problems, Times: times, LogTime: time.Now(), GameLength: gameLength}
}

func ParseLog(line string) Log {
	parts := strings.Split(line, " ")

	logTimeUnix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		panic(err)
	}
	logTime := time.Unix(logTimeUnix, 0)

	gameLength, err := strconv.Atoi(parts[1])
	if err != nil {
		panic(err)
	}

	var problems []Problem
	var times []int64
	for i := 2; i < len(parts)-3; i += 4 {
		problem := ParseProblem(parts[i] + " " + parts[i+1] + " " + parts[i+2])
		problems = append(problems, problem)
		time, err := strconv.ParseInt(parts[i+3], 10, 64)
		if err != nil {
			panic(err)
		}
		times = append(times, time)
	}

	return Log{Problems: problems, Times: times, LogTime: logTime, GameLength: gameLength}
}

func (log Log) String() string {
	var sb strings.Builder

	sb.WriteString(strconv.FormatInt(time.Now().Unix(), 10) + " ")
	sb.WriteString(strconv.Itoa(log.GameLength) + " ")
	for i := 0; i < len(log.Problems)-1; i++ {
		sb.WriteString(log.Problems[i].String() + " " + strconv.FormatInt(log.Times[i], 10) + " ")
	}
	sb.WriteString(log.Problems[len(log.Problems)-1].String() + " " + "-1" + "\r\n")

	return sb.String()
}

type AdditionConfig struct {
	MinLeft  int
	MaxLeft  int
	MinRight int
	MaxRight int
}

type SubtractionConfig struct {
	MinLeft                    int
	MaxLeft                    int
	MinRight                   int
	MaxRight                   int
	ForceNonnegativeDifference bool
}

type MultiplicationConfig struct {
	MinLeft  int
	MaxLeft  int
	MinRight int
	MaxRight int
}

type DivisionConfig struct {
	MinLeft            int
	MaxLeft            int
	MinRight           int
	MaxRight           int
	ForceCleanDivision bool
}

type Config struct {
	AdditionConfig            AdditionConfig
	SubtractionConfig         SubtractionConfig
	MultiplicationConfig      MultiplicationConfig
	DivisionConfig            DivisionConfig
	OverrideSubtractionConfig bool
	OverrideDivisionConfig    bool
	Duration                  int
	LegalOperations           []string
}

var mode Mode
var currentProblem Problem

const ClearSignal = "clear"
const QuitSignal = "quit"

func handleClargs(config *Config) {
	clargs := os.Args
	if len(clargs) > 1 && clargs[1] == "-s" {
		mode = StatsMode
	} else {
		mode = GameMode
	}
	for i := 1; i < len(os.Args)-1; i++ {
		if clargs[i] == "-t" {
			num, err := strconv.Atoi(clargs[i+1])
			if err != nil {
				fmt.Printf("Enter an integer number of seconds\r\n")
			}
			config.Duration = num
		}
	}
}

func readInput(buf []byte, channel chan string) {
	var answerBuf = make([]byte, 10)
	answerBufFront := 0
	for {
		select {
		case x, ok := <-channel:
			if ok {
				if x == ClearSignal {
					answerBufFront = 0
				}
			} else {
				fmt.Printf("Something went wrong!")
			}
		default:
		}
		n, err := os.Stdin.Read(buf)
		if err == nil && n > 0 {
			//fmt.Printf("Read: %v\r\n", buf[:n])
			if (buf[0] < 48 || buf[0] > 57) && buf[0] != 0x7f && buf[0] != 'q' {
				continue
			}
			if buf[0] == 127 || buf[0] == 8 {
				if answerBufFront <= 0 {
					continue
				}
				answerBufFront--

				//fmt.Printf("\b \b")
			} else if buf[0] == 'q' {
				channel <- QuitSignal
				return
			} else {
				if answerBufFront >= len(answerBuf) {
					continue
				}
				answerBuf[answerBufFront] = buf[0]
				answerBufFront++
			}
			fmt.Printf("\r\033[K")
			fmt.Printf("%s: %s", currentProblem, string(answerBuf[0:answerBufFront]))

			//fmt.Printf("%c", buf[0])
			//fmt.Printf("Current answer: %s\n", answerBuf[0:answerBufFront])
			channel <- string(answerBuf[0:answerBufFront])
		}
		time.Sleep(time.Millisecond * 5)
	}
}

func randRange(min int, max int) int {
	return rand.IntN(max-min+1) + min
}

func genAdditionProblem(config AdditionConfig) Problem {
	var problem Problem
	problem.Operation = "+"
	problem.FirstNum = randRange(config.MinLeft, config.MaxLeft)
	problem.SecondNum = randRange(config.MinRight, config.MaxRight)
	return problem
}

func genMultiplicationProblem(config MultiplicationConfig) Problem {
	var problem Problem
	problem.Operation = "*"
	problem.FirstNum = randRange(config.MinLeft, config.MaxLeft)
	problem.SecondNum = randRange(config.MinRight, config.MaxRight)
	return problem
}

func genSubtractionProblem(config SubtractionConfig) Problem {
	var problem Problem
	problem.Operation = "-"
	for true {
		problem.FirstNum = randRange(config.MinLeft, config.MaxLeft)
		problem.SecondNum = randRange(config.MinRight, config.MaxRight)
		if !config.ForceNonnegativeDifference || problem.FirstNum-problem.SecondNum >= 0 {
			return problem
		}
	}
	//this will never hit
	return problem
}

func genDivisionProblem(config DivisionConfig) Problem {
	var problem Problem
	problem.Operation = "/"
	for true {
		problem.FirstNum = randRange(config.MinLeft, config.MaxLeft)
		problem.SecondNum = randRange(config.MinRight, config.MaxRight)
		if !config.ForceCleanDivision || problem.FirstNum%problem.SecondNum == 0 {
			return problem
		}
	}
	//this will never hit
	return problem
}

func genProblem(config Config) Problem {
	operation := config.LegalOperations[randRange(0, len(config.LegalOperations)-1)]

	if operation == "-" && config.OverrideSubtractionConfig {
		addProblem := genAdditionProblem(config.AdditionConfig)
		ans := getProblemAnswer(addProblem)
		return Problem{ans, "-", addProblem.FirstNum}
	} else if operation == "/" && config.OverrideDivisionConfig {
		multProblem := genMultiplicationProblem(config.MultiplicationConfig)
		ans := getProblemAnswer(multProblem)
		return Problem{ans, "/", multProblem.FirstNum}
	} else if operation == "+" {
		return genAdditionProblem(config.AdditionConfig)
	} else if operation == "*" {
		return genMultiplicationProblem(config.MultiplicationConfig)
	} else if operation == "-" {
		return genSubtractionProblem(config.SubtractionConfig)
	} else {
		return genDivisionProblem(config.DivisionConfig)
	}
}

func getProblemAnswer(problem Problem) int {
	expression, err := govaluate.NewEvaluableExpression(problem.String())
	if err != nil {
		fmt.Print("Error parsing expression\r\n")
		panic(err)
	}
	evaluated, err := expression.Evaluate(nil)
	if err != nil {
		fmt.Print("Error evaluating expression\r\n")
	}
	return int(evaluated.(float64))
}

func saveScores(problems []Problem, times []int64, filepath string, config Config) {
	var file *os.File
	var err error
	file, err = os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(NewLog(problems, times, config.Duration).String())
}

func gameLoop(config Config, inputChannel chan string, oldState *term.State) {
	fmt.Printf("duration will be %d\r\n", config.Duration)
	var problems []Problem
	var times []int64
	score := 0
	timer := time.NewTimer(time.Duration(config.Duration) * time.Second)
	firstProblem := true
	cleanup := func() {
		fmt.Printf("\r\nScore: %d\r\n", score)
		saveScores(problems, times, "scores.txt", config)

		term.Restore(int(os.Stdin.Fd()), oldState)
		return
	}
	defer cleanup()

	go func() {
		<-timer.C
		cleanup()
		os.Exit(0)
	}()

	for {
		problem := genProblem(config)
		currentProblem = problem
		problemAns := getProblemAnswer(problem)
		problems = append(problems, problem)
		if firstProblem {
			fmt.Printf("%s: ", problem)
			firstProblem = false
		} else {
			fmt.Printf("\r\n%s: ", problem)
		}
		startTime := time.Now()

		for {
			userAns := <-inputChannel
			if userAns == strconv.Itoa(problemAns) {
				times = append(times, time.Now().Sub(startTime).Milliseconds())
				//fmt.Printf("\r\nYou got the right answer\r\n")
				score++
				inputChannel <- ClearSignal
				break
			}
			if userAns == QuitSignal {
				return
			}
		}
	}
}

func printStats(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var logs []Log
	for scanner.Scan() {
		line := scanner.Text()
		if len(strings.Trim(line, "\r\n\t ")) == 0 {
			continue
		}
		logs = append(logs, ParseLog(line))
	}
	var times []int64
	for _, log := range logs {
		for _, time := range log.Times {
			if time == -1 {
				continue
			}
			times = append(times, time)
		}
	}
	median, iqr := MedianAndIqr(times)
	mean, stdev := MeanAndStdev(times)
	fmt.Printf("Median: %d \r\nIQR: %d\r\n", median, iqr)
	fmt.Printf("Mean: %d \r\nSTDev: %d\r\n", mean, stdev)
}

func main() {
	add := AdditionConfig{2, 100, 2, 100}
	sub := SubtractionConfig{100, 2, 100, 2, true}
	mult := MultiplicationConfig{2, 12, 2, 100}
	div := DivisionConfig{1200, 2, 100, 2, true}
	config := Config{add, sub, mult, div, true, true, 120, []string{"+", "-", "/", "*"}}

	handleClargs(&config)

	switch mode {
	case StatsMode:
		printStats("scores.txt")
		return
	case ConfigMode: //TODO: implement config mode
	case GameMode:
		fmt.Printf("Game mode!\r\n")
		oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
		if err != nil {
			panic(err)
		}

		fd := int(os.Stdin.Fd())
		var buf = make([]byte, 1)

		err = syscall.SetNonblock(fd, true)
		if err != nil {
			panic(err)
		}

		inputChannel := make(chan string)
		go readInput(buf, inputChannel)

		gameLoop(config, inputChannel, oldState)
	}

}
