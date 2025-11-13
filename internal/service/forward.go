package service

import (
	"context"
	"fmt"
	"time"

	"net/http"

	"github.com/rs/zerolog"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"

	"github.com/codozor/fwkeeper/internal/config"
)

type Forwarder struct {
	locator Locator
	configuration config.PortForwardConfiguration

	client kubernetes.Interface

	transport http.RoundTripper

	upgrader spdy.Upgrader
}

type forwarderWriter struct {
	logger *zerolog.Logger
	level zerolog.Level
}

func (w *forwarderWriter) Write(buf []byte) (n int, err error) {
	n = len(buf)

	if n > 0 && buf[n-1] == '\n' {
        buf = buf[:n-1]
    }

	switch w.level {
    case zerolog.ErrorLevel:
        w.logger.Error().Msg(string(buf))
    case zerolog.WarnLevel:
        w.logger.Warn().Msg(string(buf))
    case zerolog.InfoLevel:
        w.logger.Info().Msg(string(buf))
    default:
        w.logger.Debug().Msg(string(buf))
    }
	return n, nil
}

func NewForwarder(locator Locator, configuration config.PortForwardConfiguration, client kubernetes.Interface, transport http.RoundTripper, upgrader spdy.Upgrader) (*Forwarder, error) {
	return &Forwarder{
		locator: locator,
		configuration: configuration,
		client: client,
		transport: transport,
		upgrader: upgrader,
	}, nil
}

func (f *Forwarder) Start(ctx context.Context) {
	log := zerolog.Ctx(ctx)

	log.Info().Msgf("START - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)

	for {
		if ctx.Err() != nil {
			break
		}

		podName, err := f.locator.locate(ctx)
		if err != nil{
			log.Error().Err(err).Msgf("ERROR - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
			f.delayRetry(ctx)
			continue
		}

		pod, err := f.client.CoreV1().Pods(f.configuration.Namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			log.Error().Err(err).Msgf("ERROR - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
			f.delayRetry(ctx)
			continue
		}

		if pod.Status.Phase != corev1.PodRunning {
			err := fmt.Errorf("pod not in running state")
			log.Error().Err(err).Msgf("ERROR - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
			f.delayRetry(ctx)
			continue
		}

		// Prepare URL 
		req := f.client.CoreV1().RESTClient().Post().
			Resource("pods").
			Namespace(f.configuration.Namespace).
        	Name(podName).
        	SubResource("portforward")


		// Create the dialer
    	dialer := spdy.NewDialer(f.upgrader, &http.Client{Transport: f.transport}, "POST", req.URL())

		// Prepare channel for stop/ready
    	stopCh := make(chan struct{}, 1)
    	readyCh := make(chan struct{})
    	errCh := make(chan error)

		outWriter := &forwarderWriter { logger: log, level: zerolog.InfoLevel }
		errWriter := &forwarderWriter { logger: log, level: zerolog.ErrorLevel }

		fw, err := portforward.New(dialer, f.configuration.Ports, stopCh, readyCh, outWriter, errWriter)
		if err != nil {
			log.Error().Err(err).Msgf("ERROR - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
			f.delayRetry(ctx)
			continue
		}

		// Stop the forwarder when context canceled
		go func() {
    		<-ctx.Done()
    		close(stopCh)
    	}()

		// Start forwards
    	go func() {
        	errCh <- fw.ForwardPorts()
    	}()

		select {
		case <- readyCh:
			log.Info().Msgf("READY - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
		case <- errCh:
			log.Error().Err(err).Msgf("ERROR - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
			f.delayRetry(ctx)
			continue
		}

		err = <- errCh
		
		log.Error().Err(err).Msgf("ERROR - Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
	}

	log.Info().Msgf("STOP Forwarder %s(%s %s) %v", f.configuration.Name, f.configuration.Namespace, f.configuration.Resource, f.configuration.Ports)
}

func (f *Forwarder) delayRetry(ctx context.Context) {
	select {
		case <- time.After(1 * time.Second):
		case <- ctx.Done():
			break
	}
}
