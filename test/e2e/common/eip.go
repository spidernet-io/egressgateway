// Copyright 2022 Authors of spidernet-io
// SPDX-License-Identifier: Apache-2.0

package common

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/spidernet-io/e2eframework/framework"
	corev1 "k8s.io/api/core/v1"
)

func generateCmd(f *framework.Framework, pod *corev1.Pod, serverIP string, ctx context.Context) *exec.Cmd {
	curlServer := fmt.Sprintf("nettools-client -addr %s -protocol %s -tcpPort %s -udpPort %s -webPort %s", serverIP, Env[MOD], Env[TCP_PORT], Env[UDP_PORT], Env[WEB_PORT])
	args := fmt.Sprintf("kubectl --kubeconfig %s exec %s -n %s -- %s", f.Info.KubeConfigPath, pod.Name, pod.Namespace, curlServer)
	return exec.CommandContext(ctx, "sh", "-c", args)
}

func CheckEIPinClientPod(f *framework.Framework, pod *corev1.Pod, eIP, serverIP string, expect bool, retry int, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	retryNum := 0

RETRY:
	cmd := generateCmd(f, pod, serverIP, ctx)
	r, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	cmd.Stderr = cmd.Stdout
	err = cmd.Start()
	if err != nil {
		return err
	}

	var udpOk, tcpOk, webSocketOk bool
	for {
		select {
		case <-ctx.Done():
			err = cmd.Process.Kill()
			if err != nil {
				return err
			}
			return ERR_TIMEOUT
		default:
			tmp := make([]byte, 1024)
			_, err = r.Read(tmp)
			if err == io.EOF {
				return ERR_CHECK_EIP
			}
			if err != nil {
				return err
			}
			out := string(tmp)
			GinkgoWriter.Println(out)

			if strings.Contains(out, resetByPeer) {
				if retryNum < retry {
					retryNum++
					goto RETRY
				}
			}
			if expect {
				if strings.Contains(out, WEBSOCKET) && strings.Contains(out, eIP) {
					webSocketOk = true
				}
				if strings.Contains(out, UDP) && strings.Contains(out, eIP) {
					udpOk = true
				}
				if strings.Contains(out, TCP) && strings.Contains(out, eIP) {
					tcpOk = true
				}
				if udpOk && tcpOk && webSocketOk {
					return nil
				}
			} else {
				if strings.Contains(out, WEBSOCKET) && !strings.Contains(out, eIP) {
					webSocketOk = true
				}
				if strings.Contains(out, UDP) && !strings.Contains(out, eIP) {
					udpOk = true
				}
				if strings.Contains(out, TCP) && !strings.Contains(out, eIP) {
					tcpOk = true
				}
				if udpOk && tcpOk && webSocketOk {
					return nil
				}
			}
			err = cmd.Process.Kill()
			if err != nil {
				return err
			}
			time.Sleep(time.Second)
			goto RETRY
		}
	}
}

func GetClientPodLog(f *framework.Framework, pod *corev1.Pod, serverIP string, connectionTimeout time.Duration, logDuration time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), connectionTimeout)
	defer cancel()

	cmd := generateCmd(f, pod, serverIP, ctx)

	var output bytes.Buffer
	cmd.Stdout = &output

	err := cmd.Start()
	if err != nil {
		return "", err
	}

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-time.After(logDuration):
		err := cmd.Process.Kill()
		if err != nil {
			return "", err
		}
		<-done
		return output.String(), nil
	case err := <-done:
		return output.String(), err
	}
}

func CheckEipInClientPod(f *framework.Framework, pod *corev1.Pod, eIP, serverIP string, expect bool, retry, logDuration time.Duration) error {
	timeout := logDuration * time.Duration(retry+2)
	connectionTimeout := logDuration * time.Duration(retry+1)
	ctx, cancel := context.WithTimeout(context.TODO(), timeout)
	defer cancel()

	var udpOk, tcpOk, webSocketOk bool

	for {
		select {
		case <-ctx.Done():
			return ERR_TIMEOUT
		default:
			out, err := GetClientPodLog(f, pod, serverIP, connectionTimeout, logDuration)
			Expect(err).NotTo(HaveOccurred())
			GinkgoWriter.Printf("the client-pod log: \n%s\n", out)
			if expect {
				if strings.Contains(out, WEBSOCKET) && strings.Contains(out, eIP) {
					webSocketOk = true
				}
				if strings.Contains(out, UDP) && strings.Contains(out, eIP) {
					udpOk = true
				}
				if strings.Contains(out, TCP) && strings.Contains(out, eIP) {
					tcpOk = true
				}
				if udpOk && tcpOk && webSocketOk {
					return nil
				}
			} else {
				if strings.Contains(out, WEBSOCKET) && !strings.Contains(out, eIP) {
					webSocketOk = true
				}
				if strings.Contains(out, UDP) && !strings.Contains(out, eIP) {
					udpOk = true
				}
				if strings.Contains(out, TCP) && !strings.Contains(out, eIP) {
					tcpOk = true
				}
				if udpOk && tcpOk && webSocketOk {
					return nil
				}
			}
			time.Sleep(time.Second)
		}
	}
}
