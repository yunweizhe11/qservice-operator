/*
Copyright 2022.

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

package v1beta1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//存活探针 struct
type LivenessProbe struct {
	Action              string `json:"action,omitempty"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32  `json:"periodSeconds,omitempty"`
}

//就绪探针
type ReadinessProbe struct {
	Action              string `json:"action,omitempty"`
	InitialDelaySeconds int32  `json:"initialDelaySeconds,omitempty"`
	PeriodSeconds       int32  `json:"periodSeconds,omitempty"`
}

type IngressPath struct {
	Path string `json:"path,omitempty"`
	Port int64  `json:"port,omitempty"`
}
type IngressSpec struct {
	Domain string        `json:"domain,omitempty"`
	Paths  []IngressPath `json:"paths,omitempty"`
}

type MountSpec struct {
	Type       string `json:"type,omitempty"`
	Name       string `json:"name,omitempty"`
	Pvc        string `json:"pvc,omitempty"`
	Path       string `json:"path,omitempty"`
	Targetpath string `json:"targetpath,omitempty"`
	Subpath    string `json:"subpath,omitempty"`
}

type resourcelimt struct {
	Cpu    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

type ResourcesSpec struct {
	Limits   resourcelimt `json:"limits,omitempty"`
	Requests resourcelimt `json:"requests,omitempty"`
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// QserviceSpec defines the desired state of Qservice
type QserviceSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of Qservice. Edit qservice_types.go to remove/update
	Envs           map[string]string `json:"envs,omitempty"`           //对应服务环境变量map
	Image          string            `json:"image,omitempty"`          //对应服务镜像地址
	LivenessProbe  LivenessProbe     `json:"livenessProbe,omitempty"`  //存活探针
	ReadinessProbe ReadinessProbe    `json:"readinessProbe,omitempty"` //就绪探针
	Ports          []string          `json:"ports,omitempty"`          //端口列表
	Ingress        []IngressSpec     `json:"ingress,omitempty"`        //ingress
	Mount          []MountSpec       `json:"mount,omitempty"`          //挂载
	Resources      ResourcesSpec     `json:"resources,omitempty"`      //resource资源
}

// QserviceStatus defines the observed state of Qservice
type QserviceStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Qservice is the Schema for the qservices API
type Qservice struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   QserviceSpec   `json:"spec,omitempty"`
	Status QserviceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// QserviceList contains a list of Qservice
type QserviceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Qservice `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Qservice{}, &QserviceList{})
}
