package cmd

import (
	"github.com/shanbay/kubeds"
	"github.com/shanbay/kubeds/test/resource"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	k8sApiMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	bootstrapFile string
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test kubeds",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		app := leizu.InitApplication(viper.GetViper())
		// write bootstrap file
		ns := viper.GetString("namespace")

		bootstrap := resource.MakeBootstrap(uint32(app.Config.GetInt("xdsPort")), 19000)
		services, err := app.KubeClient.CoreV1().Services(ns).List(k8sApiMetaV1.ListOptions{})
		if err != nil {
			logrus.Warnln(err)
		}
		for _, svc := range services.Items {
			clusterName := svc.Name + "." + svc.Namespace
			cluster := resource.MakeCluster(app.Config.GetBool("ads"), clusterName)
			bootstrap.StaticResources.Clusters = append(bootstrap.StaticResources.Clusters, *cluster)
		}
		logrus.WithField("path", bootstrapFile).Infoln("please start envoy with bootstrap file")
		app.Serve()
	},
}

func init() {
	rootCmd.AddCommand(testCmd)
}
