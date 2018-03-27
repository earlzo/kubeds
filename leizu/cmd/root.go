package cmd

import (
	"fmt"
	"os"

	"github.com/shanbay/leizu"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/util/homedir"
)

var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "leizu",
	Short: "A envoy api implementation for kubernetes",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {
		app := leizu.InitApplication(viper.GetViper())
		app.Serve()
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().BoolP("out-cluster", "o", viper.GetBool("outCluster"), "using out cluster kubeconfig")
	viper.BindPFlag("outCluster", rootCmd.PersistentFlags().Lookup("out-cluster"))

	rootCmd.PersistentFlags().StringP("kubeconfig", "k", viper.GetString("kubeConfigPath"), "absolute path to the kubeconfig file")
	viper.BindPFlag("kubeConfigPath", rootCmd.PersistentFlags().Lookup("kubeconfig"))

	rootCmd.PersistentFlags().StringP("namespace", "n", viper.GetString("nameSpace"), "namespace to listen")
	viper.BindPFlag("nameSpace", rootCmd.PersistentFlags().Lookup("namespace"))

	rootCmd.PersistentFlags().StringP("address", "a", viper.GetString("grpcServerAddress"), "address for grpc server")
	viper.BindPFlag("grpcServerAddress", rootCmd.Flags().Lookup("address"))

	// currently we do not support ADS
	//rootCmd.PersistentFlags().Bool("ads", viper.GetBool("ads"), "Use ADS instead of separate xDS services")
	//viper.BindPFlag("ads", rootCmd.Flags().Lookup("ads"))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	leizu.LoadDefaultSettingsFor(viper.GetViper())

	// Find home directory.
	home := homedir.HomeDir()

	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Search config in home directory with name ".github.com\shanbay\leizu" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigName(".leizu.yml")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
