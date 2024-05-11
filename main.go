package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"

	"github.com/yankeguo/rg"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func main() {
	var err error
	defer func() {
		if err == nil {
			return
		}
		log.Println("error:", err.Error())
		os.Exit(1)
	}()
	defer rg.Guard(&err)

	config := rg.Must(rest.InClusterConfig())

	client := rg.Must(kubernetes.NewForConfig(config))

	var (
		optSource    string
		optNamespace string
		optName      string
	)

	flag.StringVar(&optSource, "source", "", "source secret, in format of [NAMESPACE]/[NAME]")
	flag.StringVar(&optNamespace, "namespace", "", "target namespace, support regex")
	flag.StringVar(&optName, "name", "", "target secret name")
	flag.Parse()

	if optNamespace == "" {
		err = fmt.Errorf("namespace is required")
		return
	}

	regexpNamespace := regexp.MustCompile(optNamespace)

	var (
		sourceNamespace string
		sourceName      string
	)

	if splits := strings.Split(optSource, "/"); len(splits) == 2 {
		sourceNamespace = splits[0]
		sourceName = splits[1]
	} else {
		sourceNamespace = string(rg.Must(os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")))
		sourceName = optSource
	}

	if optName == "" {
		optName = sourceName
	}

	ctx := context.Background()

	secret := rg.Must(client.CoreV1().Secrets(sourceNamespace).Get(ctx, sourceName, metav1.GetOptions{}))

	namespaces := rg.Must(client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{}))

	for _, item := range namespaces.Items {
		if !regexpNamespace.MatchString(item.Name) {
			continue
		}
		rg.Must0(replicateSecret(ctx, client, secret, item.Name, optName))
	}
}

func replicateSecret(ctx context.Context, client *kubernetes.Clientset, secret *corev1.Secret, namespace, name string) (err error) {
	var existed *corev1.Secret

	if existed, err = client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{}); err != nil {
		if errors.IsNotFound(err) {
			log.Printf("creating secret: %s/%s", namespace, name)
			_, err = client.CoreV1().Secrets(namespace).Create(ctx, &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:        name,
					Labels:      secret.Labels,
					Annotations: secret.Annotations,
				},
				Type: secret.Type,
				Data: secret.Data,
			}, metav1.CreateOptions{})
		}
		return
	}

	for k, v := range secret.Data {
		if existed.Data == nil {
			goto doUpdate
		}
		if !bytes.Equal(v, existed.Data[k]) {
			goto doUpdate
		}
	}

	log.Printf("already up-to-date: %s/%s", namespace, name)

	return

doUpdate:

	existed.Data = secret.Data

	log.Printf("updating secret: %s/%s", namespace, name)

	_, err = client.CoreV1().Secrets(namespace).Update(ctx, existed, metav1.UpdateOptions{})

	return
}
