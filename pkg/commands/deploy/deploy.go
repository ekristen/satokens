package deploy

import (
	"fmt"
	"github.com/ekristen/satokens/pkg/commands/global"
	"github.com/ekristen/satokens/pkg/common"
	"github.com/rancher/wrangler/pkg/apply"
	"github.com/rancher/wrangler/pkg/kubeconfig"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"path/filepath"
	"strings"

	_ "github.com/rancher/wrangler/pkg/generated/controllers/core"
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

	pathParts := strings.Split(c.String("path"), "/")
	filePath := filepath.Join(pathParts[len(pathParts)-1])
	newParts := pathParts[:len(pathParts)-1]
	dirPath := filepath.Join(newParts...)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.String("pod-name"),
			Namespace: c.String("namespace"),
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "server",
					Image: c.String("image"),
					Command: []string{
						"satokens",
					},
					Args: []string{
						"server",
						fmt.Sprintf("--path=%s", c.Path("path")),
					},
					Ports: []corev1.ContainerPort{
						{
							Name:          "server",
							ContainerPort: 44044,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "sa-token",
							MountPath: dirPath,
						},
					},
				},
			},
			Volumes: []corev1.Volume{
				{
					Name: "sa-token",
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Path:              filePath,
										ExpirationSeconds: &[]int64{c.Int64("expiration")}[0],
										Audience:          c.String("audience"),
									},
								},
							},
						},
					},
				},
			},
		},
	}

	logrus.Info("creating/update pod")

	if err := apply.
		WithSetID("satokens").
		WithSetOwnerReference(false, false).
		WithDynamicLookup().
		ApplyObjects(pod); err != nil {
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
		&cli.StringFlag{
			Name:  "path",
			Usage: "the path to the token file",
			Value: "/var/run/secrets/satokens/token",
		},
		&cli.Int64Flag{
			Name:  "expiration",
			Usage: "token expiration in seconds",
			Value: 7200,
		},
		&cli.StringFlag{
			Name:  "audience",
			Value: "sts.amazonaws.com",
		},
		&cli.StringFlag{
			Name:  "image",
			Usage: "image",
			Value: fmt.Sprintf("ghcr.io/ekristen/satokens:%s", common.AppVersion.Summary),
		},
	}

	cliCmd := &cli.Command{
		Name:   "deploy",
		Usage:  "deploy pod to the cluster",
		Action: Execute,
		Flags:  append(flags, global.Flags()...),
		Before: global.Before,
	}

	common.RegisterCommand(cliCmd)
}
