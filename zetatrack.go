package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"os"
	"slices"
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
	trimmedLine := strings.Trim(line, "\r\n\t ")
	parts := strings.Split(trimmedLine, " ")

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

func (config AdditionConfig) String() string {
	return fmt.Sprintf("%d-%d\t%d-%d", config.MinLeft, config.MaxLeft, config.MinRight, config.MaxRight)
}

type SubtractionConfig struct {
	MinLeft                    int
	MaxLeft                    int
	MinRight                   int
	MaxRight                   int
	ForceNonnegativeDifference bool
}

func (config SubtractionConfig) String() string {
	return fmt.Sprintf("%d-%d\t%d-%d\t%t", config.MinLeft, config.MaxLeft, config.MinRight, config.MaxRight, config.ForceNonnegativeDifference)
}

type MultiplicationConfig struct {
	MinLeft  int
	MaxLeft  int
	MinRight int
	MaxRight int
}

func (config MultiplicationConfig) String() string {
	return fmt.Sprintf("%d-%d\t%d-%d", config.MinLeft, config.MaxLeft, config.MinRight, config.MaxRight)
}

type DivisionConfig struct {
	MinLeft            int
	MaxLeft            int
	MinRight           int
	MaxRight           int
	ForceCleanDivision bool
}

func (config DivisionConfig) String() string {
	return fmt.Sprintf("%d-%d\t%d-%d\t%t", config.MinLeft, config.MaxLeft, config.MinRight, config.MaxRight, config.ForceCleanDivision)
}

type Config struct {
	Name                      string
	AdditionConfig            AdditionConfig
	SubtractionConfig         SubtractionConfig
	MultiplicationConfig      MultiplicationConfig
	DivisionConfig            DivisionConfig
	OverrideSubtractionConfig bool
	OverrideDivisionConfig    bool
	Duration                  int
	LegalOperations           []string
}

func (config *Config) Load(filepath string) {
	fmt.Printf("\r\nLoading config %s\r\n", filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	buffer := make([]byte, 100*1024)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		panic(err)
	}
	json.Unmarshal(buffer[:n], config)
}

func (config Config) Save(filepath string) {
	file, err := os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	res, err := json.Marshal(config)
	if err != nil {
		panic(err)
	}
	file.Write(res)
}

func (config Config) String() string {
	return fmt.Sprintf("%s\r\n%s\r\n%s \r\n%s \r\n%s \r\n%t %t %d %s\r\n", config.Name, config.AdditionConfig.String(), config.SubtractionConfig.String(), config.MultiplicationConfig.String(), config.DivisionConfig.String(), config.OverrideSubtractionConfig, config.OverrideDivisionConfig, config.Duration, strings.Join(config.LegalOperations, " "))
}

func GetZetamacConfig() Config {
	add := AdditionConfig{2, 100, 2, 100}
	sub := SubtractionConfig{2, 100, 2, 100, true}
	mult := MultiplicationConfig{2, 12, 2, 100}
	div := DivisionConfig{2, 1200, 2, 100, true}
	return Config{"default", add, sub, mult, div, true, true, 120, []string{"+", "-", "/", "*"}}
}

var mode Mode
var currentProblem Problem

const ClearSignal = "clear"
const QuitSignal = "quit"

func handleClargs(config *Config) {
	clargs := os.Args
	if len(clargs) <= 1 {
		if fileExists("configs/default.txt") {
			config.Load("configs/default.txt")
		} else {
			*config = GetZetamacConfig()
		}
		return
	}
	if clargs[1] == "-s" {
		mode = StatsMode
	} else if clargs[1] == "-c" {
		mode = ConfigMode
	} else {
		//we are in game mode, so load relevant config
		config.Load("configs/" + clargs[1] + ".txt")
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil || !os.IsNotExist(err)
}

func getCleanInput(reader *bufio.Reader) string {
	line, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	return strings.Trim(line, "\r\n\t ")
}

func bracketCurrentOption(cond bool) string {
	if cond {
		return " [y]/n: "
	} else {
		return " y/[n]"
	}
}

func setByInput(input string, option *bool) {
	input = strings.ToLower(input)
	if input == "y" || input == "yes" {
		*option = true
	}
	if input == "n" || input == "no" {
		*option = false
	}
}

func setupAdditionConfig(config *AdditionConfig, reader *bufio.Reader) {
	fmt.Printf("\r\nSmallest left number [%d]: ", config.MinLeft)
	line := getCleanInput(reader)
	num, err := strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinLeft = num
	}
	fmt.Printf("\r\nLargest left number [%d]: ", config.MaxLeft)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxLeft = num
	}
	fmt.Printf("\r\nSmallest right number [%d]: ", config.MinRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinRight = num
	}
	fmt.Printf("\r\nLargest right number [%d]: ", config.MaxRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxRight = num
	}
}

func setupSubtractionConfig(config *SubtractionConfig, reader *bufio.Reader) {
	fmt.Printf("\r\nSmallest left number [%d]: ", config.MinLeft)
	line := getCleanInput(reader)
	num, err := strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinLeft = num
	}
	fmt.Printf("\r\nLargest left number [%d]: ", config.MaxLeft)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxLeft = num
	}
	fmt.Printf("\r\nSmallest right number [%d]: ", config.MinRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinRight = num
	}
	fmt.Printf("\r\nLargest right number [%d]: ", config.MaxRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxRight = num
	}

	fmt.Printf("\r\nForce non-negative difference?%s", bracketCurrentOption(config.ForceNonnegativeDifference))
	setByInput(getCleanInput(reader), &config.ForceNonnegativeDifference)
}

func setupMultiplicationConfig(config *MultiplicationConfig, reader *bufio.Reader) {
	fmt.Printf("\r\nSmallest left number [%d]: ", config.MinLeft)
	line := getCleanInput(reader)
	num, err := strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinLeft = num
	}
	fmt.Printf("\r\nLargest left number [%d]: ", config.MaxLeft)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxLeft = num
	}
	fmt.Printf("\r\nSmallest right number [%d]: ", config.MinRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinRight = num
	}
	fmt.Printf("\r\nLargest right number [%d]: ", config.MaxRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxRight = num
	}
}

func setupDivisionConfig(config *DivisionConfig, reader *bufio.Reader) {
	fmt.Printf("\r\nSmallest left number [%d]: ", config.MinLeft)
	line := getCleanInput(reader)
	num, err := strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinLeft = num
	}
	fmt.Printf("\r\nLargest left number [%d]: ", config.MaxLeft)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		fmt.Printf("making change")
		config.MaxLeft = num
	}
	fmt.Printf("\r\nSmallest right number [%d]: ", config.MinRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MinRight = num
	}
	fmt.Printf("\r\nLargest right number [%d]: ", config.MaxRight)
	line = getCleanInput(reader)
	num, err = strconv.Atoi(line)
	if err == nil && len(line) > 0 {
		config.MaxRight = num
	}

	fmt.Printf("\r\nForce numbers to be evenly divisible?%s", bracketCurrentOption(config.ForceCleanDivision))
	setByInput(getCleanInput(reader), &config.ForceCleanDivision)
}
func setupConfig() {
	var config Config
	err := os.MkdirAll("configs", 0755)
	if err != nil {
		panic(err)
	}
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("Enter config name, or leave empty to override the default: ")
	configName := getCleanInput(reader)
	if len(configName) == 0 {
		fmt.Printf("\r\nModifying default config.")
		config.Name = "default"
		if fileExists("configs/default.txt") {
			config.Load("configs/default.txt")
		} else {
			config = GetZetamacConfig()
		}
	} else if fileExists("configs/" + configName + ".txt") {
		fmt.Printf("\r\nModifying existing config.")
		config.Load("configs/" + configName + ".txt")
	} else {
		fmt.Printf("\r\nInitializing new config.")
		config = GetZetamacConfig()
		config.Name = configName
	}

	fmt.Printf("\r\nModify game meta-settings? y/[n]: ")
	input := getCleanInput(reader)
	if input == "y" {
		fmt.Printf("\r\nEnter a new name for this config, or leave blank for unchanged [%s]: ", config.Name)
		input = getCleanInput(reader)
		if len(input) != 0 {
			config.Name = input
		}
		fmt.Printf("\r\nShould subtraction problems be addition problems in reverse?%s", bracketCurrentOption(config.OverrideSubtractionConfig))
		setByInput(getCleanInput(reader), &config.OverrideSubtractionConfig)
		fmt.Printf("\r\nShould divsion problems be multiplication problems in reverse?%s", bracketCurrentOption(config.OverrideDivisionConfig))
		setByInput(getCleanInput(reader), &config.OverrideDivisionConfig)
		fmt.Printf("\r\nEnter the desired game duration in seconds, or leave blank for unchanged [%d]: ", config.Duration)
		input = getCleanInput(reader)
		num, err := strconv.Atoi(input)
		if err == nil && len(input) > 0 {
			config.Duration = num
		}
		var ops []string
		fmt.Printf("\r\nEnable addition?%s", bracketCurrentOption(slices.Contains(config.LegalOperations, "+")))
		input = getCleanInput(reader)
		if input == "y" {
			ops = append(ops, "+")
		}
		fmt.Printf("\r\nEnable subtraction?%s", bracketCurrentOption(slices.Contains(config.LegalOperations, "-")))
		input = getCleanInput(reader)
		if input == "y" {
			ops = append(ops, "-")
		}
		fmt.Printf("\r\nEnable multiplication?%s", bracketCurrentOption(slices.Contains(config.LegalOperations, "*")))
		input = getCleanInput(reader)
		if input == "y" {
			ops = append(ops, "*")
		}
		fmt.Printf("\r\nEnable division?%s", bracketCurrentOption(slices.Contains(config.LegalOperations, "/")))
		input = getCleanInput(reader)
		if input == "y" {
			ops = append(ops, "/")
		}
		config.LegalOperations = ops
	}
	fmt.Printf("\r\nModify addition settings? y/[n]: ")
	if getCleanInput(reader) == "y" {
		setupAdditionConfig(&config.AdditionConfig, reader)
	}
	fmt.Printf("\r\nModify subtraction settings? y/[n]: ")
	if getCleanInput(reader) == "y" {
		setupSubtractionConfig(&config.SubtractionConfig, reader)
	}
	fmt.Printf("\r\nModify multiplication settings? y/[n]: ")
	if getCleanInput(reader) == "y" {
		setupMultiplicationConfig(&config.MultiplicationConfig, reader)
	}
	fmt.Printf("\r\nModify division settings? y/[n]: ")
	if getCleanInput(reader) == "y" {
		setupDivisionConfig(&config.DivisionConfig, reader)
	}

	config.Save("configs/" + config.Name + ".txt")
}

func validateConfig(config *Config) {
	fmt.Printf("%s", config.String())
	valid := true

	//game rules
	//1) no non-positive duration
	if config.Duration <= 0 {
		fmt.Printf("CONFIG ERROR: NON-POSITIVE GAME DURATION\r\n")
		valid = false
	}

	//addition rules
	//1) no non-positive operands
	//2) operands less than maxint/2
	//3) maxes >= mins
	if config.AdditionConfig.MinLeft < 0 || config.AdditionConfig.MinRight < 0 {
		fmt.Printf("CONFIG ERROR: NON-POSITIVE ADDITION OPERANDS\r\n")
		valid = false
	}
	if config.AdditionConfig.MaxLeft >= math.MaxInt/2 || config.AdditionConfig.MaxRight >= math.MaxInt/2 {
		fmt.Printf("CONFIG ERROR: ADDITION OPERANDS GREATER THAN HALF OF MAX INTEGER VALUE\r\n")
		valid = false
	}
	if config.AdditionConfig.MaxLeft < config.AdditionConfig.MinLeft || config.AdditionConfig.MaxRight < config.AdditionConfig.MinRight {
		fmt.Printf("CONFIG ERROR: NO POSSIBLE ADDITION OPERANDS\r\n")
		valid = false
	}

	//subtraction rules
	//1) no non-positive operands
	//2) if the option is set, leftmax > rightmin (so we can always generate difference of at least 0)
	//3) maxes >= mins
	if config.SubtractionConfig.MinLeft <= 0 || config.SubtractionConfig.MinRight <= 0 {
		fmt.Printf("CONFIG ERROR: NON-POSITIVE SUBTRACTION OPERANDS\r\n")
		valid = false
	}
	if config.SubtractionConfig.ForceNonnegativeDifference && config.SubtractionConfig.MaxLeft < config.SubtractionConfig.MinRight {
		fmt.Printf("CONFIG ERROR: NO POSSIBLE NON-NEGATIVE DIFFERENCES\r\n")
		valid = false
	}
	if config.SubtractionConfig.MaxLeft < config.SubtractionConfig.MinLeft || config.SubtractionConfig.MaxRight < config.SubtractionConfig.MinRight {
		fmt.Printf("CONFIG ERROR: NO POSSIBLE SUBTRACTION OPERANDS\r\n")
		valid = false
	}

	//multiplication rules
	//1) no non-positive operands
	//2) operands less than sqrt(maxint)
	//3) maxes >= mins
	if config.MultiplicationConfig.MinLeft <= 0 || config.MultiplicationConfig.MinRight <= 0 {
		fmt.Printf("CONFIG ERROR: NON-POSITIVE MULTIPLICATION OPERANDS\r\n")
		valid = false
	}
	if config.MultiplicationConfig.MaxLeft >= int(math.Sqrt(math.MaxInt)) || config.MultiplicationConfig.MaxRight >= int(math.Sqrt(math.MaxInt/2)) {
		fmt.Printf("CONFIG ERROR: MULTIPLICATION OPERANDS GREATER THAN SQUARE ROOT OF MAX INTEGER VALUE\r\n")
		valid = false
	}
	if config.MultiplicationConfig.MaxLeft < config.MultiplicationConfig.MinLeft || config.MultiplicationConfig.MaxRight < config.MultiplicationConfig.MinRight {
		fmt.Printf("CONFIG ERROR: NO POSSIBLE MULTIPLICATION OPERANDS\r\n")
		valid = false
	}

	//divison rules
	//1) no non-positive operands
	//2) leftmax >= rightmin (so we can always generate quotient of at least 1)
	//3) maxes >= mins
	if config.DivisionConfig.MinLeft <= 0 || config.DivisionConfig.MinRight <= 0 {
		fmt.Printf("CONFIG ERROR: NON-POSITIVE DIVISION OPERANDS\r\n")
		valid = false
	}
	if config.DivisionConfig.MaxLeft < config.DivisionConfig.MinRight {
		fmt.Printf("CONFIG ERROR: NO POSSIBLE NON-ZERO QUOTIENTS\r\n")
		valid = false
	}
	if config.DivisionConfig.MaxLeft < config.DivisionConfig.MinLeft || config.DivisionConfig.MaxRight < config.DivisionConfig.MinRight {
		fmt.Printf("CONFIG ERROR: NO POSSIBLE DIVISION OPERANDS\r\n")
		valid = false
	}
	if !valid {
		os.Exit(1)
	}
}

func main() {
	config := GetZetamacConfig()
	handleClargs(&config)

	switch mode {
	case StatsMode:
		printStats("scores.txt")
		return
	case ConfigMode: //TODO: implement config mode
		setupConfig()
		return
	case GameMode:
		validateConfig(&config)
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
