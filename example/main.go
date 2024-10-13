package main

import (
	"context"
	"fmt"
	"time"

	zaprpc "github.com/achyuthcodes30/ZapRPC"
)

// CalculatorService defines the interface for our calculator service
type CalculatorService interface {
	Add(a, b int) int
	Subtract(a, b int) int
	Multiply(a, b int) int
	Divide(a, b int) (float64, error)
}

// CalculatorServiceImpl implements the CalculatorService interface
type CalculatorServiceImpl struct{}

func (c *CalculatorServiceImpl) Add(a, b int) int {
	return a + b
}

func (c *CalculatorServiceImpl) Subtract(a, b int) int {
	return a - b
}

func (c *CalculatorServiceImpl) Multiply(a, b int) int {
	return a * b
}

func (c *CalculatorServiceImpl) Divide(a, b int) (float64, error) {
	if b == 0 {
		return 0, fmt.Errorf("division by zero")
	}
	return float64(a) / float64(b), nil
}

func clientMain() {
	CalcConn, _ := zaprpc.NewConn(context.Background(), "localhost:5000", nil)
	additionResult, _ := zaprpc.Zap(context.Background(), CalcConn, "Calculator.Add", 10, 20)
	fmt.Println(additionResult)

}

func serverMain() {
	CalcServer := zaprpc.NewZapServer()
	CalcServer.RegisterService("Calculator", &CalculatorServiceImpl{})
	CalcServer.Serve(5000, nil)

}

func main() {
	go func() {
		serverMain()
	}()
	time.Sleep(time.Second * 1)
	clientMain()

}
