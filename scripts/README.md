# Combinator K3s 部署指南

## 文件说明

- `k3s-deployment.yaml` - 完整的 K8s 部署配置

## 快速开始

### 0. 拉取 YAML

```bash
curl https://raw.githubusercontent.com/jabberwocky238/combinator/main/scripts/k3s-deployment.yaml -o combinator-k3s-deployment.yaml
```

curl -X POST "http://combinator.app238.com/rdb/query" -H "Content-Type: application/json" -H "X-Combinator-RDB-ID: 1" -d "{\"stmt\":\"select * from longlivecombinator\",\"args\":[]}"
curl -X POST "http://combinator.app238.com/rdb/query" -H "Content-Type: application/json" -H "X-Combinator-RDB-ID: 0" -d "{\"stmt\":\"select * from longlivecombinator\",\"args\":[]}"

### 1. 部署

```bash
kubectl apply -f combinator-k3s-deployment.yaml
```

镜像会自动从 GitHub Container Registry 拉取。

### 2. 查看状态

```bash
kubectl get all -n combinator
```

### 3. 端口转发（本地访问）

```bash
kubectl port-forward -n combinator svc/combinator 8899:8899
```

### 4. 测试

```bash
curl http://localhost:8899/health
```

### 5. 清理

```bash
kubectl delete -f combinator-k3s-deployment.yaml
```

## 部署架构

```
combinator namespace
├── PostgreSQL
│   ├── Deployment (1 replica)
│   ├── Service (ClusterIP)
│   ├── PVC (1Gi)
│   └── ConfigMap (初始化脚本)
├── Redis
│   ├── Deployment (1 replica)
│   ├── Service (ClusterIP)
│   └── PVC (500Mi)
└── Combinator
    ├── Deployment (2 replicas)
    ├── Service (ClusterIP)
    ├── ConfigMap (config.json)
    └── Ingress (可选)
```
