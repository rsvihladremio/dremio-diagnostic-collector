package clusterstats

type ClusterStats struct {
	DremioVersion string `json:"dremioVersion"`
	ClusterID     string `json:"clusterID"`
	NodeName      string `json:"nodeName"`
}
