package common

import (
	"fmt"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
)

func NewClientInCluster() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func MustNewClientInCluster() *kubernetes.Clientset {
	client, err := NewClientInCluster()
	if err != nil {
		panic(err)
	}
	return client
}

func NewClientFromKubeconf(kubeconf string) (*kubernetes.Clientset, error) {
	config, err := NewConfigFromKubeconf(kubeconf)
	if err != nil {
		return nil, err
	}
	return kubernetes.NewForConfig(config)
}

func NewConfigFromKubeconf(kubeconf string) (*rest.Config, error) {
	config, err := clientcmd.BuildConfigFromFlags("", kubeconf)
	if err != nil {
		return nil, err
	}
	return config, nil
}

func ExitSignal() <-chan os.Signal {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGTERM, syscall.SIGQUIT)
	return ch
}

func DumpSignal() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGUSR1)
	for range ch {
		if _, err := DumpStacks("/var/log"); err != nil {
			klog.Error(err.Error())
		}
	}
}

func DumpStacks(dir string) (string, error) {
	var (
		buf       []byte
		stackSize int
	)
	bufferLen := 16384
	for stackSize == len(buf) {
		buf = make([]byte, bufferLen)
		stackSize = runtime.Stack(buf, true)
		bufferLen *= 2
	}
	buf = buf[:stackSize]
	var f *os.File
	if dir != "" {
		path := filepath.Join(dir, fmt.Sprintf("goroutine-stacks-%s.log", strings.Replace(time.Now().Format(time.RFC3339), ":", "", -1)))
		var err error
		f, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return "", fmt.Errorf("failed to open file to write the goroutine stacks: %s", err.Error())
		}
		defer f.Close()
	} else {
		f = os.Stderr
	}
	if _, err := f.Write(buf); err != nil {
		return "", fmt.Errorf("failed to write goroutine stacks: %s", err.Error())
	}
	return f.Name(), nil
}

func NewFSWatcher(files ...string) (*fsnotify.Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		err = watcher.Add(f)
		if err != nil {
			watcher.Close()
			return nil, err
		}
	}

	return watcher, nil
}
