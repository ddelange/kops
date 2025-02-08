/*
Copyright 2017 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package dump

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// logDumper gets all the nodes from a kubernetes cluster and dumps a well-known set of logs
type logDumper struct {
	sshClientFactory sshClientFactory

	artifactsDir string

	services     []string
	files        []string
	podSelectors []string
}

// NewLogDumper is the constructor for a logDumper
func NewLogDumper(bastionAddress string, sshConfig *ssh.ClientConfig, keyRing agent.Agent, artifactsDir string) *logDumper {
	sshClientFactory := &sshClientFactoryImplementation{
		keyRing:   keyRing,
		sshConfig: sshConfig,
	}
	if bastionAddress != "" {
		log.Printf("detected a bastion instance, with the address: %s", bastionAddress)
		sshClientFactory.bastion = bastionAddress
	}

	d := &logDumper{
		sshClientFactory: sshClientFactory,
		artifactsDir:     artifactsDir,
	}

	d.services = []string{
		"node-problem-detector",
		"kubelet",
		"containerd",
		"docker",
		"kops-configuration",
		"protokube",
	}
	d.files = []string{
		"kops-controller",
		"kube-apiserver",
		"kube-scheduler",
		"rescheduler",
		"kube-controller-manager",
		"etcd",
		"etcd-events",
		"etcd-cilium",
		"glbc",
		"cluster-autoscaler",
		"kube-addon-manager",
		"fluentd",
		"kube-proxy",
		"node-problem-detector",
		"cloud-init-output",
		"startupscript",
		"kern",
		"docker",
		"aws-routed-eni/ipamd",
		"aws-routed-eni/plugin",
	}
	d.podSelectors = []string{
		"k8s-app=external-dns",
		"k8s-app=dns-controller",
	}

	return d
}

// DumpAllNodes connects to every node from kubectl get nodes and dumps the logs.
// additionalIPs holds IP addresses of instances found by the deployment tool;
// if the IPs are not found from kubectl get nodes, then these will be dumped also.
// This allows for dumping log on nodes even if they don't register as a kubernetes
// node, or if a node fails to register, or if the whole cluster fails to start.
func (d *logDumper) DumpAllNodes(ctx context.Context, nodes corev1.NodeList, maxNodesToDump int, additionalIPs, additionalPrivateIPs []string) error {
	var special, regular, dumped []*corev1.Node

	log.Printf("starting to dump %d nodes fetched through the Kubernetes APIs", len(nodes.Items))
	for i := range nodes.Items {
		node := &nodes.Items[i]

		if _, ok := node.Labels["node-role.kubernetes.io/master"]; ok {
			special = append(special, node)
			continue
		}
		if _, ok := node.Labels["node-role.kubernetes.io/control-plane"]; ok {
			special = append(special, node)
			continue
		}
		if _, ok := node.Labels["node-role.kubernetes.io/api-server"]; ok {
			special = append(special, node)
			continue
		}

		regular = append(regular, node)
	}

	for i := range special {
		node := special[i]
		err := d.dumpRegistered(ctx, node)
		if err != nil {
			log.Printf("could not dump node %s: %v", node.Name, err)
		} else {
			dumped = append(dumped, node)
		}
	}

	for i := range regular {
		if len(dumped) >= maxNodesToDump {
			log.Printf("stopping dumping nodes: %d nodes dumped", maxNodesToDump)
			return nil
		}
		node := regular[i]
		err := d.dumpRegistered(ctx, node)
		if err != nil {
			log.Printf("could not dump node %s: %v", node.Name, err)
		} else {
			dumped = append(dumped, node)
		}
	}

	notDumped := findInstancesNotDumped(additionalIPs, dumped)
	for _, ip := range notDumped {
		if len(dumped) >= maxNodesToDump {
			log.Printf("stopping dumping nodes: %d nodes dumped", maxNodesToDump)
			return nil
		}
		err := d.dumpNotRegistered(ctx, ip, false)
		if err != nil {
			return err
		}
	}

	notDumped = findInstancesNotDumped(additionalPrivateIPs, dumped)
	for _, ip := range notDumped {
		if len(dumped) >= maxNodesToDump {
			log.Printf("stopping dumping nodes: %d nodes dumped", maxNodesToDump)
			return nil
		}
		err := d.dumpNotRegistered(ctx, ip, true)
		if err != nil {
			return err
		}
	}

	return nil
}

func (d *logDumper) dumpRegistered(ctx context.Context, node *corev1.Node) error {
	if ctx.Err() != nil {
		log.Printf("stopping dumping nodes: %v", ctx.Err())
		return ctx.Err()
	}

	var publicIP, privateIP string
	for _, address := range node.Status.Addresses {
		if address.Type == "ExternalIP" {
			publicIP = address.Address
			break
		} else if address.Type == "InternalIP" {
			if privateIP == "" {
				privateIP = address.Address
			}
		}
	}

	if publicIP != "" {
		return d.dumpNode(ctx, node.Name, publicIP, false)
	} else {
		return d.dumpNode(ctx, node.Name, privateIP, true)
	}
}

func (d *logDumper) dumpNotRegistered(ctx context.Context, ip string, useBastion bool) error {
	if ctx.Err() != nil {
		log.Printf("stopping dumping nodes: %v", ctx.Err())
		return ctx.Err()
	}

	log.Printf("dumping node not registered in kubernetes: %s", ip)
	err := d.dumpNode(ctx, ip, ip, useBastion)
	if err != nil {
		log.Printf("error dumping node %s: %v", ip, err)
	}
	return nil
}

// findInstancesNotDumped returns ips from the slice that do not appear as any address of the nodes
func findInstancesNotDumped(ips []string, dumped []*corev1.Node) []string {
	var notDumped []string
	dumpedAddresses := make(map[string]bool)
	for _, node := range dumped {
		for _, address := range node.Status.Addresses {
			dumpedAddresses[address.Address] = true
		}
	}

	for _, ip := range ips {
		if !dumpedAddresses[ip] {
			notDumped = append(notDumped, ip)
		}
	}
	return notDumped
}

// DumpNode connects to a node and dumps the logs.
func (d *logDumper) dumpNode(ctx context.Context, name string, ip string, useBastion bool) error {
	if ip == "" {
		return fmt.Errorf("could not find address for %v, ", name)
	}

	log.Printf("Dumping node %s", name)

	n, err := d.connectToNode(ctx, name, ip, useBastion)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}

	// As long as we connect to the node we will not return an error;
	// a failure to collect a log (or even any logs at all) is not
	// considered an error in dumping the node.
	// TODO(justinsb): clean up / rationalize
	errors := n.dump(ctx)
	for _, e := range errors {
		log.Printf("error dumping node %s: %v", name, e)
	}

	if err := n.Close(); err != nil {
		log.Printf("error closing connection: %v", err)
	}

	return nil
}

// sshClient is an interface abstracting *ssh.Client, which allows us to test it
type sshClient interface {
	io.Closer

	// ExecPiped runs the command, piping stdout & stderr
	ExecPiped(ctx context.Context, command string, stdout io.Writer, stderr io.Writer) error
}

// sshClientFactory is an interface abstracting to a node over SSH
type sshClientFactory interface {
	Dial(ctx context.Context, host string, useBastion bool) (sshClient, error)
}

// logDumperNode holds state for a particular node we are dumping
type logDumperNode struct {
	client sshClient
	dumper *logDumper

	dir string
}

// connectToNode makes an SSH connection to the node and returns a logDumperNode
func (d *logDumper) connectToNode(ctx context.Context, nodeName string, host string, useBastion bool) (*logDumperNode, error) {
	client, err := d.sshClientFactory.Dial(ctx, host, useBastion)
	if err != nil {
		return nil, fmt.Errorf("unable to SSH to %q: %v", host, err)
	}
	return &logDumperNode{
		client: client,
		dir:    filepath.Join(d.artifactsDir, nodeName),
		dumper: d,
	}, nil
}

// logDumperNode cleans up any state in the logDumperNode
func (n *logDumperNode) Close() error {
	return n.client.Close()
}

// dump captures the well-known set of logs
func (n *logDumperNode) dump(ctx context.Context) []error {
	if ctx.Err() != nil {
		return []error{ctx.Err()}
	}

	var errors []error

	// Capture kernel log
	if err := n.shellToFile(ctx, "sudo journalctl --output=short-precise -k", filepath.Join(n.dir, "kern.log")); err != nil {
		errors = append(errors, err)
	}

	// Capture full journal - needed so we can see e.g. disk mounts
	// This does duplicate the other files, but ensures we have all output
	if err := n.shellToFile(ctx, "sudo journalctl --output=short-precise", filepath.Join(n.dir, "journal.log")); err != nil {
		errors = append(errors, err)
	}

	// Capture logs from any systemd services in our list that are registered
	services, err := n.listSystemdUnits(ctx)
	if err != nil {
		errors = append(errors, fmt.Errorf("error listing systemd services: %v", err))
	}
	for _, s := range n.dumper.services {
		name := s + ".service"
		for _, service := range services {
			if service == name {
				if err := n.shellToFile(ctx, "sudo journalctl --output=cat -u "+name, filepath.Join(n.dir, s+".log")); err != nil {
					errors = append(errors, err)
				}
			}
		}
	}

	// Capture iptables configuration
	if err := n.shellToFile(ctx, "sudo iptables -t nat --list-rules", filepath.Join(n.dir, "iptables-nat.log")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "sudo iptables -t filter --list-rules", filepath.Join(n.dir, "iptables-filter.log")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "sudo nft list ruleset", filepath.Join(n.dir, "nftables-ruleset.log")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "ip route show table all", filepath.Join(n.dir, "ip-routes.log")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "ip rule list", filepath.Join(n.dir, "ip-rules.log")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "ip -s link", filepath.Join(n.dir, "ip-link.log")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "ss -s", filepath.Join(n.dir, "netstat.log")); err != nil {
		errors = append(errors, err)
	}

	// Capture any file logs where the files exist
	fileList, err := n.findFiles(ctx, "/var/log")
	if err != nil {
		errors = append(errors, fmt.Errorf("error reading /var/log: %v", err))
	}
	for _, name := range n.dumper.files {
		prefix := "/var/log/" + name + ".log"
		for _, f := range fileList {
			if !strings.HasPrefix(f, prefix) {
				continue
			}
			if err := n.shellToFile(ctx, "sudo cat '"+strings.ReplaceAll(f, "'", "'\\''")+"'", filepath.Join(n.dir, strings.ReplaceAll(strings.TrimPrefix(f, "/var/log/"), "/", "_"))); err != nil {
				errors = append(errors, err)
			}
		}
	}

	for _, selector := range n.dumper.podSelectors {
		kv := strings.Split(selector, "=")
		logFile := fmt.Sprintf("%v.log", kv[len(kv)-1])
		if err := n.shellToFile(ctx, "if command -v kubectl &> /dev/null; then kubectl logs -n kube-system --all-containers -l \""+selector+"\"; fi", filepath.Join(n.dir, logFile)); err != nil {
			errors = append(errors, err)
		}
	}

	if err := n.shellToFile(ctx, "cat /etc/hosts", filepath.Join(n.dir, "etchosts")); err != nil {
		errors = append(errors, err)
	}
	if err := n.shellToFile(ctx, "sysctl -a", filepath.Join(n.dir, "sysctls")); err != nil {
		errors = append(errors, err)
	}

	return errors
}

// findFiles lists files under the specified directory (recursively)
func (n *logDumperNode) findFiles(ctx context.Context, dir string) ([]string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := n.client.ExecPiped(ctx, "sudo find "+dir+" -print0", &stdout, &stderr)
	if err != nil {
		return nil, fmt.Errorf("error listing %q: %v", dir, err)
	}

	paths := []string{}
	for _, b := range bytes.Split(stdout.Bytes(), []byte{0}) {
		if len(b) == 0 {
			// Likely the last value
			continue
		}
		paths = append(paths, string(b))
	}
	return paths, nil
}

// listSystemdUnits returns the list of systemd units on the node
func (n *logDumperNode) listSystemdUnits(ctx context.Context) ([]string, error) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := n.client.ExecPiped(ctx, "sudo systemctl list-units -t service --no-pager --no-legend --all", &stdout, &stderr)
	if err != nil {
		return nil, fmt.Errorf("error listing systemd units: %v", err)
	}

	var services []string
	for _, line := range strings.Split(stdout.String(), "\n") {
		tokens := strings.Fields(line)
		if len(tokens) == 0 || tokens[0] == "" {
			continue
		}
		services = append(services, tokens[0])
	}
	return services, nil
}

// shellToFile executes a command and copies the output to a file
func (n *logDumperNode) shellToFile(ctx context.Context, command string, destPath string) error {
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		log.Printf("unable to mkdir on %q: %v", filepath.Dir(destPath), err)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("error creating file %q: %v", destPath, err)
	}
	defer f.Close()

	if err := n.client.ExecPiped(ctx, command, f, f); err != nil {
		return fmt.Errorf("error executing command %q: %v", command, err)
	}

	return nil
}

// sshClientImplementation is the default implementation of sshClient, binding to a *ssh.Client
type sshClientImplementation struct {
	client    *ssh.Client
	forwardTo string
}

var _ sshClient = &sshClientImplementation{}

// ExecPiped implements sshClientImplementation::ExecPiped
func (s *sshClientImplementation) ExecPiped(ctx context.Context, cmd string, stdout io.Writer, stderr io.Writer) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}

	finished := make(chan error)
	go func() {
		session, err := s.client.NewSession()
		if err != nil {
			finished <- fmt.Errorf("error creating ssh session: %v", err)
			return
		}
		defer session.Close()

		session.Stdout = stdout
		session.Stderr = stderr

		if s.forwardTo != "" {
			cmd = fmt.Sprintf("ssh -o 'StrictHostKeyChecking no' %s %s", quoteShell(s.forwardTo), quoteShell(cmd))
		}

		klog.V(2).Infof("running SSH command: %v", cmd)

		finished <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		log.Print("closing SSH tcp connection due to context completion")

		// terminate the TCP connection to force a disconnect - we assume everyone is using the same context.
		// We could make this better by sending a signal on the session, waiting and then closing the session,
		// and only if we still haven't succeeded then closing the TCP connection.  This is sufficient for our
		// current usage though - and hopefully that logic will be implemented in the SSH package itself.
		s.Close()

		<-finished // Wait for cancellation
		return ctx.Err()

	case err := <-finished:
		return err
	}
}

func quoteShell(s string) string {
	var q strings.Builder
	q.WriteString("'")
	for _, c := range s {
		if c == '\'' || c == '\\' {
			q.WriteString("'\\")
		}
		q.WriteRune(c)
		if c == '\'' || c == '\\' {
			q.WriteRune('\'')
		}
	}
	q.WriteString("'")
	return q.String()
}

// Close implements sshClientImplementation::Close
func (s *sshClientImplementation) Close() error {
	return s.client.Close()
}

// sshClientFactoryImplementation is the default implementation of sshClientFactory
type sshClientFactoryImplementation struct {
	bastion   string
	sshConfig *ssh.ClientConfig
	keyRing   agent.Agent
}

var _ sshClientFactory = &sshClientFactoryImplementation{}

// Dial implements sshClientFactory::Dial
func (f *sshClientFactoryImplementation) Dial(ctx context.Context, host string, useBastion bool) (sshClient, error) {
	var addr string
	if useBastion {
		addr = f.bastion
	} else {
		addr = host
	}
	addr = net.JoinHostPort(addr, "22")
	d := net.Dialer{
		Timeout: 5 * time.Second,
	}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, err
	}

	// We have a TCP connection; we will force-close it to support context cancellation

	var client *ssh.Client
	finished := make(chan error)
	go func() {
		c, chans, reqs, err := ssh.NewClientConn(conn, addr, f.sshConfig)
		if err == nil {
			client = ssh.NewClient(c, chans, reqs)
			if useBastion {
				err = agent.ForwardToAgent(client, f.keyRing)
				if err != nil {
					err = fmt.Errorf("forwarding ssh auth to keyring: %w", err)
				}
			}
		}

		if err == nil && useBastion {
			session, err := client.NewSession()
			if err != nil {
				finished <- fmt.Errorf("creating ssh session: %w", err)
				return
			}
			defer session.Close()

			err = agent.RequestAgentForwarding(session)
			if err != nil {
				finished <- fmt.Errorf("requesting agent forwarding: %w", err)
				return
			}
		}

		finished <- err
	}()

	select {
	case <-ctx.Done():
		log.Print("cancelling SSH tcp connection due to context completion")
		conn.Close() // Close the TCP connection to force cancellation
		<-finished   // Wait for cancellation
		return nil, ctx.Err()
	case err := <-finished:
		if err != nil {
			return nil, err
		}
		if !useBastion {
			host = ""
		}
		return &sshClientImplementation{
			client:    client,
			forwardTo: host,
		}, nil
	}
}
