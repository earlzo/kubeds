package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"k8s.io/client-go/rest"
	"github.com/spf13/viper"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/kubernetes"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/sanity-io/litter"
	"encoding/json"
	"io/ioutil"
	"os"
)

// testCmd represents the test command
var testCmd = &cobra.Command{
	Use:   "test",
	Short: "A brief description of your command",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		var (
			kubeConfig *rest.Config
			err        error
		)
		kubeConfigPath := viper.GetString("KubeConfigPath")
		fmt.Printf("using out cluster config: %s", kubeConfigPath)
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigPath)
		if err != nil {
			panic(err.Error())
		}

		clientset, err := kubernetes.NewForConfig(kubeConfig)
		if err != nil {
			logrus.Fatalln(err)
		}
		ns := "douya"
		services, err := clientset.CoreV1().Services(ns).List(metav1.ListOptions{})
		if err != nil {
			logrus.Warnln(err)
		}
		data, err := json.Marshal(services)
		if err != nil {
			logrus.Warnln(err)
		}
		err = ioutil.WriteFile("services.json", data, os.ModePerm)
		if err != nil {
			logrus.Warnln(err)
		}
		litter.Dump(services)

		pods, err := clientset.CoreV1().Pods(ns).List(metav1.ListOptions{})
		if err != nil {
			logrus.Warnln(err)
		}
		data, err = json.Marshal(pods)
		if err != nil {
			logrus.Warnln(err)
		}
		err = ioutil.WriteFile("pods.json", data, os.ModePerm)
		if err != nil {
			logrus.Warnln(err)
		}
		litter.Dump(pods)

		endpoints, err := clientset.CoreV1().Endpoints(ns).List(metav1.ListOptions{})
		if err != nil {
			logrus.Warnln(err)
		}
		data, err = json.Marshal(endpoints)
		if err != nil {
			logrus.Warnln(err)
		}
		err = ioutil.WriteFile("endpoints.json", data, os.ModePerm)
		if err != nil {
			logrus.Warnln(err)
		}
		litter.Dump(endpoints)
	},
}

func init() {
	rootCmd.AddCommand(testCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// testCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// testCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
