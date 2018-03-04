package leizu

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Hasher is a single cache key hash.
type Hasher struct {
}

// ID function that always returns the same value.
func (h Hasher) ID(node *core.Node) string {
	return node.Id
}

var (
	once sync.Once
	app  *Application
)

func InitApplication(config *viper.Viper) *Application {
	once.Do(func() {
		if config == nil {
			config = viper.GetViper()
		}
		app = &Application{
			logger: logrus.New(),
			ctx:    context.Background(),
		}

		// init snapCache
		app.cache = cache.NewSnapshotCache(true, Hasher{}, app.logger)
		app.server = xds.NewServer(app.cache, nil)
		app.grpcServer = grpc.NewServer()

		v2.RegisterEndpointDiscoveryServiceServer(app.grpcServer, app.server)

		// init client
		var (
			kubeConfig *rest.Config
			err        error
		)
		if viper.GetBool("OutCluster") {
			kubeConfigPath := viper.GetString("KubeConfigPath")
			fmt.Printf("using out cluster config: %s", kubeConfigPath)
			kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
			if err != nil {
				panic(err.Error())
			}
		} else {
			kubeConfig, err = rest.InClusterConfig()
			if err != nil {
				panic(err.Error())
			}
		}
		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			app.logger.Fatalln(err)
		}
		app.KubeClient = clientset
	})
	return app
}

// Application is program entry
type Application struct {
	logger     *logrus.Logger
	config     *viper.Viper
	ctx        context.Context
	cache      cache.SnapshotCache
	server     xds.Server
	grpcServer *grpc.Server
	KubeClient *kubernetes.Clientset
}

func (a *Application) Serve() {
	port := a.config.Unmarshal
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		a.logger.WithError(err).Fatalln("failed to listen")
	}
	go func() {
		if err = a.grpcServer.Serve(lis); err != nil {
			a.logger.Fatalln(err)
		}
	}()
	go func() {
		podWatcher, err := a.KubeClient.CoreV1().Pods("").Watch(metav1.ListOptions{})
		if err != nil {
			a.logger.WithError(err).Fatalln()
		}
		for event := range podWatcher.ResultChan() {
			a.logger.WithField("event", event.Type).Infof("pod event received")
			// pod := event.Object.(*v1.Pod)
			// endpoint := endpoint.Endpoint{
			// 	Address: &core.Address{
			// 		Address: &core.Address_SocketAddress{
			// 			SocketAddress: &core.SocketAddress{
			// 				Protocol: core.TCP,
			// 				Address:  pod.Status.PodIP,
			// 				PortSpecifier: &core.SocketAddress_PortValue{
			// 					PortValue: 5000,
			// 				},
			// 			},
			// 		},
			// 	},
			// }
		}
	}()
	<-a.ctx.Done()
	a.grpcServer.GracefulStop()
}
