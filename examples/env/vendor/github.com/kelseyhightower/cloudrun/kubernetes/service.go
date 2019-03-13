package kubernetes

type Service struct {
	Spec ServiceSpec `json:"spec,omitempty"`
}

type ServiceSpec struct {
	RunLatest RunLatest `json:"runLatest,omitempty"`
}

type RunLatest struct {
	Configuration Configuration `json:"configuration,omitempty"`
}

type Configuration struct {
	RevisionTemplate RevisionTemplate `json:"revisionTemplate"`
}

type RevisionTemplate struct {
	Spec RevisionSpec `json:"spec,omitempty"`
}

type RevisionSpec struct {
	Container Container `json:"container"`
}

type Container struct {
	Env []EnvVar `json:"env,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}
