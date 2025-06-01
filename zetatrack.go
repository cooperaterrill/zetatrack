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

var GameloopTime int = 120

const ClearSignal = "clear"

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
			if buf[0] == 0x7f {
				if answerBufFront <= 0 {
					continue
				}
				answerBufFront--
				fmt.Printf("\b \b")
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

func main() {
	handleClargs()
	timer := time.NewTimer(time.Duration(GameloopTime) * time.Second)

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

	score := 0
	go func() {
		<-timer.C
		fmt.Printf("\r\nScore: %d\r\n", score)
		os.Exit(0)
	}()

	inputChannel := make(chan string)
	go readInput(buf, inputChannel)

	firstProblem := true
	for {
		problem := genProblem(1, 12, []string{"+", "-", "*", "/"}, 1, 99)
		problemString := fmt.Sprintf("%d %s %d", problem.FirstNum, problem.Operation, problem.SecondNum)
		if firstProblem {
			fmt.Printf("%s: ", problemString)
			firstProblem = false
		} else {
			fmt.Printf("\r\n%s: ", problemString)
		}
		expression, err := govaluate.NewEvaluableExpression(problemString)
		if err != nil {
			fmt.Print("Error parsing expression\r\n")
			panic(err)
		}
		evaluated, err := expression.Evaluate(nil)
		if err != nil {
			fmt.Print("Error evaluating expression\r\n")
		}
		questionAns := int(evaluated.(float64))
		for {
			userAns := <-inputChannel
			if userAns == strconv.Itoa(questionAns) {
				//fmt.Printf("\r\nYou got the right answer\r\n")
				score++
				inputChannel <- ClearSignal
				break
			}
		}
	}
}
