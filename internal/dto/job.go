package dto

type JobRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
}

type Job struct {
	Name        string
	Namespace   string
	Completions int32
	Age         string
	Conditions  []string
	Resumed     bool
	Status      string
	Succeeded   int32
	Parallelism int32
	Duration    string

	CreatedAt      string
	Labels         map[string]string
	Annotations    map[string]string
	ManagedFields  []ManagedField
	Selector       string
	CompletionMode string
	StartTime      string
	StartTimeAge   string
	CompletedAt    string
	CompletedAtAge string
	PodsStatuses   string
	PodStatus      string
}
