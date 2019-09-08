// Package client describes all client-side operations.
package client

import (
	"fmt"
	"github.com/fgahr/tilo/config"
	"github.com/fgahr/tilo/msg"
	"github.com/fgahr/tilo/server"
	"github.com/pkg/errors"
	"net"
	"os"
	"text/tabwriter"
	"time"
	"net/rpc/jsonrpc"
)

// A struct holding a connection to the server and performing communication
// with it.
type Client struct {
	conn           net.Conn
	requestTimeout time.Duration
	Conf           *config.Params
}

// Create a new client to communicate with the server.
func NewClient(params *config.Params) (*Client, error) {
	c := Client{
		conn:           nil,
		requestTimeout: 5 * time.Second,
		Conf:           params,
	}
	return &c, nil
}

// Interact with the server based on the program's line arguments.
func (c *Client) HandleArgs(args []string) error {
	err := c.ensureServerIsRunning()
	if err != nil {
		return err
	}

	request, err := msg.ParseRequest(args, time.Now())
	if err != nil {
		return err
	}

	err = c.performRequest(request)
	return err
}

// Close the client's connection to the server.
func (c *Client) Close() error {
	if c.conn == nil {
		return errors.New("Client is not connected.")
	}
	return c.conn.Close()
}

// Perform a request-response-cycle, evaluating the server response to the request.
func (c *Client) performRequest(req msg.Request) error {
	rpcClient, err := jsonrpc.Dial("unix", c.Conf.Socket())
	if err != nil {
		return err
	}
	var resp msg.Response
	err = rpcClient.Call("RequestHandler.HandleRequest", req, &resp)
	if err != nil {
		return err
	}

	err = resp.Err()
	if err != nil {
		return errors.Wrap(err, "Error while processing request")
	} else {
		return c.printResponse(resp)
	}
}

// Print a response as formatted output.
func (c *Client) printResponse(resp msg.Response) error {
	// NOTE: This function could easily exist without depending on a client.
	// However, this allows to configure the output in some way at a later date.
	w := tabwriter.NewWriter(os.Stdout, 0, 4, 1, ' ', 0)
	for _, line := range resp.Body {
		noTab := true
		for _, word := range line {
			if noTab {
				noTab = false
			} else {
				fmt.Fprint(w, "\t")
			}
			fmt.Fprint(w, word)
		}
		fmt.Fprint(w, "\n")
	}
	return w.Flush()
}

// If the server is not already running, start it in a new background thread
// and wait for it to come online.
func (c *Client) ensureServerIsRunning() error {
	// If connected we already know it is running.
	if c.conn != nil {
		return nil
	}

	// Query server status.
	running, err := server.IsRunning(c.Conf)
	if err != nil {
		return err
	}

	// Start server if it isn't running.
	// FIXME: This section seems to be broken, always running into a timeout.
	if !running {
		err := server.StartInBackground(c.Conf)
		if err != nil {
			return err
		}

		// Wait for server to become available
		notifyChan := make(chan struct{})
		go func(ch chan<- struct{}) {
			for {
				up, _ := server.IsRunning(c.Conf)
				if up {
					ch <- struct{}{}
					return
				}
				time.Sleep(100 * time.Millisecond)
			}
		}(notifyChan)
		select {
		case <-notifyChan:
			return nil
		case <-time.After(5 * time.Second):
			close(notifyChan)
			return errors.New("Timeout exceeded trying to bring up server.")
		}
	}
	return nil
}
