package argparse

import (
	"fmt"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"os"
	"strings"
	"time"
)

// Types of task list
type tlist int

type taskHandler interface {
	handleTasks(cmd *msg.Cmd, args []string) ([]string, error)
}

type noTaskHandler struct{}

func (h noTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	return args, nil
}

type singleTaskHandler struct{}

func (h singleTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, errors.New("Require single task but none is given")
	}
	if tasks, err := GetTaskNames(args[0]); err != nil {
		return args, err
	} else if len(tasks) == 0 {
		return args, errors.New("Require single task but none is given")
	} else if len(tasks) > 1 {
		return args, errors.New("Require single task but several are given")
	} else if tasks[0] == AllTasks {
		return args, errors.New("Require single task name but found '" + AllTasks + "'")
	} else {
		cmd.Tasks = tasks
	}
	return args[1:], nil
}

type multiTaskHandler struct{}

func (h multiTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	if len(args) == 0 {
		return args, errors.New("Require one or more tasks but none is given")
	}
	if tasks, err := GetTaskNames(args[0]); err != nil {
		return args, err
	} else if len(tasks) == 0 {
		return args, errors.New("Require one or more tasks but none is given")
	} else {
		if len(tasks) > 1 {
			for _, task := range tasks {
				if task == AllTasks {
					return args, errors.New("When given, '" + AllTasks + "' must be the only task")
				}
			}
		}
		cmd.Tasks = tasks
	}
	return args[1:], nil
}

type ParamHandler interface {
	HandleParams(cmd *msg.Cmd, params []string) ([]string, error)
}

type noParamHandler struct{}

func (h noParamHandler) HandleParams(cmd *msg.Cmd, params []string) ([]string, error) {
	return params, nil
}

// TODO: Move methods to builder?
type Parser struct {
	taskHandler  taskHandler
	paramHandler ParamHandler
}

func NewParser() *Parser {
	return &Parser{taskHandler: nil, paramHandler: nil}
}

func (p *Parser) WithoutTask() *Parser {
	p.taskHandler = new(noTaskHandler)
	return p
}

func (p *Parser) WithSingleTask() *Parser {
	p.taskHandler = new(singleTaskHandler)
	return p
}

func (p *Parser) WithMultipleTasks() *Parser {
	p.taskHandler = new(multiTaskHandler)
	return p
}

func (p *Parser) WithoutParams() *Parser {
	p.paramHandler = new(noParamHandler)
	return p
}

func (p *Parser) WithParamHandler(h ParamHandler) *Parser {
	p.paramHandler = h
	return p
}

// Parse the given arguments.
func (p *Parser) Parse(args []string) (msg.Cmd, error) {
	cmd := msg.Cmd{}
	if p.taskHandler == nil {
		panic("Argument parser does not know how to handle tasks")
	}
	restArgs, err := p.taskHandler.handleTasks(&cmd, args)
	if err != nil {
		return cmd, err
	}
	if p.paramHandler == nil {
		panic("Argument parser does not know how to handle parameters")
	}
	unusedArgs, err := p.paramHandler.HandleParams(&cmd, restArgs)
	if err != nil {
		return cmd, err
	} else {
		WarnUnused(unusedArgs)
		return cmd, nil
	}
}

const (
	// FIXME: Shouldn't be a constant here.
	AllTasks string = ":all"
)

// Warn the user about arguments being unevaluated.
func WarnUnused(args []string) {
	if len(args) > 0 {
		fmt.Fprintln(os.Stderr, "Ignoring unused arguments:", args)
	}
}

// Split task names given as a comma-separated field, check for validity.
func GetTaskNames(taskField string) ([]string, error) {
	if taskField == AllTasks {
		return []string{AllTasks}, nil
	}

	tasks := strings.Split(taskField, ",")
	for _, task := range tasks {
		if !validTaskName(task) {
			return nil, errors.Errorf("Invalid task name: %s", task)
		}
	}
	return tasks, nil
}

// Whether the given name is valid for a task.
func validTaskName(name string) bool {
	if strings.HasPrefix(name, ":") {
		return false
	} else if hasWhitespace(name) {
		return false
	}
	return true
}

func isKeyword(word string) bool {
	return strings.HasPrefix(word, ":") || !hasWhitespace(word)
}

func stripKeyword(raw string) string {
	return strings.TrimLeft(raw, ":")
}

func hasWhitespace(str string) bool {
	return strings.ContainsAny(str, " \t\n")
}

// TODO
type Quantity struct {
	// TODO
}

func (q *Quantity) Add(more Quantity) {
	// TODO
}

type Quantifier interface {
	Parse(str string) (Quantity, error)
	Describe() string
}

type ListQuantifier struct {
	elem Quantifier
}

func ListQuantifierOf(elem Quantifier) Quantifier {
	return ListQuantifier{elem}
}

func (lq ListQuantifier) Parse(str string) (Quantity, error) {
	qnt := Quantity{}
	for _, part := range strings.Split(str, ",") {
		nxt, err := lq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt.Add(nxt)
	}
	return qnt, nil
}

func (lq ListQuantifier) Describe() string {
	return fmt.Sprintf("%s,...", lq.elem.Describe())
}

type PairQuantifier struct {
	elem Quantifier
}

func PairQuantifierOf(elem Quantifier) Quantifier {
	return PairQuantifier{elem}
}

func (pq PairQuantifier) Parse(str string) (Quantity, error) {
	qnt := Quantity{}
	fields := strings.Split(str, ":")
	if len(fields) != 2 {
		return qnt, errors.New("Not a pair: " + str)
	}
	for _, part := range fields {
		nxt, err := pq.elem.Parse(part)
		if err != nil {
			return qnt, err
		}
		qnt.Add(nxt)
	}
	return qnt, nil
}

func (pq PairQuantifier) Describe() string {
	return fmt.Sprintf("%s:%[1]s", pq.elem.Describe())
}

type DateQuantifier struct{}

func (dq DateQuantifier) Parse(str string) (Quantity, error) {
	// TODO: Fix once Quantity is defined
	_, err := time.Parse("2006-01-02", str)
	return Quantity{}, err
}

func (dq DateQuantifier) Describe() string {
	return "YYYY-MM-DD"
}

type MonthQuantifier struct{}

func (mq MonthQuantifier) Parse(str string) (Quantity, error) {
	// TODO: Fix once Quantity is defined
	_, err := time.Parse("2006-01", str)
	return Quantity{}, err
}

func (mq MonthQuantifier) Describe() string {
	return "YYYY-MM"
}

type YearQuantifier struct{}

func (yq YearQuantifier) Parse(str string) (Quantity, error) {
	// TODO: Fix once Quantity is defined
	_, err := time.Parse("2006", str)
	return Quantity{}, err
}

func (yq YearQuantifier) Describe() string {
	return "YYYY"
}
