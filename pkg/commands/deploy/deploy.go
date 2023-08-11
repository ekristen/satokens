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
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/rancher/wrangler/pkg/generated/controllers/apps"
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

	kube, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}

	pathParts := strings.Split(c.String("path"), "/")
	filePath := filepath.Join(pathParts[len(pathParts)-1])
	newParts := pathParts[:len(pathParts)-1]
	dirPath := filepath.Join(newParts...)

	/*
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.String("pod-name"),
				Namespace: c.String("namespace"),
			},
			Spec: appsv1.DeploymentSpec{
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app": "satokens",
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app": "satokens",
						},
					},
					Spec: ,
				},
			},
		}
	*/

	var objects []runtime.Object

	if c.Bool("create-service-account") {
		sa := &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.String("service-account-name"),
				Namespace: c.String("namespace"),
			},
		}

		objects = append(objects, sa)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.String("pod-name"),
			Namespace: c.String("namespace"),
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: c.String("service-account-name"),
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

	objects = append(objects, pod)

	for {
		err := apply.
			WithSetID("satokens").
			WithDynamicLookup().
			WithPatcher(corev1.SchemeGroupVersion.WithKind("Pod"), func(namespace, name string, pt types.PatchType, data []byte) (runtime.Object, error) {
				err := kube.CoreV1().Pods(namespace).Delete(c.Context, name, metav1.DeleteOptions{})
				if err == nil {
					return nil, fmt.Errorf("replace pod")
				}
				return nil, err
			}).
			ApplyObjects(objects...)
		if err != nil && !strings.Contains(err.Error(), "replace pod") {
			return err
		}
		if err != nil && strings.Contains(err.Error(), "replace pod") {
			logrus.Info("replacing existing pod")
			time.Sleep(3 * time.Second)
			continue
		}

		logrus.Info("creating pod")

		break
	}

	return nil
}

func init() {
	flags := []cli.Flag{
		&cli.StringFlag{
			Name:    "pod-name",
			Usage:   "pod-name",
			Value:   "satokens",
			EnvVars: []string{"POD_NAME"},
		},
		&cli.StringFlag{
			Name:    "namespace",
			Usage:   "namespace to use for the pod",
			EnvVars: []string{"NAMESPACE"},
			Value:   "default",
		},
		&cli.StringFlag{
			Name:    "path",
			Usage:   "the path to the token file",
			Value:   "/var/run/secrets/satokens/token",
			EnvVars: []string{"PATH"},
		},
		&cli.Int64Flag{
			Name:    "expiration",
			Usage:   "token expiration in seconds",
			Value:   7200,
			EnvVars: []string{"EXPIRATION"},
			Aliases: []string{"exp"},
		},
		&cli.StringFlag{
			Name:    "audience",
			Value:   "sts.amazonaws.com",
			EnvVars: []string{"AUDIENCE"},
			Aliases: []string{"aud"},
		},
		&cli.StringFlag{
			Name:    "image",
			Usage:   "image",
			EnvVars: []string{"IMAGE"},
			Value:   fmt.Sprintf("ghcr.io/ekristen/satokens:%s", common.AppVersion.Summary),
		},
		&cli.StringFlag{
			Name:    "service-account-name",
			Usage:   "the name of the service account",
			EnvVars: []string{"SERVICE_ACCOUNT"},
			Value:   "default",
		},
		&cli.BoolFlag{
			Name:    "create-service-account",
			Usage:   "create service account if it doesn't exist",
			EnvVars: []string{"CREATE_SERVICE_ACCOUNT", "CREATE_SA"},
			Aliases: []string{"create", "c"},
		},
	}

	cliCmd := &cli.Command{
		Name:  "deploy",
		Usage: "deploy satokens pod to the cluster",
		Description: `The deploy command adds a pod to the cluster by default in the default namespace, and attaches
the default service account to the pod. For more advanced use you can provide the service account name you want to use
or tell the deploy command to create the service account for you.`,
		Action: Execute,
		Flags:  append(flags, global.Flags()...),
		Before: global.Before,
	}

	common.RegisterCommand(cliCmd)
}
