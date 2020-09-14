package main

import (
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	iotv1 "github.com/ctison/iot/operator/api/v1"
	"github.com/ctison/iot/operator/controllers"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(iotv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	// Setup MQTT flags.
	var mqttURL, clientCert, clientKey, serverCert string
	flag.StringVar(&mqttURL, "url", "tls://iot.fr-par.scw.cloud:8883", "URL of the mqtt server.")
	flag.StringVar(&clientCert, "client-cert", "crt.pem", "Path to client certificate")
	flag.StringVar(&clientKey, "client-key", "key.pem", "Path to client key")
	flag.StringVar(&serverCert, "server-cert", "ca.pem", "Path to server certificate")

	flag.Parse()

	// Instantiate mqtt client
	if len(os.Args) != 2 {
		fmt.Println(`Usage: fridge-operator CLIENT_ID

Available flags:`)
		flag.PrintDefaults()
		return
	}

	ctrl.SetLogger(zap.New(zap.UseDevMode(true)))

	// mqttServerURL, err := url.Parse(mqttURL)
	// if err != nil {
	// 	ctrl.Log.Error(err, "failed to parse mqtt server URL")
	// 	return
	// }

	mqttTLSConfig, err := newTLSConfig(clientCert, clientKey, serverCert)
	if err != nil {
		ctrl.Log.Error(err, "failed to instantiate tls config")
		return
	}

	clientOptions := mqtt.NewClientOptions()
	clientOptions.AutoReconnect = true
	clientOptions.SetClientID(os.Args[1])
	clientOptions.SetTLSConfig(mqttTLSConfig)
	clientOptions.AddBroker(mqttURL)

	mqttClient := mqtt.NewClient(clientOptions)

	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		ctrl.Log.Error(err, "failed to connect to MQTT server")
		return
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "5d1c80a8.ctison.dev",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.FridgeReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Fridge"),
		Scheme: mgr.GetScheme(),
		MQTT:   mqttClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Fridge")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// Create a new TLS config from PEM paths.
func newTLSConfig(clientCert, clientKey, serverCert string) (*tls.Config, error) {
	certpool := x509.NewCertPool()
	pemCerts, err := ioutil.ReadFile(serverCert)
	if err != nil {
		return nil, err
	}
	certpool.AppendCertsFromPEM(pemCerts)

	// Import client certificate/key pair
	var certs []tls.Certificate
	cert, err := tls.LoadX509KeyPair(clientCert, clientKey)
	if err != nil {
		return nil, err
	}
	certs = append(certs, cert)

	// Create tls.Config with desired tls properties
	return &tls.Config{
		// RootCAs = certs used to verify server cert.
		RootCAs: certpool,
		// ClientAuth = whether to request cert from server.
		// Since the server is set up for SSL, this happens
		// anyways.
		ClientAuth: tls.NoClientCert,
		// ClientCAs = certs used to validate client cert.
		ClientCAs: nil,
		// InsecureSkipVerify = verify that cert contents
		// match server. IP matches what is in cert etc.
		InsecureSkipVerify: false,
		// Certificates = list of certs client sends to server.
		Certificates: certs,
	}, nil
}
