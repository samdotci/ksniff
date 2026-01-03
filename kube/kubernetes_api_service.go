package kube

import (
	"context"
	"io"
	"time"

	"ksniff/utils"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type KubernetesApiService interface {
	ExecuteCommand(podName string, containerName string, command []string, stdOut io.Writer) (int, error)

	CreateEphemeralContainer(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error
}

type KubernetesApiServiceImpl struct {
	clientset       *kubernetes.Clientset
	restConfig      *rest.Config
	targetNamespace string
}

func NewKubernetesApiService(clientset *kubernetes.Clientset,
	restConfig *rest.Config, targetNamespace string) KubernetesApiService {

	return &KubernetesApiServiceImpl{clientset: clientset,
		restConfig:      restConfig,
		targetNamespace: targetNamespace}
}

func (k *KubernetesApiServiceImpl) ExecuteCommand(podName string, containerName string, command []string, stdOut io.Writer) (int, error) {

	log.Infof("executing command: '%s' on container: '%s', pod: '%s', namespace: '%s'", command, containerName, podName, k.targetNamespace)
	stdErr := new(Writer)

	executeTcpdumpRequest := ExecCommandRequest{
		KubeRequest: KubeRequest{
			Clientset:  k.clientset,
			RestConfig: k.restConfig,
			Namespace:  k.targetNamespace,
			Pod:        podName,
			Container:  containerName,
		},
		Command: command,
		StdErr:  stdErr,
		StdOut:  stdOut,
	}

	exitCode, err := PodExecuteCommand(executeTcpdumpRequest)
	if err != nil {
		log.WithError(err).Errorf("failed executing command: '%s', exitCode: '%d', stdErr: '%s'",
			command, exitCode, stdErr.Output)

		return exitCode, err
	}

	log.Infof("command: '%s' executing successfully exitCode: '%d', stdErr :'%s'", command, exitCode, stdErr.Output)

	return exitCode, err
}

func (k *KubernetesApiServiceImpl) CreateEphemeralContainer(podName string, targetContainer string, ephemeralContainerName string, tcpdumpImage string, timeout time.Duration) error {
	log.Debugf("creating ephemeral container '%s' in pod '%s'", ephemeralContainerName, podName)

	// Get the current pod
	pod, err := k.clientset.CoreV1().Pods(k.targetNamespace).Get(context.TODO(), podName, v1.GetOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to get pod '%s'", podName)
	}

	// Check if ephemeral container already exists
	for _, ec := range pod.Spec.EphemeralContainers {
		if ec.Name == ephemeralContainerName {
			log.Infof("ephemeral container '%s' already exists in pod '%s'", ephemeralContainerName, podName)
			return nil
		}
	}

	// Create the ephemeral container spec
	// NET_RAW capability is required for tcpdump to capture packets
	ephemeralContainer := corev1.EphemeralContainer{
		EphemeralContainerCommon: corev1.EphemeralContainerCommon{
			Name:            ephemeralContainerName,
			Image:           tcpdumpImage,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"sh", "-c", "sleep 10000000"},
			TTY:             true,
			Stdin:           true,
			SecurityContext: &corev1.SecurityContext{
				Capabilities: &corev1.Capabilities{
					Add: []corev1.Capability{"NET_RAW", "NET_ADMIN"},
				},
			},
		},
		TargetContainerName: targetContainer,
	}

	// Add the ephemeral container to the pod
	pod.Spec.EphemeralContainers = append(pod.Spec.EphemeralContainers, ephemeralContainer)

	_, err = k.clientset.CoreV1().Pods(k.targetNamespace).UpdateEphemeralContainers(
		context.TODO(),
		podName,
		pod,
		v1.UpdateOptions{},
	)
	if err != nil {
		return errors.Wrapf(err, "failed to create ephemeral container '%s' in pod '%s'", ephemeralContainerName, podName)
	}

	log.Infof("ephemeral container '%s' created in pod '%s', waiting for it to be ready", ephemeralContainerName, podName)

	// Wait for the ephemeral container to be running
	verifyContainerState := func() bool {
		podStatus, err := k.clientset.CoreV1().Pods(k.targetNamespace).Get(context.TODO(), podName, v1.GetOptions{})
		if err != nil {
			return false
		}

		for _, cs := range podStatus.Status.EphemeralContainerStatuses {
			if cs.Name == ephemeralContainerName {
				if cs.State.Running != nil {
					return true
				}
			}
		}
		return false
	}

	if !utils.RunWhileFalse(verifyContainerState, timeout, 1*time.Second) {
		return errors.Errorf("ephemeral container '%s' did not become ready within timeout (%s)", ephemeralContainerName, timeout)
	}

	log.Infof("ephemeral container '%s' is now running", ephemeralContainerName)

	return nil
}
