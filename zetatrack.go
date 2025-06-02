package main

import (
	"fmt"
	"os"
	"strconv"
	"syscall"
	"time"

	"math/rand/v2"

	"github.com/Knetic/govaluate"
	"golang.org/x/term"
)

type Problem struct {
	FirstNum  int
	Operation string
	SecondNum int
}

func (problem Problem) String() string {
	return fmt.Sprintf("%d %s %d", problem.FirstNum, problem.Operation, problem.SecondNum)
}

var GameloopTime int = 120

const ClearSignal = "clear"
const QuitSignal = "quit"

//func fileExists(filepath string) bool {
//	_, err := os.Stat(filepath)
//	if err != nil {
//		if os.IsNotExist(err) {
//			return false
//		}
//	}
//	return true
//}

func handleClargs() {
	clargs := os.Args
	for i := 1; i < len(os.Args)-1; i++ {
		if clargs[i] == "-t" {
			num, err := strconv.Atoi(clargs[i+1])
			if err != nil {
				fmt.Printf("Enter an integer number of seconds\r\n")
			}
			GameloopTime = num
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
			//fmt.Printf("Read: %q", buf[0])
			if (buf[0] < 48 || buf[0] > 57) && buf[0] != 0x7f && buf[0] != 'q' {
				continue
			}
			if buf[0] == 0x7f {
				if answerBufFront <= 0 {
					continue
				}
				answerBufFront--
				fmt.Printf("\b \b")
			} else if buf[0] == 'q' {
				channel <- QuitSignal
			} else {
				if answerBufFront >= len(answerBuf) {
					continue
				}
				answerBuf[answerBufFront] = buf[0]
				answerBufFront++
			}
			fmt.Printf("%c", buf[0])
			//fmt.Printf("Current answer: %s\n", answerBuf[0:answerBufFront])
			channel <- string(answerBuf[0:answerBufFront])
		}
		time.Sleep(time.Millisecond * 5)
	}
}

func randRange(min int, max int) int {
	return rand.IntN(max-min+1) + min
}

func genProblem(firstNumMinValue int, firstNumMaxValue int, legalOps []string, secondNumMinValue int, secondNumMaxValue int) Problem {
	var problem Problem
	problem.Operation = legalOps[randRange(0, len(legalOps)-1)]

	problem.FirstNum = randRange(firstNumMinValue, firstNumMaxValue)
	problem.SecondNum = randRange(secondNumMinValue, secondNumMaxValue)

	if problem.Operation == "/" {
		if problem.FirstNum < problem.SecondNum {
			problem.FirstNum, problem.SecondNum = problem.SecondNum, problem.FirstNum
		}
		problem.FirstNum -= problem.FirstNum % problem.SecondNum
	}
	if problem.Operation == "-" {
		if problem.FirstNum < problem.SecondNum {
			problem.FirstNum, problem.SecondNum = problem.SecondNum, problem.FirstNum
		}
	}
	return problem
}

func getProblemAnswer(problemString string) int {
	expression, err := govaluate.NewEvaluableExpression(problemString)
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

func saveScores(problems []Problem, times []int64, filepath string) {
	var file *os.File
	var err error
	file, err = os.OpenFile(filepath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	file.WriteString(time.Now().String() + " ")
	for i := 0; i < len(problems)-1; i++ {
		file.WriteString(problems[i].String() + " " + strconv.FormatInt(times[i], 10) + " ")
	}
	file.WriteString(problems[len(problems)-1].String() + " " + "-1" + " ")
}

func gameLoop(inputChannel chan string) {
	var problems []Problem
	var times []int64
	score := 0
	timer := time.NewTimer(time.Duration(GameloopTime) * time.Second)
	firstProblem := true
	cleanup := func() {
		fmt.Printf("\r\nScore: %d\r\n", score)
		saveScores(problems, times, "scores.txt")

		return
	}
	defer cleanup()

	go func() {
		<-timer.C
		cleanup()
		os.Exit(0)
	}()

	for {
		problem := genProblem(1, 12, []string{"+", "-", "*", "/"}, 1, 99)
		problemAns := getProblemAnswer(problem.String())
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

func main() {
	handleClargs()

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		panic(err)
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	fd := int(os.Stdin.Fd())
	var buf = make([]byte, 1)

	err = syscall.SetNonblock(fd, true)
	if err != nil {
		panic(err)
	}

	inputChannel := make(chan string)
	go readInput(buf, inputChannel)

	gameLoop(inputChannel)
}
