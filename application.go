package leizu

import (
	"context"
	"fmt"
	"net"
	"sync"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_api_v2_core2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/pkg/cache"
	xds "github.com/envoyproxy/go-control-plane/pkg/server"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Hasher is a single cache key hash.
type Hasher struct {
}

// ID function that always returns the same value.
func (h Hasher) ID(node *envoy_api_v2_core2.Node) string {
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
			config: config,
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
		if viper.GetBool("outCluster") {
			kubeConfigPath := viper.GetString("kubeConfigPath")
			fmt.Printf("using out cluster config: %s", kubeConfigPath)
			kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
			if err != nil {
				app.logger.WithError(err).Fatalln("load config failed")
			}
		} else {
			kubeConfig, err = rest.InClusterConfig()
			if err != nil {
				app.logger.WithError(err).Fatalln("load config failed")
			}
		}
		app.logger.WithFields(logrus.Fields{
			"host": kubeConfig.Host,
			"username": kubeConfig.Username,
			"userAgent": kubeConfig.UserAgent,
		}).Infoln("k8s config was loaded")
		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			app.logger.WithError(err).Fatalln("make k8s client failed")
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
	go func() {
		addr := a.config.GetString("grpcServerAddress")
		lis, err := net.Listen("tcp", addr)
		if err != nil {
			a.logger.WithError(err).Fatalln("failed to listen")
		}
		a.logger.WithField("grpcServerAddress", addr).Infoln("start listening grpc")
		if err = a.grpcServer.Serve(lis); err != nil {
			a.logger.WithError(err).Fatalln("serve grpc server failed")
		}
	}()

	go func() {
		// watch k8s cluster endpoints, and set set snapshot after changes
		nameSpace := a.config.GetString("nameSpace")
		endWatcher, err := a.KubeClient.CoreV1().Endpoints(nameSpace).Watch(metav1.ListOptions{})
		if err != nil {
			a.logger.WithError(err).Fatalln("watch endpoints changes failed")
		}
		a.logger.Infoln("start watching Endpoints events")
		for event := range endWatcher.ResultChan() {
			a.logger.WithField("event", event.Type).Infoln("endpoints event received")
			var healthStatus envoy_api_v2_core2.HealthStatus
			switch event.Type {
			case watch.Added, watch.Modified:
				healthStatus = envoy_api_v2_core2.HealthStatus_HEALTHY
			case watch.Deleted, watch.Error:
				healthStatus = envoy_api_v2_core2.HealthStatus_UNHEALTHY
			default:
				healthStatus = envoy_api_v2_core2.HealthStatus_UNKNOWN
			}
			endpoints := event.Object.(*v1.Endpoints)
			envoyEndpoints := a.Endpoints2ClusterLoadAssignment(endpoints, healthStatus)
			snapShot := cache.NewSnapshot(
				endpoints.ResourceVersion,
				[]cache.Resource{envoyEndpoints},
				[]cache.Resource{},
				[]cache.Resource{},
				[]cache.Resource{},
			)
			// TODO: dispatch Node
			if err := a.cache.SetSnapshot("", snapShot); err != nil {
				a.logger.WithError(err).Errorln("SetSnapshot failed ")
			}
			a.logger.WithField("version", endpoints.ResourceVersion).Infoln("set new snapshot")
		}
	}()
	<-a.ctx.Done()
	a.grpcServer.GracefulStop()
}

func (a *Application) Endpoints2ClusterLoadAssignment(endpoints *v1.Endpoints, healthStatus envoy_api_v2_core2.HealthStatus) *v2.ClusterLoadAssignment {
	clusterName := endpoints.ObjectMeta.Name + "." + endpoints.ObjectMeta.Namespace

	lbEndpoints := make([]endpoint.LbEndpoint, 0)
	for _, subset := range endpoints.Subsets {
		for _, port := range subset.Ports {
			for _, address := range subset.Addresses {
				var protocol envoy_api_v2_core2.SocketAddress_Protocol
				switch port.Protocol {
				case v1.ProtocolTCP:
					protocol = envoy_api_v2_core2.TCP
				case v1.ProtocolUDP:
					protocol = envoy_api_v2_core2.UDP
				}
				lbEndpoints = append(lbEndpoints, endpoint.LbEndpoint{
					HealthStatus: healthStatus,
					Endpoint: &endpoint.Endpoint{
						Address: &envoy_api_v2_core2.Address{
							Address: &envoy_api_v2_core2.Address_SocketAddress{
								SocketAddress: &envoy_api_v2_core2.SocketAddress{
									Protocol: protocol,
									Address:  address.IP,
									PortSpecifier: &envoy_api_v2_core2.SocketAddress_PortValue{
										PortValue: uint32(port.Port),
									},
								},
							},
						},
					},
				})
			}
		}
	}

	a.logger.WithFields(logrus.Fields{
		"clusterName":      clusterName,
		"healthStatus":     healthStatus,
		"lbEndPointsCount": len(lbEndpoints),
	}).Infoln("converted k8s endpoints to envoy cluster load assignment")
	return &v2.ClusterLoadAssignment{
		ClusterName: clusterName,
		Endpoints: []endpoint.LocalityLbEndpoints{{
			LbEndpoints: lbEndpoints,
		}},
	}
}
