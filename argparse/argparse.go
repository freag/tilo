package argparse

import (
	"fmt"
	"github.com/fgahr/tilo/msg"
	"github.com/pkg/errors"
	"os"
	"strings"
)

const (
	ParamIdentifierPrefix = ":"
	// TODO: Should it be a public constant here? Other options? Package-private?
	AllTasks string = ParamIdentifierPrefix + "all"
)

type taskHandler interface {
	handleTasks(cmd *msg.Cmd, args []string) ([]string, error)
	// TODO: Better options here? Write to a writer?
	describe() string
}

type noTaskHandler struct{}

func (h noTaskHandler) handleTasks(cmd *msg.Cmd, args []string) ([]string, error) {
	return args, nil
}

func (h noTaskHandler) describe() string {
	return ""
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

func (h singleTaskHandler) describe() string {
	return "task"
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

func (h multiTaskHandler) describe() string {
	return AllTasks + "|task,..."
}

type ArgHandler interface {
	// Parse the params and modify cmd accordingly.
	// Returns unused arguments and a possible error.
	HandleArgs(cmd *msg.Cmd, args []string) ([]string, error)
}

type noArgHandler struct{}

func (h noArgHandler) HandleArgs(cmd *msg.Cmd, args []string) ([]string, error) {
	return args, nil
}

// TODO: Move methods to builder?
type Parser struct {
	command     string
	taskHandler taskHandler
	argHandler  ArgHandler
}

func CommandParser(command string) *Parser {
	return &Parser{command: command, taskHandler: nil, argHandler: nil}
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
	p.argHandler = new(noArgHandler)
	return p
}

func (p *Parser) WithArgHandler(h ArgHandler) *Parser {
	p.argHandler = h
	return p
}

// Parse the given arguments.
func (p *Parser) Parse(args []string) (msg.Cmd, error) {
	cmd := msg.Cmd{Op: p.command}
	if p.taskHandler == nil {
		panic("Argument parser does not know how to handle tasks")
	}
	restArgs, err := p.taskHandler.handleTasks(&cmd, args)
	if err != nil {
		return cmd, err
	}
	if p.argHandler == nil {
		panic("Argument parser does not know how to handle parameters")
	}
	unusedArgs, err := p.argHandler.HandleArgs(&cmd, restArgs)
	if err != nil {
		return cmd, err
	} else {
		WarnUnused(unusedArgs)
		return cmd, nil
	}
}

// Warn the user about arguments being unevaluated.
// If args is empty, no warning is issued.
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
	if isParamIdentifier(name) {
		return false
	} else if hasWhitespace(name) {
		return false
	}
	return true
}

func stripKeyword(raw string) string {
	return strings.TrimLeft(raw, ":")
}

func hasWhitespace(str string) bool {
	return strings.ContainsAny(str, " \t\n")
}

func SingleQuantity(t string, elems ...string) []msg.Quantity {
	return []msg.Quantity{msg.Quantity{Type: t, Elems: elems}}
}

type Quantifier interface {
	Parse(str string) ([]msg.Quantity, error)
	DescribeUsage() string
}

type Param struct {
	Name        string
	RequiresArg bool
	Quantifier  Quantifier
	Description string
}

type paramHandler struct {
	params map[string]Param
}

func (p paramHandler) HandleArgs(cmd *msg.Cmd, args []string) ([]string, error) {
	quant := []msg.Quantity{}
	unused := []string{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if isParamIdentifier(arg) {
			if param, ok := p.params[cleanParam(arg)]; ok {
				pArg := ""
				if param.RequiresArg {
					if strings.Contains(arg, "=") {
						// Quantity contained in argument.
						pArg = strings.Split(arg, "=")[1]
					} else {
						// Quantity in next argument.
						i++
						if i == len(args) {
							return args, errors.New("No argument for parameter " + param.Name)
						}
						pArg = args[i]
					}
				} else {
					// If no arg is required, we can pass the empty string.
				}
				// Parse and add to list.
				q, err := param.Quantifier.Parse(pArg)
				if err != nil {
					return unused, err
				}
				quant = append(quant, q...)
			}
		} else {
			unused = append(unused, arg)
		}
	}
	return unused, nil
}

func HandlerForParams(params []Param) ArgHandler {
	var pmap map[string]Param
	for _, param := range params {
		if _, ok := pmap[param.Name]; ok {
			panic("Duplicate param name: " + param.Name)
		}
		pmap[param.Name] = param
	}

	return paramHandler{params: pmap}
}

func isParamIdentifier(str string) bool {
	return strings.HasPrefix(str, ParamIdentifierPrefix)
}

func cleanParam(str string) string {
	return strings.TrimLeft(strings.Split(str, "=")[0], ParamIdentifierPrefix)
}
