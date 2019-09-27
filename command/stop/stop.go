package stop

import (
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
)

type StopOperation struct {
	// No state required
}

func (op StopOperation) Command() string {
	return "stop"
}

func (op StopOperation) Parser() *argparse.Parser {
	return argparse.CommandParser(op.Command()).WithoutTask().WithoutParams()
}

func (op StopOperation) ClientExec(cl *client.Client, cmd msg.Cmd) error {
	cl.EstablishConnection()
	cl.SendToServer(cmd)
	resp := cl.ReceiveFromServer()
	cl.PrintResponse(resp)
	return errors.Wrap(cl.Error(), "Failed to stop the current task")
}

func (op StopOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	task, stopped := srv.StopCurrentTask()
	if stopped {
		if err := srv.SaveTask(task); err != nil {
			resp.SetError(err)
		}
		resp.AddStoppedTask(task)
	} else {
		resp.SetError(errors.New("No active task"))
	}
	return srv.Answer(req, resp)
}

func (op StopOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Stop the current task",
		LongDescription:  "Stop the current task",
	}
}

func init() {
	command.RegisterOperation(StopOperation{})
}
