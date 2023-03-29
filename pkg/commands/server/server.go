package server

import (
	"context"
	"github.com/ekristen/satokens/pkg/commands/global"
	"github.com/ekristen/satokens/pkg/common"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http"
	"os"
	"time"
)

type Payload struct {
	Contents []byte `json:"contents"`
}

func Execute(c *cli.Context) error {
	router := mux.NewRouter().StrictSlash(true)
	router.Path("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := os.ReadFile(c.Path("path"))
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if err := json.NewEncoder(w).Encode(Payload{
			Contents: data,
		}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	})

	srv := &http.Server{
		Addr:    c.String("addr"),
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Fatalf("listen: %s\n", err)
		}
	}()
	logrus.Info("starting server")

	<-c.Context.Done()

	logrus.Info("shutting down server")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer func() {
		cancel()
	}()

	if err := srv.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("unable to shutdown the api server gracefully")
		return err
	}

	return nil
}

func init() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "path",
			Usage: "the path to the token file",
			Value: "",
		},
		&cli.StringFlag{
			Name:  "addr",
			Usage: "the address to host the server on",
			Value: ":44044",
		},
	}

	cliCmd := &cli.Command{
		Name:   "server",
		Usage:  "run the remote http server that is connected to from the mount command",
		Action: Execute,
		Flags:  append(flags, global.Flags()...),
		Before: global.Before,
		Hidden: true,
	}

	common.RegisterCommand(cliCmd)
}
