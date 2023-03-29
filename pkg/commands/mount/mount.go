package mount

import (
	"context"
	"fmt"
	"github.com/ekristen/satokens/pkg/commands/global"
	"github.com/ekristen/satokens/pkg/common"
	"github.com/ekristen/satokens/pkg/portforward"
	"github.com/ekristen/satokens/pkg/tokenfs"
	"github.com/jacobsa/fuse"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/kubernetes"
	"strings"
	"time"
)

func Before(c *cli.Context) error {
	if err := global.Before(c); err != nil {
		return err
	}

	if c.Args().Len() == 1 {
		c.Set("mount-path", c.Args().First())
	}

	return nil
}

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

	go func() {
		opts := portforward.PortForwardOptions{
			Config:        cfg,
			RESTClient:    kube.CoreV1().RESTClient(),
			Namespace:     c.String("namespace"),
			PodName:       c.String("pod-name"),
			PodClient:     kube.CoreV1(),
			Address:       []string{"0.0.0.0"},
			Ports:         []string{"44044:44044"},
			PortForwarder: portforward.DefaultPortForwarder{},
			StopChannel:   make(chan struct{}, 1),
			ReadyChannel:  make(chan struct{}),
		}

		logrus.Info("connecting to satokens pod in cluster")

		if err := opts.RunPortForward(); err != nil {
			logrus.WithError(err).Error("unable to run port forward")
		}
	}()

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

	logrus.Info("starting token filesystem")

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
			Name:    "pod-name",
			Usage:   "pod-name",
			EnvVars: []string{"POD_NAME"},
			Value:   "satokens",
		},
		&cli.StringFlag{
			Name:    "namespace",
			Usage:   "namespace to use for the pod",
			EnvVars: []string{"NAMESPACE"},
			Value:   "default",
		},
		&cli.PathFlag{
			Name:     "mount-path",
			EnvVars:  []string{"MOUNT_PATH"},
			Required: true,
		},
	}

	cliCmd := &cli.Command{
		Name:   "mount",
		Usage:  "mount the token to a local path",
		Action: Execute,
		Flags:  append(flags, global.Flags()...),
		Before: Before,
	}

	common.RegisterCommand(cliCmd)
}
