package mount

import (
	"context"
	"fmt"
	"github.com/ekristen/satokens/pkg/commands/global"
	"github.com/ekristen/satokens/pkg/common"
	"github.com/ekristen/satokens/pkg/kubeutils"
	"github.com/ekristen/satokens/pkg/tokenfs"
	"github.com/jacobsa/fuse"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

func Execute(c *cli.Context) error {
	// TODO: check if mount-path exists, and error
	// TODO: run mkdir -p
	// TODO: rmdir

	cfg, err := kubeconfig.GetNonInteractiveClientConfig(c.String("kubeconfig")).ClientConfig()
	if err != nil {
		return err
	}

	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	tunnel := kubeutils.NewTunnel(kube.RESTClient(), cfg, c.String("namespace"), c.String("pod-name"), 44044)
	defer tunnel.Close()

	if err := tunnel.ForwardPort(); err != nil {
		return err
	}

	server, err := tokenfs.NewTokenFS()
	if err != nil {
		return err
	}

	go func() {
		<-c.Context.Done()

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer func() {
			cancel()
		}()

		logrus.Info("attempting to unmount")
		if err := unmount(ctx, c.Path("mount-path")); err != nil {
			logrus.WithError(err).Error("unable to unmount path")
		}
	}()

	fcfg := &fuse.MountConfig{
		ReadOnly: true,
	}

	mfs, err := fuse.Mount(c.Path("mount-path"), server, fcfg)
	if err != nil {
		logrus.Fatalf("Mount: %v", err)
	}

	if err = mfs.Join(context.Background()); err != nil {
		logrus.Fatalf("Join: %v", err)
	}

	logrus.Info("unmount complete")

	return nil
}

func unmount(ctx context.Context, dir string) error {
	delay := 10 * time.Millisecond
	for {
		err := fuse.Unmount(dir)
		if err == nil {
			return err
		}

		if strings.Contains(err.Error(), "resource busy") {
			logrus.Warn("Resource busy error while unmounting; trying again")
			time.Sleep(delay)
			delay = time.Duration(1.3 * float64(delay))
			continue
		}

		return fmt.Errorf("unmount: %v", err)
	}
}

func init() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:  "pod-name",
			Usage: "pod-name",
			Value: "satokens",
		},
		&cli.StringFlag{
			Name:    "namespace",
			Usage:   "namespace to use for the pod",
			EnvVars: []string{"NAMESPACE"},
			Value:   "default",
		},
		&cli.PathFlag{
			Name:  "mount-path",
			Value: "/tmp/satokens",
		},
	}

	cliCmd := &cli.Command{
		Name:   "mount",
		Usage:  "mount the token to a local path",
		Action: Execute,
		Flags:  append(flags, global.Flags()...),
		Before: global.Before,
	}

	common.RegisterCommand(cliCmd)
}
