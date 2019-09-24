package ping

import (
	"fmt"
	"github.com/fgahr/tilo/argparse"
	"github.com/fgahr/tilo/client"
	"github.com/fgahr/tilo/command"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"os"
	"time"
)

type PingOperation struct {
	// No state required
}

func (op PingOperation) Command() string {
	return "ping"
}

func (op PingOperation) ClientExec(cl *client.Client, args ...string) error {
	argparse.WarnUnused(args)
	pingCmd := msg.Cmd{Op: op.Command()}
	before := time.Now()
	if _, err := fmt.Fprintln(os.Stderr, "Sending ping to server"); err != nil {
		return err
	}
	cl.SendToServer(pingCmd)
	cl.ReceiveFromServer() // Ignoring response
	after := time.Now()
	if cl.Failed() {
		return cl.Error()
	}
	_, err := fmt.Fprintf(os.Stderr, "Received pong from server after %v\n", after.Sub(before))
	return err
}

func (op PingOperation) ServerExec(srv *server.Server, req *server.Request) error {
	defer req.Close()
	resp := msg.Response{}
	resp.Status = msg.RespSuccess
	resp.AddPong()
	return srv.Answer(req, resp)
}

func (op PingOperation) Help() command.Doc {
	return command.Doc{
		ShortDescription: "Check whether the server is running",
		LongDescription:  "Check whether the server is running",
	}
}

func init() {
	command.RegisterOperation(PingOperation{})
}