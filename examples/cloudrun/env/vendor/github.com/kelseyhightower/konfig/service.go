// Copyright 2019 The Konfig Authors. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.

package konfig

type Service struct {
	Spec ServiceSpec `json:"spec,omitempty"`
}

type ServiceSpec struct {
	RevisionTemplate RevisionTemplate `json:"template,omitempty"`
}

type RevisionTemplate struct {
	Spec RevisionSpec `json:"spec,omitempty"`
}

type RevisionSpec struct {
	Containers []Container `json:"containers"`
}

type Container struct {
	Env []EnvVar `json:"env,omitempty"`
}

type EnvVar struct {
	Name  string `json:"name"`
	Value string `json:"value,omitempty"`
}
