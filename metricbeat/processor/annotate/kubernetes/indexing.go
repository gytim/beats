package kubernetes

import (
	"fmt"
	"strings"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors/annotate/kubernetes"
	corev1 "github.com/ericchiang/k8s/api/v1"
)

const (
	IpPortIndexerName = "ip_port"
)

func init() {
	// Register default indexers
	kubernetes.Indexing.AddIndexer(IpPortIndexerName, newIpPortIndexer)
	cfg := common.NewConfig()

	//Add IP Port Indexer as a default indexer
	kubernetes.Indexing.AddDefaultIndexerConfig(IpPortIndexerName, *cfg)

	config := map[string]interface{}{
		"lookup_fields": []string{"metricset.host"},
	}
	fieldCfg, err := common.NewConfigFrom(config)
	if err == nil {
		//Add field matcher with field to lookup as metricset.host
		kubernetes.Indexing.AddDefaultMatcherConfig(kubernetes.FieldMatcherName, *fieldCfg)
	}
}

// IpPortIndexer indexes pods based on all their host:port combinations
type IpPortIndexer struct {
	genMeta kubernetes.GenMeta
}

func newIpPortIndexer(_ common.Config, genMeta kubernetes.GenMeta) (kubernetes.Indexer, error) {
	return &IpPortIndexer{genMeta: genMeta}, nil
}

func (h *IpPortIndexer) GetMetadata(pod *corev1.Pod) []kubernetes.MetadataIndex {
	commonMeta := h.genMeta.GenerateMetaData(pod)
	hostPorts := h.GetIndexes(pod)
	var metadata []kubernetes.MetadataIndex

	if pod.Status.PodIP == nil {
		return metadata
	}
	for i := 0; i < len(hostPorts); i++ {
		dobreak := false
		containerMeta := commonMeta.Clone()
		for _, container := range pod.Spec.Containers {
			ports := container.Ports

			for _, port := range ports {
				if port.ContainerPort == nil {
					continue
				}
				if strings.Index(hostPorts[i], fmt.Sprintf("%s:%d", *pod.Status.PodIP, *port.ContainerPort)) != -1 {
					containerMeta["container"] = container.Name
					dobreak = true
					break
				}
			}

			if dobreak {
				break
			}

		}

		metadata = append(metadata, kubernetes.MetadataIndex{
			Index: hostPorts[i],
			Data:  containerMeta,
		})
	}

	return metadata
}

func (h *IpPortIndexer) GetIndexes(pod *corev1.Pod) []string {
	var hostPorts []string

	ip := pod.Status.PodIP
	if ip == nil {
		return hostPorts
	}
	for _, container := range pod.Spec.Containers {
		ports := container.Ports

		for _, port := range ports {
			if port.ContainerPort != nil {
				hostPorts = append(hostPorts, fmt.Sprintf("%s:%d", *ip, *port.ContainerPort))
			} else {
				hostPorts = append(hostPorts, *ip)
			}

		}

	}

	return hostPorts
}
