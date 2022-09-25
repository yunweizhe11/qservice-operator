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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	appsv2 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	netV1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "gitee.com/yunweizhe/operator-qservice/api/v1beta1"
)

// QserviceReconciler reconciles a Qservice object
type QserviceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func int32Ptr(i int32) *int32 { return &i }

// 处理挂载
func volumeSpec(instance *appsv1.Qservice) []corev1.Volume {
	mountVolumes := []corev1.Volume{}
	if len(instance.Spec.Mount) == 0 {
		return mountVolumes
	}
	// corev1.VolumeSource.PersistentVolumeClaim
	for _, volume := range instance.Spec.Mount {
		volumes := corev1.Volume{}
		// VolumeSource := corev1.VolumeSource{}
		volumes.Name = volume.Name
		if volume.Type == "pvc" {
			volumes := corev1.Volume{
				Name: volume.Name, VolumeSource: corev1.VolumeSource{
					PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
						ClaimName: volume.Pvc,
					},
				},
			}
			mountVolumes = append(mountVolumes, volumes)
		} else if volume.Type == "hostpath" {
			volumes := corev1.Volume{
				Name: volume.Name,
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: volume.Path,
					},
				},
			}
			mountVolumes = append(mountVolumes, volumes)
		}
	}
	return mountVolumes
}

// newContainer
func newContainer(instance *appsv1.Qservice, r *QserviceReconciler) []corev1.Container {
	log := r.Log.WithValues("newContainer", instance.Name)
	containerPorts := []corev1.ContainerPort{}
	for _, svcPort := range instance.Spec.Ports {
		cport := corev1.ContainerPort{}
		conport := strings.Split(svcPort, ":") //端口分隔符
		// cport.ContainerPort = conport[len(conport)-1] //获取容器内端口
		port, err := strconv.Atoi(conport[len(conport)-1])
		if err != nil {
			log.Error(err, "port str to int fail")
		}
		cport.ContainerPort = int32(port)
		containerPorts = append(containerPorts, cport)
	}
	volumeMounts := []corev1.VolumeMount{}
	for _, mount := range instance.Spec.Mount {
		mounts := corev1.VolumeMount{}
		mounts.Name = mount.Name
		mounts.MountPath = mount.Targetpath
		if mount.Subpath != "" {
			mounts.SubPath = mount.Subpath
		}
		volumeMounts = append(volumeMounts, mounts)
	}
	EnvsList := []corev1.EnvVar{}
	for envName, envValue := range instance.Spec.Envs {
		Envs := corev1.EnvVar{}
		Envs.Name = envName
		Envs.Value = envValue
		EnvsList = append(EnvsList, Envs)
	}
	//check Handler
	LivenessProbe_Action, err := url.Parse(instance.Spec.LivenessProbe.Action)
	if err != nil {
		log.Error(err, "LivenessProbe_Action parse fail")
	}
	ReadinessProbe_Action, err := url.Parse(instance.Spec.ReadinessProbe.Action)
	if err != nil {
		log.Error(err, "ReadinessProbe_Action parse fail")
	}
	LivenessProbe_Action_port, _ := strconv.ParseInt(strings.Split(LivenessProbe_Action.Host, ":")[1], 10, 32)
	ReadinessProbe_Action_port, _ := strconv.ParseInt(strings.Split(ReadinessProbe_Action.Host, ":")[1], 10, 32)
	return []corev1.Container{
		{
			Name:            instance.Name,
			Image:           instance.Spec.Image,
			Ports:           containerPorts,
			ImagePullPolicy: corev1.PullIfNotPresent,
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					"cpu":    resource.MustParse(instance.Spec.Resources.Requests.Cpu),
					"memory": resource.MustParse(instance.Spec.Resources.Requests.Memory),
				},
				Limits: corev1.ResourceList{
					"cpu":    resource.MustParse(instance.Spec.Resources.Limits.Cpu),
					"memory": resource.MustParse(instance.Spec.Resources.Limits.Memory),
				},
			},
			Env:          EnvsList,
			VolumeMounts: volumeMounts,
			LivenessProbe: &corev1.Probe{
				InitialDelaySeconds: instance.Spec.LivenessProbe.InitialDelaySeconds,
				PeriodSeconds:       instance.Spec.LivenessProbe.PeriodSeconds,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   LivenessProbe_Action.Path,
						Port:   intstr.IntOrString{IntVal: int32(LivenessProbe_Action_port)},
						Host:   strings.Split(LivenessProbe_Action.Host, ":")[0],
						Scheme: corev1.URIScheme(strings.ToUpper(LivenessProbe_Action.Scheme)),
					},
				},
			},
			ReadinessProbe: &corev1.Probe{
				InitialDelaySeconds: instance.Spec.ReadinessProbe.InitialDelaySeconds,
				PeriodSeconds:       instance.Spec.ReadinessProbe.PeriodSeconds,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path:   ReadinessProbe_Action.Path,
						Port:   intstr.IntOrString{IntVal: int32(ReadinessProbe_Action_port)},
						Host:   strings.Split(ReadinessProbe_Action.Host, ":")[0],
						Scheme: corev1.URIScheme(strings.ToUpper(ReadinessProbe_Action.Scheme)),
					},
				},
			},
		},
	}
}

func NewDployment(instance *appsv1.Qservice, r *QserviceReconciler) *appsv2.Deployment {
	labels := map[string]string{"app": instance.Name}
	selector := metav1.LabelSelector{MatchLabels: labels}
	deployment := appsv2.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        instance.Name,
			Namespace:   instance.Namespace,
			Labels:      labels,
			Annotations: instance.Annotations,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, schema.GroupVersionKind{
					Group:   appsv1.GroupVersion.Group,
					Version: appsv1.GroupVersion.Version,
					Kind:    instance.Kind,
				}),
			},
		},
		Spec: appsv2.DeploymentSpec{
			Replicas: int32Ptr(1),
			Selector: &selector,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Name:   instance.Name,
				},
				Spec: corev1.PodSpec{Containers: newContainer(instance, r), Volumes: volumeSpec(instance)},
			},
		},
	}
	return &deployment
}

//根据参数进行创建service
func Toservice(instance *appsv1.Qservice, req ctrl.Request) *corev1.Service {
	Ports := []corev1.ServicePort{}
	var ServiceType corev1.ServiceType
	Ser_type := 0
	for _, singleport := range instance.Spec.Ports {
		port := corev1.ServicePort{}
		port_list := strings.Split(singleport, ":")
		if len(port_list) == 2 {
			sourceport, _ := strconv.ParseInt(port_list[len(port_list)-1], 10, 32)
			nodePort, _ := strconv.ParseInt(port_list[0], 10, 32)
			targetPort := intstr.IntOrString{IntVal: int32(sourceport)}
			port.Name = instance.Name + strconv.FormatInt(int64(sourceport), 10)
			port.TargetPort = targetPort
			port.Port = int32(sourceport)
			port.NodePort = int32(nodePort)
			port.Protocol = corev1.ProtocolTCP
			Ser_type = Ser_type + 1
		} else {
			sourceport, _ := strconv.ParseInt(port_list[len(port_list)-1], 10, 32)
			targetPort := intstr.IntOrString{IntVal: int32(sourceport)}
			port.Name = instance.Name + strconv.FormatInt(int64(sourceport), 10)
			port.TargetPort = targetPort
			port.Port = int32(sourceport)
			port.Protocol = corev1.ProtocolTCP
		}
		Ports = append(Ports, port)

	}
	if Ser_type == 0 {
		ServiceType = corev1.ServiceTypeClusterIP
	} else {
		ServiceType = corev1.ServiceTypeNodePort
	}
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			Labels:    map[string]string{"app": instance.Name},
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, schema.GroupVersionKind{
					Group:   appsv1.GroupVersion.Group,
					Version: appsv1.GroupVersion.Version,
					Kind:    instance.Kind,
				}),
			},
		},

		Spec: corev1.ServiceSpec{
			Ports:    Ports,
			Selector: map[string]string{"app": instance.Name},
			Type:     ServiceType,
		},
	}
}

//根据参数进行创建ingress
func Toingress(instance *appsv1.Qservice, req ctrl.Request) *netV1.Ingress {
	pathType := netV1.PathTypePrefix
	// pathType := netV1.PathTypeImplementationSpecific
	roles := []netV1.IngressRule{}
	for _, domain := range instance.Spec.Ingress {
		role := netV1.IngressRule{}
		Paths := []netV1.HTTPIngressPath{}
		role.Host = domain.Domain
		for _, Ipath := range domain.Paths {
			path := netV1.HTTPIngressPath{}
			path.Path = Ipath.Path
			// path.Path = "/"
			path.PathType = &pathType
			path.Backend = netV1.IngressBackend{
				Service: &netV1.IngressServiceBackend{
					Name: instance.Name,
					Port: netV1.ServiceBackendPort{
						// Name:   instance.Name,
						Number: int32(Ipath.Port),
					},
				},
			}
			Paths = append(Paths, path)
		}
		role.IngressRuleValue = netV1.IngressRuleValue{
			HTTP: &netV1.HTTPIngressRuleValue{
				Paths: Paths,
			},
		}
		roles = append(roles, role)
	}
	ingress := netV1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: "extensions/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				*metav1.NewControllerRef(instance, schema.GroupVersionKind{
					Group:   appsv1.GroupVersion.Group,
					Version: appsv1.GroupVersion.Version,
					Kind:    instance.Kind,
				}),
			},
		},
		Spec: netV1.IngressSpec{
			Rules: roles,
		},
	}

	return &ingress
}

func Todeployment(instance *appsv1.Qservice, req ctrl.Request, r *QserviceReconciler) error {
	ctx := context.Background()
	log := r.Log.WithValues("Todeployment", req.NamespacedName)
	log.Info("format deployment")
	deployment_instance := &appsv2.Deployment{}
	log.Info("format deployment end")
	if err := r.Get(ctx, req.NamespacedName, deployment_instance); err != nil {
		if !errors.IsNotFound(err) { //deployment 获取报错 报错不是不存在
			return err
		}
		log.Info("deployment create loadding..")
		deployment := NewDployment(instance, r)
		if err := r.Client.Create(ctx, deployment); err != nil {
			return err
		}
		log.Info("service create loadding..")
		svc := Toservice(instance, req)
		if err := r.Client.Create(ctx, svc); err != nil {
			return err
		}
		if len(instance.Spec.Ingress) == 0 {
			log.Info("ingress is None")
		} else {
			log.Info("ingress create loadding...")
			ingress := Toingress(instance, req)
			if err := r.Client.Create(ctx, ingress); err != nil {
				return err
			}
		}

	} else {
		oldSpec := &appsv1.QserviceSpec{}
		if err := json.Unmarshal([]byte(instance.Annotations["spec"]), oldSpec); err != nil {
			return err
		}
		//d对比
		if !reflect.DeepEqual(instance.Spec, *oldSpec) {
			newDeployment1 := NewDployment(instance, r)
			currDeployment := &appsv2.Deployment{}
			if err := r.Client.Get(ctx, req.NamespacedName, currDeployment); err != nil {
				return err
			}
			currDeployment.Spec = newDeployment1.Spec
			if err := r.Client.Update(ctx, currDeployment); err != nil {
				return err
			}
			//update service
			newservice := Toservice(instance, req)
			currentService := &corev1.Service{}
			if err := r.Client.Get(ctx, req.NamespacedName, currentService); err != nil {
				return err
			}
			ClusterIP := currentService.Spec.ClusterIP
			currentService.Spec = newservice.Spec
			currentService.Spec.ClusterIP = ClusterIP
			if err := r.Client.Update(ctx, currentService); err != nil {
				return err
			}
			if len(instance.Spec.Ingress) == 0 {
				log.Info("ingress is None")
			} else {
				//update ingress
				newIngress := Toingress(instance, req)
				currentIngress := &netV1.Ingress{}
				if err := r.Client.Get(ctx, req.NamespacedName, currentIngress); err != nil {
					return err
				}
				currentIngress.Spec = newIngress.Spec
				if err := r.Client.Update(ctx, currentIngress); err != nil {
					return err
				}
			}
		}
	}
	// 关联Annotations
	data, _ := json.Marshal(instance.Spec)
	if instance.Annotations != nil {
		instance.Annotations["spec"] = string(data)
	} else {
		instance.Annotations = map[string]string{"spec": string(data)}
	}
	if err := r.Client.Update(ctx, instance); err != nil {
		return err
	}
	return nil
}

func CreateQservice(instance *appsv1.Qservice, req ctrl.Request, r *QserviceReconciler) error {
	instance = CheckArgs(instance)
	fmt.Println("CreateQservice loading...")
	r.Log.Info("CreateQservice loading...")
	deployment_err := Todeployment(instance, req, r)
	if deployment_err != nil {
		return deployment_err
	}
	return nil
}

func CheckArgs(instance *appsv1.Qservice) *appsv1.Qservice {
	//check resources
	if instance.Spec.Resources.Limits.Cpu == "" {
		instance.Spec.Resources.Limits.Cpu = "500m"
	}
	if instance.Spec.Resources.Limits.Memory == "" {
		instance.Spec.Resources.Limits.Memory = "500Mi"
	}
	if instance.Spec.Resources.Requests.Cpu == "" {
		instance.Spec.Resources.Requests.Cpu = "500m"
	}
	if instance.Spec.Resources.Requests.Memory == "" {
		instance.Spec.Resources.Requests.Memory = "500Mi"
	}
	//check mount

	//check ingress
	//check port
	return instance
}

//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="apps",resources=deployments/status,verbs=get
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",resources=services/status,verbs=get
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="networking.k8s.io",resources=ingresses/status,verbs=get

//+kubebuilder:rbac:groups=apps.tech,resources=qservices,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.tech,resources=qservices/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.tech,resources=qservices/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Qservice object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *QserviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("Qservice", req.NamespacedName)
	instance := &appsv1.Qservice{}
	err := r.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("2.1. instance not found, maybe removed")
			return ctrl.Result{}, nil
		}
		log.Error(err, "2.2 error")
		return ctrl.Result{}, err
	}
	if instance.DeletionTimestamp != nil {
		return reconcile.Result{}, nil
	}
	// if instance.Spec.Resources.Requests.Cpu == "" {
	// 	fmt.Println("request/cpu is None")
	// 	instance.Spec.Resources.Requests.Cpu = "1"
	// 	instance.Spec.Resources.Requests.Memory = "100Mi"
	// }
	err1 := CreateQservice(instance, req, r)
	if err1 != nil {
		log.Error(err1, "create qservice fail")
		return ctrl.Result{}, err
	}
	log.Info("qservice create success", req.Name, req.NamespacedName)
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *QserviceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Qservice{}).
		Complete(r)
}
