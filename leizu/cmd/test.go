package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"

	"github.com/envoyproxy/go-control-plane/pkg/test/resource"
	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/glog"
	"github.com/shanbay/leizu"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	xdsPort       uint
	ads           bool
	bootstrapFile string
	envoyBinary   string
)

var testCmd = &cobra.Command{
	Use:   "test",
	Short: "test leizu",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		kubeClient, err := leizu.SimpleKubeClient(viper.GetViper())
		if err != nil {
			logrus.WithError(err).Fatalln("get kube client failed")
		}
		ns := viper.GetString("namespace")
		fmt.Print(kubeClient, ns)

		app := leizu.InitApplication(viper.GetViper())
		go app.Serve()

		// write bootstrap file
		bootstrap := resource.MakeBootstrap(ads, uint32(xdsPort), 19000)
		buf := &bytes.Buffer{}
		if err := (&jsonpb.Marshaler{OrigName: true}).Marshal(buf, bootstrap); err != nil {
			glog.Fatal(err)
		}
		if err := ioutil.WriteFile(bootstrapFile, buf.Bytes(), 0644); err != nil {
			glog.Fatal(err)
		}

		// start envoy
		envoy := exec.Command("envoy", "-c", bootstrapFile, "--drain-time-s", "1")
		envoy.Stdout = os.Stdout
		envoy.Stderr = os.Stderr
		envoy.Start()
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	testCmd.Flags().BoolVar(&ads, "ads", true, "Use ADS instead of separate xDS services")
	testCmd.Flags().StringVar(&bootstrapFile, "bootstrap", "bootstrap.json", "Bootstrap file name")
	testCmd.Flags().StringVar(&envoyBinary, "envoy", "envoy", "Envoy binary file")
}
