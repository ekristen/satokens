package destroy

import (
	"github.com/ekristen/satokens/pkg/commands/global"
	"github.com/ekristen/satokens/pkg/common"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/urfave/cli/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func Execute(c *cli.Context) error {
	cfg, err := kubeconfig.GetNonInteractiveClientConfig(c.String("kubeconfig")).ClientConfig()
	if err != nil {
		return err
	}

	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	pods := kube.CoreV1().Pods(c.String("namespace"))

	if err := pods.Delete(c.Context, c.String("pod-name"), metav1.DeleteOptions{}); err != nil {
		return err
	}

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
