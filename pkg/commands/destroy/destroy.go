package destroy

import (
	"github.com/ekristen/satokens/pkg/commands/global"
	"github.com/ekristen/satokens/pkg/common"
	"github.com/rancher/wrangler/pkg/apply"
	corev1client "github.com/rancher/wrangler/pkg/generated/controllers/core"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"k8s.io/apimachinery/pkg/runtime"
	"time"
)

func Execute(c *cli.Context) error {
	cfg, err := kubeconfig.GetNonInteractiveClientConfig(c.String("kubeconfig")).ClientConfig()
	if err != nil {
		return err
	}

	apply, err := apply.NewForConfig(cfg)
	if err != nil {
		return err
	}

	core, err := corev1client.NewFactoryFromConfig(cfg)
	if err != nil {
		return err
	}

	if err := core.Start(c.Context, 50); err != nil {
		return err
	}

	if err := core.Sync(c.Context); err != nil {
		return err
	}

	_ = core.Core().V1().Pod().Cache()
	_ = core.Core().V1().ServiceAccount().Cache()

	time.Sleep(10 * time.Second)

	var objects = make([]runtime.Object, 0)

	if err := apply.
		WithSetID("satokens").
		WithDynamicLookup().
		WithStrictCaching().
		WithCacheTypes(core.Core().V1().ServiceAccount(), core.Core().V1().Pod()).
		ApplyObjects(objects...); err != nil {
		return err
	}

	logrus.Info("destruction successful")

	return nil
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
	}

	cliCmd := &cli.Command{
		Name:   "destroy",
		Usage:  "remove the pod from the cluster",
		Action: Execute,
		Flags:  append(flags, global.Flags()...),
		Before: global.Before,
	}

	common.RegisterCommand(cliCmd)
}
