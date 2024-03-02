package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"time"

	corev1 "k8s.io/api/core/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func GenerateRandomString(length int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	bytes := make([]byte, length)

	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	for i, b := range bytes {
		bytes[i] = letters[b%byte(len(letters))]
	}

	return string(bytes), nil
}

func GetLogs(ctx context.Context, podName, namespace string, c *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	podLogOpts := corev1.PodLogOptions{
		Follow: true,
	}

	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &podLogOpts)

	logStream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("error opening log stream for pod %s in namespace %s: %v", podName, namespace, err)
	}
	defer logStream.Close()

	_, err = io.Copy(os.Stdout, logStream)
	if err != nil && err != io.EOF {
		return fmt.Errorf("error streaming logs for pod %s in namespace %s: %v", podName, namespace, err)
	}

	return nil
}
func WaitForPodReady(podName, namespace string, c *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %v", err)
	}

	return wait.PollImmediate(time.Second, 5*time.Minute, func() (bool, error) {

		pod, err := clientset.CoreV1().Pods(namespace).Get(context.TODO(), podName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		fmt.Printf("creating pod %s on namespace %s status:%s \n", pod.Name, pod.Namespace, pod.Status.Phase)
		for _, cond := range pod.Status.Conditions {

			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return true, nil
			}
		}
		return false, nil
	})
}
