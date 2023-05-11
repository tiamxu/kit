package kube

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/urfave/cli/v2"
	appsv1 "k8s.io/api/apps/v1"
	apiv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

var kubeconf string

func HomeDir() string {
	return os.Getenv("HOME")
}

type kubeClient struct {
	*exec.Cmd
}

// tools commands
func kubectl(args ...string) *kubeClient {
	return &kubeClient{
		&exec.Cmd{
			Path: "/usr/local/bin/kubectl",
			Args: append([]string{"kubectl"}, args...),
		},
	}
}

// exec command
func RunRestart(c *cli.Context) error {
	return restartAction(c)
}

func RunGetDeployment(c *cli.Context) error {
	return getDeploymentAction(c)
}
func CreateDeployment(c *cli.Context) error {
	return createDeploymentAction(c)
}

// command action
func restartAction(c *cli.Context) error {
	kubectl("get", "deploy")
	return nil
}
func getDeploymentAction(c *cli.Context) error {
	var (
		stderr bytes.Buffer
		stdout bytes.Buffer
		cmd    *exec.Cmd
	)
	// namespace = c.String("namespace")
	// name = c.String("name")
	// kubeconf = os.Getenv("HOME") + "/.kube/config-" + namespace
	fmt.Printf("namespace:%s,name:%s\n", namespace, name)
	if name == "" {
		cmd = exec.Command("kubectl", "get", "deploy", "-n", namespace, "--kubeconfig", kubeconf)

	} else {
		cmd = exec.Command("kubectl", "get", "deploy", name, "-n", namespace, "--kubeconfig", kubeconf)
	}
	cmd.Stderr = &stderr
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		fmt.Println(stderr.String())
		return err
	}
	fmt.Println(stdout.String())

	return nil
}

func createDeploymentAction(c *cli.Context) error {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconf)
	if err != nil {
		fmt.Println(err)
		return err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err)
	}
	deploymentsClient := clientset.AppsV1().Deployments(namespace)

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name: "demo-deployment",
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: int32Ptr(2),
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": "demo",
				},
			},
			Template: apiv1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "demo",
					},
				},
				Spec: apiv1.PodSpec{
					Containers: []apiv1.Container{
						{
							Name:  "web",
							Image: "nginx:1.12",
							Ports: []apiv1.ContainerPort{
								{
									Name:          "http",
									Protocol:      apiv1.ProtocolTCP,
									ContainerPort: 80,
								},
							},
						},
					},
				},
			},
		},
	}

	// Create Deployment
	fmt.Println("Creating deployment...")
	result, err := deploymentsClient.Create(context.TODO(), deployment, metav1.CreateOptions{})
	if err != nil {
		panic(err)
	}
	fmt.Printf("Created deployment %q.\n", result.GetObjectMeta().GetName())
	return nil
}
