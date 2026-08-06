// Command tctl is the Tennon operator/dev CLI — a client over the Tennon API.
package main

import (
	"context"
	"os"
	"os/signal"

	"github.com/yurifrl/tennon/cli"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if err := cli.NewRootCmd().ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
