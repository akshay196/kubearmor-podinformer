package main

import (
	"context"
	"errors"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/workqueue"
)

var (
	// Chagne your node name here
	nodeName = "kubearmor-podinformer-control-plane"
)

type PodInformer struct {
	indexer  cache.Indexer
	queue    workqueue.RateLimitingInterface
	informer cache.Controller

	stop           chan struct{}
	updatedPodChan chan *v1.Pod
	deletedPodChan chan string
	wg             sync.WaitGroup
}

func main() {

	// Part 1
	config, err := clientcmd.BuildConfigFromFlags("", "/home/akshay/.kube/config")
	if err != nil {
		log.Fatalf("getting kubernetes config: %s", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatalf("getting Kubernetes client: %w", err)
	}

	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", nodeName).String()
	pods, err := clientset.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{
		FieldSelector: fieldSelector,
	})
	if err != nil {
		log.Fatalf("cannot fetch pods: %s", err)
	}

	containerNames := []string{}
	for _, pod := range pods.Items {

		for _, c := range pod.Spec.Containers {
			containerNames = append(containerNames, c.Name)
		}
	}

	log.Infof("Initial containers: %v", containerNames)

	// Part 2
	podListWatcher := cache.NewListWatchFromClient(clientset.CoreV1().RESTClient(), "pods", "", fields.OneTermEqualSelector("spec.nodeName", nodeName))
	queue := workqueue.NewRateLimitingQueue(workqueue.DefaultControllerRateLimiter())
	indexer, informer := cache.NewIndexerInformer(podListWatcher, &v1.Pod{}, 0, cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
		UpdateFunc: func(old interface{}, new interface{}) {
			key, err := cache.MetaNamespaceKeyFunc(new)
			if err == nil {
				queue.Add(key)
			}
		},
		DeleteFunc: func(obj interface{}) {
			// IndexerInformer uses a delta queue, therefore for deletes we have to use this
			// key function.
			key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
			if err == nil {
				queue.Add(key)
			}
		},
	}, cache.Indexers{})

	p := &PodInformer{
		indexer:        indexer,
		queue:          queue,
		informer:       informer,
		stop:           make(chan struct{}),
		updatedPodChan: make(chan *v1.Pod),
		deletedPodChan: make(chan string),
	}

	go p.Run(1, p.stop)

	go func() {
		for {
			select {
			case key, ok := <-p.deletedPodChan:
				if !ok {
					continue
				}
				log.Infof("Pod with key %s delete", key)
			case pod, ok := <-p.updatedPodChan:
				if !ok {
					continue
				}
				for _, s := range pod.Status.ContainerStatuses {
					log.Infof("New containers: %s", s.Name)
				}
			}
		}
	}()

	for {
	}
}

func (p *PodInformer) Run(threadiness int, stopCh chan struct{}) {
	defer runtime.HandleCrash()

	// Let the workers stop when we are done
	defer p.queue.ShutDown()
	log.Info("Starting Pod controller")

	go p.informer.Run(stopCh)

	// Wait for all involved caches to be synced, before processing items from the queue is started
	if !cache.WaitForCacheSync(stopCh, p.informer.HasSynced) {
		runtime.HandleError(errors.New("timed out waiting for caches to sync"))
		return
	}

	for i := 0; i < threadiness; i++ {
		p.wg.Add(1)
		go wait.Until(p.runWorker, time.Second, stopCh)
	}

	<-stopCh
	log.Info("Stopping Pod controller")
}

func (p *PodInformer) runWorker() {
	defer p.wg.Done()

	for p.processNextItem() {
	}
}

func (p *PodInformer) processNextItem() bool {
	// Wait until there is a new item in the working queue
	key, quit := p.queue.Get()
	if quit {
		return false
	}

	defer p.queue.Done(key)

	err := p.notifyChans(key.(string))
	if err != nil {
		log.Error(err)
	}
	return true
}

func (p *PodInformer) notifyChans(key string) error {
	obj, exists, err := p.indexer.GetByKey(key)
	if err != nil {
		log.Errorf("Fetching object with key %s from store failed with %v", key, err)
		return err
	}
	defer p.queue.Forget(key)

	if !exists {
		p.deletedPodChan <- key
		return nil
	}

	p.updatedPodChan <- obj.(*v1.Pod)
	return nil
}
