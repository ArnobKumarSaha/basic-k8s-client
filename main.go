package main

import (
	"context"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"
	kmapi "kmodules.xyz/client-go/api/v1"
	cu "kmodules.xyz/client-go/client"
	"kmodules.xyz/client-go/conditions"
	kubedb "kubedb.dev/apimachinery/apis/kubedb/v1alpha2"
	kubedbscheme "kubedb.dev/apimachinery/client/clientset/versioned/scheme"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scm = runtime.NewScheme()
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scm))
	utilruntime.Must(kubedbscheme.AddToScheme(scm))
}

func main() {
	config := getRESTConfig()
	_ = kubernetesClient(config)
	_ = kubeBuilderClient(config)
	_ = testCreateOrPatch(config)
	_ = testPatchStatus(config)
}

func getRESTConfig() *rest.Config {
	//var kubeconfig string
	//if home := homedir.HomeDir(); home != "" {
	//	flag.StringVar(&kubeconfig, "kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	//} else {
	//	flag.StringVar(&kubeconfig, "kubeconfig", "", "absolute path to the kubeconfig file")
	//}
	//flag.Parse()

	home := homedir.HomeDir()
	config, err := clientcmd.BuildConfigFromFlags("", filepath.Join(home, ".kube", "config"))
	if err != nil {
		panic(err.Error())
	}
	return config
}

func kubernetesClient(config *rest.Config) error {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return err
	}
	fmt.Printf("There are %d pods in the cluster\n", len(pods.Items))
	return nil
}

func kubeBuilderClient(config *rest.Config) error {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		return err
	}

	var depList appsv1.DeploymentList
	mp := make(map[string]string)
	mp["metadata.name"] = "helm-controller"
	err = kc.List(context.Background(), &depList, client.MatchingFieldsSelector{Selector: fields.Set(mp).AsSelector()})
	if err != nil {
		return err
	}

	var mongodbs kubedb.MongoDBList
	err = kc.List(context.TODO(), &mongodbs)
	for _, m := range mongodbs.Items {
		klog.Infof("%s/%s , ", m.Namespace, m.Name)
	}
	return nil
}

func testCreateOrPatch(config *rest.Config) error {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		return err
	}

	mg := &kubedb.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mg",
			Namespace: "demo",
		},
	}
	klog.Infof("Trying to create mongodb")
	v, err := cu.CreateOrPatch(context.TODO(), kc, mg, func(obj client.Object, createOp bool) client.Object {
		db := obj.(*kubedb.MongoDB)
		db.Spec.Version = "5.0.3"
		db.Spec.Replicas = pointer.Int32(1)
		return db
	})
	if err != nil {
		klog.Infof("%s \n", err.Error())
		return err
	}
	klog.Infof("%+v, %+v", v, mg)
	return nil
}

func testPatchStatus(config *rest.Config) error {
	kc, err := client.New(config, client.Options{
		Scheme: scm,
		Mapper: nil,
	})
	if err != nil {
		return err
	}

	mg := &kubedb.MongoDB{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "mg",
			Namespace: "demo",
		},
	}
	klog.Infof("Trying to patching mongodb status")
	v, err := cu.PatchStatus(context.TODO(), kc, mg, func(obj client.Object) client.Object {
		db := obj.(*kubedb.MongoDB)
		db.Status.Conditions = conditions.SetCondition(db.Status.Conditions, kmapi.Condition{
			Type:    "aaaa",
			Status:  "aaa",
			Reason:  "aa",
			Message: "a",
		})
		return db
	})
	if err != nil {
		klog.Infof("%s \n", err.Error())
		return err
	}
	klog.Infof("%+v, %+v =======  %+v \n", v, mg.Spec, mg.Status)
	return nil
}
