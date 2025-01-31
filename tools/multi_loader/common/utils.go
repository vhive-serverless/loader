package common

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/vhive-serverless/loader/pkg/config"
	"github.com/vhive-serverless/loader/tools/multi_loader/types"
)

func ReadMultiLoaderConfigurationFile(path string) types.MultiLoaderConfiguration {
	byteValue, err := os.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}

	var config types.MultiLoaderConfiguration
	err = json.Unmarshal(byteValue, &config)
	if err != nil {
		log.Fatal(err)
	}

	return config
}

func WriteMultiLoaderConfigurationFile(config types.MultiLoaderConfiguration, path string) {
	configByteValue, err := json.Marshal(config)
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(path, configByteValue, 0644)
	if err != nil {
		log.Fatal(err)
	}
}

func DeterminePlatformFromConfig(multiLoaderConfig types.MultiLoaderConfiguration) string {
	// Determine platform
	baseConfigByteValue, err := os.ReadFile(multiLoaderConfig.BaseConfigPath)
	if err != nil {
		log.Fatal(err)
	}
	var loaderConfig config.LoaderConfiguration
	// Unmarshal base configuration
	if err = json.Unmarshal(baseConfigByteValue, &loaderConfig); err != nil {
		log.Fatal(err)
	}
	return loaderConfig.Platform
}

func DetermineWorkerNodeIPs() []string {
	out := DetermineNodeIP(Worker)
	workerNodes := strings.Split(out, " \n")
	for i := range workerNodes {
		workerNodes[i] = strings.TrimSpace(workerNodes[i])
	}
	return workerNodes
}

func DetermineNodeIP(node NodeType) string {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("kubectl get nodes --show-labels --no-headers -o wide | grep nodetype=%s | awk '{print $6}'", node))
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	return strings.Trim(string(out), " \n")
}

func DeterminePodIP(podNamePrefix PodType) string {
	// Get the pod alias
	cmdPodName := exec.Command("sh", "-c", fmt.Sprintf("kubectl get pods -n knative-serving --no-headers | grep %s- | awk '{print $1}'", podNamePrefix))
	out, err := cmdPodName.CombinedOutput()

	if err != nil {
		log.Fatal("Error getting", podNamePrefix, "pod name:", err)
	}

	// Get the private ip using the pod alias
	podName := strings.Trim(string(out), "\n")
	cmdNodeIP := exec.Command("sh", "-c", fmt.Sprintf("kubectl get pod %s -n knative-serving -o=jsonpath='{.status.hostIP}'", podName))
	out, err = cmdNodeIP.CombinedOutput()

	if err != nil {
		log.Fatal("Error getting", cmdNodeIP, "node IP:", err)
	}

	nodeIp := strings.Split(string(out), "\n")[0]
	return strings.Trim(nodeIp, " ")
}

func DetermineNodesIPs(multiLoaderConfig types.MultiLoaderConfiguration) types.NodeGroup {
	log.Debug("Determining node IPs")
	nodeGroup := types.NodeGroup{
		MasterNode:     multiLoaderConfig.MasterNode,
		AutoScalerNode: multiLoaderConfig.AutoScalerNode,
		ActivatorNode:  multiLoaderConfig.ActivatorNode,
		LoaderNode:     multiLoaderConfig.LoaderNode,
		WorkerNodes:    multiLoaderConfig.WorkerNodes,
	}

	// Only a single node will be present in KinD Cluster
	if IsKinD() {
		log.Debug("KinD Cluster detected")
		nodeIP := DetermineNodeIP(Worker)
		assignDefaults(&nodeGroup.MasterNode, nodeIP)
		assignDefaults(&nodeGroup.LoaderNode, nodeIP)
		assignDefaultsSlice(&nodeGroup.WorkerNodes, []string{nodeIP})
	} else {
		assignDefaults(&nodeGroup.MasterNode, DetermineNodeIP(Master))
		assignDefaults(&nodeGroup.LoaderNode, DetermineNodeIP(Loader))
		assignDefaultsSlice(&nodeGroup.WorkerNodes, DetermineWorkerNodeIPs())
	}

	assignDefaults(&nodeGroup.AutoScalerNode, DeterminePodIP(AutoScalerPod))
	assignDefaults(&nodeGroup.ActivatorNode, DeterminePodIP(ActivatorPod))

	log.Trace("Node IPs determined", nodeGroup)
	return nodeGroup
}

func IsKinD() bool {
	cmd := exec.Command("sh", "-c", "kind get clusters")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(out)) == "knative"
}

// Helper functions	for assigning default values
func assignDefaults(target *string, value string) {
	if *target == "" {
		*target = value
	}
}

func assignDefaultsSlice(target *[]string, value []string) {
	if len(*target) == 0 {
		*target = value
	}
}

func RunRemoteCommand(node string, command string) {
	cmd := exec.Command("ssh", "-oStrictHostKeyChecking=no", "-p 22", node, command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	if len(output) > 0 {
		log.Debug(node, string(output))
	}

}

func CopyRemoteFile(remoteNode, src string, dest string) {
	cmd := exec.Command("scp", "-rp", remoteNode+":"+src, dest)
	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	if len(out) > 0 {
		log.Debug(string(out))
	}
}
