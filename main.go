package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/pflag"
	"github.com/stack-go/kubectl-netshoot/pkg/utils"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func main() {

	var namespace string
	var targets []string
	var kubeconfig *string

	pflag.StringVarP(&namespace, "namespace", "n", "", "namespaced to be used")
	pflag.StringSliceVarP(&targets, "target", "t", []string{""}, "inform the targets, ex: --target www.google.com:443 --target gmail.com:443 -t www.google.com")

	if home := homedir.HomeDir(); home != "" {
		kubeconfig = pflag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "kubeconfig absolute path")
	} else {
		kubeconfig = pflag.String("kubeconfig", "", "kubeconfig absolute path")
	}
	pflag.Parse()
	config, err := clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	configOverrides := &clientcmd.ConfigOverrides{}
	kubeConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, configOverrides)

	ns, _, err := kubeConfig.Namespace()
	if err == nil && namespace == "" {
		namespace = ns
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	r, _ := utils.GenerateRandomString(12)

	gvr := schema.GroupVersionResource{Group: "platform.io", Version: "v1alpha1", Resource: "netshoots"}

	ObjTargetList := []map[string]interface{}{}
	for _, target := range targets {
		parts := strings.SplitN(target, ":", 2)

		host := parts[0]
		port := 80
		if len(parts) > 1 {
			port, _ = strconv.Atoi(parts[1])
		}

		objTarget := map[string]interface{}{
			"host": host,
			"port": port,
		}
		ObjTargetList = append(ObjTargetList, objTarget)
	}

	netshoot := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "platform.io/v1alpha1",
			"kind":       "NetShoot",
			"metadata": map[string]interface{}{
				"name": r,
			},
			"spec": map[string]interface{}{
				"targets": ObjTargetList,
			},
		},
	}

	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Create(context.TODO(), netshoot, metav1.CreateOptions{})
	if err != nil {
		panic(err.Error())
	}

	podName := fmt.Sprintf("netshoot-%s", netshoot.GetName())
	for {
		retries := 0
		if err := utils.WaitForPodReady(podName, namespace, config); err == nil || retries > 10 {
			break
		}
		time.Sleep(time.Second * 1)
		retries++
	}
	_ = utils.GetLogs(context.TODO(), podName, namespace, config)
}
