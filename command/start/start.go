package start

import (
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
)

type StartOperation struct {
	// No state required
}

func (op StartOperation) Command() string {
	return "start"
}

func (op StartOperation) ClientExec(conf *config.Opts, args ...string) error {
	// TODO: Parse arguments, extract task name
	taskName := "foo"
	clientCmd := command.Cmd{
		Op:   op.Command(),
		Body: [][]string{[]string{taskName}},
	}

	if err := client.EnsureServerIsRunning(conf); err != nil {
		return errors.Wrapf(err, "Cannot start task '%s'", taskName)
	}

	if resp, err := client.SendToServer(conf, clientCmd); err != nil {
		return errors.Wrapf(err, "Failed to start task '%s'", taskName)
	} else {
		return client.PrintResponse(conf, resp)
	}
}

func (op StartOperation) ServerExec(srv *server.Server, _ net.Conn, cmd command.Cmd, resp *msg.Response) {
	taskName := cmd.Body[0][0]
	task, stopped := srv.StopCurrentTask()
	if stopped {
		if err := srv.SaveTask(task); err != nil {
			resp.SetError(err)
		}
		resp.AddStoppedTask(task)
	}
	srv.SetActiveTask(taskName)
}

func (op StartOperation) Help() command.Doc {
	// TODO: Improve, figure out what's required
	return command.Doc{
		ShortDescription: "Start a task",
		LongDescription:  "Start a task",
		Arguments:        []string{"<task>"},
	}
}

func init() {
	command.RegisterOperation(StartOperation{})
}
