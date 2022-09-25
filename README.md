# qservice operator

 通过简单配置创建kubernetes deployment/service/ingress 资源




```bash
kubectl get deploy,svc,ingress -n dev

NAME                          READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/elastic-ple   2/2     2            2           25d
deployment.apps/test          1/1     1            1           88s

NAME                  TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)                      AGE
service/elastic-ple   NodePort   10.68.248.204   <none>        8080:30003/TCP               27d
service/test          NodePort   10.68.170.24    <none>        80:30002/TCP,801:23137/TCP   88s

NAME                             CLASS    HOSTS            ADDRESS   PORTS   AGE
ingress.networking.k8s.io/test   <none>   test.baidu.com             80      84s
```


## 发布配置

1. 安装控制器 crd

```
kubectl apply -f deploy/qservice.yml
```

部署服务yaml文件如下

```yaml
apiVersion: apps.tech/v1beta1
kind: Qservice
metadata:
  name: example
  annotations:
    qservice/api: /example
spec:
  envs:
    SRV_EXAMPLE__Server_Debug: "False"
  image: nginx:1.21.1
  livenessProbe:
    action: http://:80/
    initialDelaySeconds: 5
    periodSeconds: 5
  readinessProbe:
    action: http://:80/
    initialDelaySeconds: 5
    periodSeconds: 5
  resources:
    limits:
      cpu: "1"
      memory: "100Mi"
    requests:
      cpu: "1"
      memory: "100Mi"
  ports:
  - 30002:80
  - "801"
  ingress:
  - domain: test.xx.com
    paths:
    - path: "/"
      port: 80
    - path: /api
      port: 801
  mount:
  - name: "test1"
    targetpath: /var/logs
    type: hostpath
    path: /var/log
    subpath: "redis"
  - name: "test"
    targetpath: /var/logs1
    type: pvc
    pvc: "local"
    subpath: "redis"
```

+ `resources`: （可选） 如果未配置会自动默认request limit cpu 500m memory 500Mi。
+ `mount`: （可选） 用于针对pod 进行挂载pvc localpath。
+ `Ingress`: （可选） 用于自动创建ingress 资源。
+ `ports`: （必须） 用于pod端口以及service端口 端口规则 需要外部端口访问 nodeport:port 无需外部端口访问 port即可。