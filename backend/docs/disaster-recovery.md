# Disaster Recovery Plan

## 1. Overview

This document defines the backup strategy, recovery procedures, and runbooks for restoring the ecommerce platform after a disaster. It covers all system components: MongoDB (primary data store), NATS JetStream (event streaming), the Go application (stateless API and Worker), Kubernetes infrastructure, secrets, and DNS.

This plan is a **living document**. Every section marked with `[NOT TESTED]` represents a procedure that must be executed in a staging or dedicated DR environment before it can be considered reliable. Recovery procedures that have never been tested are assumptions, not capabilities.

---

## 2. Recovery Objectives

### 2.1 Recovery Point Objective (RPO)

The RPO defines the maximum acceptable data loss measured in time.

| Component | RPO | Basis |
|---|---|---|
| MongoDB | 5 minutes | Continuous oplog-based replication in a replica set. Point-in-time recovery via periodic snapshots + oplog replay. For deployments without a replica set, RPO equals the backup interval (hourly). |
| NATS JetStream | 0 (no data loss) | JetStream persists messages to disk before acknowledging. With consumer replay, messages are re-delivered after recovery. Unprocessed in-flight messages may be lost if the NATS server crashes during processing. |
| Application (API/Worker) | 0 | Stateless. No local data. All state is in MongoDB and NATS. Pod recreation from the container image is instantaneous. |
| Kubernetes manifests | 0 | Stored in Git. Recovery is a `kubectl apply -k k8s/`. No application state is only in-cluster. |
| Secrets | 0 | Stored in external secret manager (recommended) or backed up separately. Secrets exist in Git only as base64 placeholders — real values must be recoverable from the secret store. |
| DNS | 0-60 seconds | TTL of DNS records determines propagation delay after a DNS failover. |

### 2.2 Recovery Time Objective (RTO)

The RTO defines the maximum acceptable downtime.

| Component | RTO | Basis |
|---|---|---|
| MongoDB | 15 minutes | Restore from snapshot, replay oplog. Replica set automatic failover is near-instantaneous (typically under 10 seconds) for single-node failures within a replica set. Full restore from backup takes longer. |
| NATS JetStream | 5 minutes | Restart with existing storage volume. JetStream replays undelivered messages to consumers. No manual intervention needed if storage is intact. |
| Application (API/Worker) | 2 minutes | Kubernetes restarts pods automatically. New pods pass readiness probes once MongoDB and NATS are available. HPA scales up to meet demand. |
| Kubernetes cluster | 30-60 minutes | Assumes managed Kubernetes (EKS/GKE/AKS). Cluster rebuild from IaC (Terraform) plus `kubectl apply -k k8s/`. Self-managed clusters take longer. |
| Secrets | 10 minutes | Recreate from external secret manager. If using Sealed Secrets, re-encrypt and apply. |
| DNS | 1-5 minutes | Update DNS record (weighted routing or failover). Propagation depends on TTL (set to 60s for critical records). |

### 2.3 Overall System RTO/RPO

| Scenario | RTO | RPO |
|---|---|---|
| Single pod crash | Automatic (0-30s) | 0 |
| Single node failure | 2-5 minutes (K8s rescheduling) | 0 |
| MongoDB primary failure | 15 minutes (replica failover) or restore | 0-5 minutes (replica set) or 1 hour (backup) |
| NATS data loss | 5 minutes (restart from volume) | 0 (persisted streams) |
| Full cluster loss | 30-60 minutes | 1 hour (last backup) |
| Region failure | 2-4 hours | 1 hour (cross-region backup) |

---

## 3. Component Backup Strategies

### 3.1 MongoDB Backup and Restore

#### Backup Strategy

MongoDB 7.x supports multiple backup approaches. The strategy depends on deployment size and criticality.

**Primary: Filesystem Snapshot (recommended for single-node Docker and small K8s deployments)**

The Docker Compose deployment uses a local Docker volume (`mongodb_data`). Backups are created by snapshotting the volume's data directory while MongoDB is running, using `mongodump` for consistency.

```bash
# Backup — run on the host with MongoDB accessible
mongodump \
  --uri="mongodb://localhost:27017/ecommerce" \
  --out="/backup/mongodb/$(date +%Y%m%d-%H%M%S)" \
  --gzip

# Result: /backup/mongodb/20260122-143000/ecommerce/*.gz
```

**Secondary: Oplog-based Point-in-Time Recovery**

For deployments using a MongoDB replica set, the oplog enables recovery to any point in time within the oplog window (typically 24-72 hours depending on oplog size).

```bash
# Backup with oplog for PITR
mongodump \
  --uri="mongodb://localhost:27017/ecommerce" \
  --out="/backup/mongodb/$(date +%Y%m%d-%H%M%S)" \
  --gzip \
  --oplog
```

**Tertiary: Cloud Provider Snapshots (for managed MongoDB)**

If using MongoDB Atlas, AWS DocumentDB, or GCP Firestore, use the provider's automated snapshot feature. These are typically configured with daily snapshots and 7-day retention at no additional cost.

#### Backup Schedule

| Environment | Frequency | Retention | Method |
|---|---|---|---|
| Production | Hourly | 30 days | `mongodump --gzip` to object storage (S3/GCS) |
| Staging | Daily | 7 days | `mongodump --gzip` to local disk |
| Development | On-demand | 3 days | `mongodump` to local disk |

#### Restore Procedure

```bash
# Step 1: Verify the backup archive exists
ls -la /backup/mongodb/20260122-143000/ecommerce/

# Step 2: Stop the application (prevents writes during restore)
kubectl scale deployment ecommerce-api --replicas=0 -n ecommerce
kubectl scale deployment ecommerce-worker --replicas=0 -n ecommerce

# Step 3: Restore to MongoDB
mongorestore \
  --uri="mongodb://mongodb:27017/ecommerce" \
  --drop \
  --gzip \
  "/backup/mongodb/20260122-143000/ecommerce/"

# Step 4: Verify data integrity
mongosh "mongodb://mongodb:27017/ecommerce" --eval '
  db = db.getSiblingDB("ecommerce");
  print("Collections: " + db.getCollectionNames().join(", "));
  print("Users count: " + db.users.countDocuments({}));
  print("Products count: " + db.products.countDocuments({}));
  print("Orders count: " + db.orders.countDocuments({}));
'

# Step 5: Restart the application
kubectl scale deployment ecommerce-api --replicas=3 -n ecommerce
kubectl scale deployment ecommerce-worker --replicas=2 -n ecommerce

# Step 6: Verify application readiness
kubectl rollout status deployment/ecommerce-api -n ecommerce --timeout=120s
curl -sf https://ecommerce.example.com/health/ready | jq .
```

**Docker Compose Restore:**

```bash
# Stop application containers
docker compose stop api worker

# Restore into the running MongoDB container
docker compose exec -T mongodb mongorestore \
  --drop --gzip \
  /backup/mongodb/20260122-143000/ecommerce/

# Restart application
docker compose start api worker
```

---

### 3.2 NATS JetStream Persistence and Recovery

#### How JetStream Persists Data

NATS JetStream stores messages on disk in the `--store_dir` directory (mapped to the `nats_data` Docker volume or a K8s PersistentVolume). Streams retain messages based on their retention policy (limits-based, time-based, or interest-based). The ecommerce platform uses JetStream for order processing, inventory updates, payment events, shipping updates, notification delivery, and analytics tracking.

#### Backup Strategy

**JetStream storage is the backup.** Unlike MongoDB, JetStream does not need a separate `dump` tool. The storage directory contains all stream data, consumer state, and metadata. The backup strategy is to copy the storage directory.

```bash
# Backup NATS JetStream storage (Docker volume)
docker run --rm \
  -v ecommerce_nats_data:/source:ro \
  -v $(pwd)/backup:/backup \
  alpine tar czf "/backup/nats-jetstream/$(date +%Y%m%d-%H%M%S).tar.gz" -C /source .

# Kubernetes: back up the PersistentVolume
kubectl get pvc -n ecommerce -l app.kubernetes.io/name=nats
# Use Velero or the cloud provider's volume snapshot to back up the PV
```

**Stream state backup (metadata only):**

```bash
# Export stream configurations (useful for recreating streams)
nats stream list --server=nats://localhost:4222 2>/dev/null || \
  curl -s http://localhost:8222/jsz | jq '.streams[] | {name, config}'
```

#### Restore Procedure

```bash
# Step 1: Stop NATS consumers (workers) to prevent replay conflicts
kubectl scale deployment ecommerce-worker --replicas=0 -n ecommerce

# Step 2: Stop NATS server
docker compose stop nats
# Or in K8s: scale the NATS StatefulSet to 0

# Step 3: Restore the JetStream storage directory
# Docker:
docker run --rm \
  -v ecommerce_nats_data:/target \
  -v $(pwd)/backup:/backup \
  alpine sh -c "rm -rf /target/* && tar xzf /backup/nats-jetstream/20260122-143000.tar.gz -C /target"

# Step 4: Start NATS server
docker compose start nats
# Wait for health check
until docker compose exec -T nats /nats-server --js --healthcheck --port 8222 2>/dev/null; do
  echo "Waiting for NATS..."
  sleep 2
done

# Step 5: Verify streams are loaded
nats stream list --server=nats://localhost:4222

# Step 6: Restart workers (they will reconnect and resume consuming)
kubectl scale deployment ecommerce-worker --replicas=2 -n ecommerce
```

**Important:** JetStream consumers track their position via the consumer state stored in the JetStream storage. If the storage is restored, consumers will resume from their last acknowledged position. If the storage is lost, consumers will start from the beginning of the stream (or the stream's retention window), which may cause duplicate message processing. The application must be idempotent to handle this.

---

### 3.3 Application Recovery

#### Why Application Recovery Is Trivial

The API and Worker are **stateless**. They hold no data locally. All persistent state is in MongoDB and NATS JetStream. The container images are stored in GHCR (or your container registry) and are immutable. Recovery is simply restarting the pods.

#### Automatic Recovery (No Intervention Needed)

Kubernetes handles the following automatically:

- **Pod crash**: Kubernetes restarts the crashed container via the `restartPolicy: Always`.
- **Node failure**: Kubernetes reschedules pods to healthy nodes. PDB ensures minimum availability during the rescheduling.
- **OOM kill**: Kubernetes restarts the container. The 512Mi memory limit prevents cascading node failures.
- **Unhealthy probes**: Liveness probe failure triggers a pod restart. Readiness probe failure removes the pod from the Service endpoint until it recovers.

#### Manual Recovery (If Automatic Recovery Fails)

```bash
# Check pod status and recent events
kubectl get pods -n ecommerce -o wide
kubectl describe pod <pod-name> -n ecommerce
kubectl logs <pod-name> -n ecommerce --previous  # logs from crashed container

# Force restart all API pods
kubectl rollout restart deployment/ecommerce-api -n ecommerce
kubectl rollout status deployment/ecommerce-api -n ecommerce --timeout=120s

# Force restart all worker pods
kubectl rollout restart deployment/ecommerce-worker -n ecommerce

# If pods are stuck in CrashLoopBackOff
kubectl delete pods -l app.kubernetes.io/component=api -n ecommerce
kubectl delete pods -l app.kubernetes.io/component=worker -n ecommerce

# If the image is missing or corrupted
kubectl set image deployment/ecommerce-api \
  api=ghcr.io/yourorg/ecommerce-api:<known-good-version> -n ecommerce
```

#### Docker Compose Recovery

```bash
# Restart everything
docker compose restart

# Rebuild and restart (if the image changed)
docker compose up -d --build

# View logs for diagnosis
docker compose logs --tail=100 -f api worker
```

---

### 3.4 Kubernetes Recovery

#### What Needs to Be Recovered

Kubernetes resources for this project are defined declaratively in `k8s/` and applied via `kubectl apply -k k8s/`. The following in-cluster resources are NOT in Git and must be backed up separately:

| Resource | Location | Backup Method |
|---|---|---|
| Namespace, Deployments, Services, etc. | `k8s/` directory in Git | Version controlled. Recover with `kubectl apply -k k8s/`. |
| Secrets (real values) | External secret manager or manual backup | Export via `kubectl get secret ecommerce-secret -o yaml` (encrypt before storing). |
| PersistentVolumes (MongoDB, NATS) | Cloud provider disks / local volumes | Velero, cloud snapshots, or `mongodump`/`nats-backup`. |
| Ingress TLS certificates | cert-manager or manual | cert-manager re-issues automatically. Manual certs need backup of private keys. |
| HPA/eviction events | Ephemeral | No backup needed. K8s reconstructs from current state. |

#### Full Cluster Recovery Procedure

```bash
# Step 1: Rebuild the Kubernetes cluster (if the entire cluster is lost)
# This is IaC-specific (Terraform, Pulumi, cloud console).
# For managed K8s (EKS/GKE/AKS), recreate the cluster, then continue below.

# Step 2: Recreate the namespace
kubectl apply -f k8s/namespace.yaml

# Step 3: Recreate service accounts
kubectl apply -f k8s/serviceaccount.yaml

# Step 4: Recreate ConfigMap (non-secret config)
kubectl apply -f k8s/configmap.yaml

# Step 5: Restore secrets (see Section 3.5)
# kubectl apply -f /backup/secrets/ecommerce-secret-restored.yaml

# Step 6: Restore PersistentVolumes
# If using cloud PV snapshots:
#   kubectl apply -f /backup/pv-restore.yaml
# If using Velero:
#   velero restore create --from-backup ecommerce-daily-20260122

# Step 7: Deploy the workloads
kubectl apply -f k8s/api-deployment.yaml
kubectl apply -f k8s/worker-deployment.yaml
kubectl apply -f k8s/api-service.yaml
kubectl apply -f k8s/worker-service.yaml
kubectl apply -f k8s/hpa.yaml
kubectl apply -f k8s/pdb.yaml
kubectl apply -f k8s/networkpolicy.yaml
kubectl apply -f k8s/ingress.yaml

# Or simply:
# kubectl apply -k k8s/

# Step 8: Wait for all pods to become ready
kubectl get pods -n ecommerce -w
kubectl rollout status deployment/ecommerce-api -n ecommerce --timeout=300s
kubectl rollout status deployment/ecommerce-worker -n ecommerce --timeout=300s

# Step 9: Verify health
curl -sk https://ecommerce.example.com/health/ready | jq .
```

#### Recommended: Velero for Kubernetes Backup

For production, use [Velero](https://velero.io/) to back up the entire namespace including PVs:

```bash
# Install Velero (AWS S3 example)
velero install \
  --provider aws \
  --bucket ecommerce-velero-backups \
  --secret-file ./credentials-velero \
  --backup-location-config region=us-east-1 \
  --plugins velero/velero-plugin-for-aws:v1.10.0

# Create a daily backup schedule
velero schedule create ecommerce-daily \
  --include-namespaces ecommerce \
  --ttl 720h  # 30 days

# On-demand backup before a risky change
velero backup create ecommerce-pre-migration --include-namespaces ecommerce

# Restore from a backup
velero restore create --from-backup ecommerce-daily-20260122
```

---

### 3.5 Secret Recovery

#### Current Secret Inventory

| Secret Name | Keys | Source | Criticality |
|---|---|---|---|
| `ecommerce-secret` | `jwt-secret`, `mongodb-uri`, `nats-url`, `webhook-secret-fake` | External secret manager or manual creation | Critical — application cannot start without these |
| Ingress TLS | `tls.crt`, `tls.key` | cert-manager or manual | High — HTTPS will fail without these |

#### Backup Procedure

**Option A: External Secret Manager (recommended for production)**

If using AWS Secrets Manager, GCP Secret Manager, HashiCorp Vault, or External Secrets Operator:

1. Secrets are already stored externally. No separate backup needed.
2. The External Secrets Operator synchronizes them into the cluster.
3. Recovery: reinstall the External Secrets Operator and re-create the `ExternalSecret` resources.

**Option B: Manual Backup**

```bash
# Export secrets (base64-encoded)
kubectl get secret ecommerce-secret -n ecommerce -o yaml > /backup/secrets/ecommerce-secret-$(date +%Y%m%d).yaml

# IMPORTANT: The exported file contains base64-encoded secrets.
# Encrypt this file before storing it.
# Using age (https://github.com/FiloSottile/age):
age -r <your-public-key> -o /backup/secrets/ecommerce-secret-$(date +%Y%m%d).yaml.age \
  /backup/secrets/ecommerce-secret-$(date +%Y%m%d).yaml

# Store the encrypted file in a secure location (S3, GCS, offline)
```

#### Restore Procedure

```bash
# Decrypt the backup file
age -d -i <your-private-key-file> \
  /backup/secrets/ecommerce-secret-20260122.yaml.age \
  > /tmp/ecommerce-secret-restored.yaml

# Remove the old resource metadata to avoid conflicts
kubectl delete secret ecommerce-secret -n ecommerce --ignore-not-found

# Apply the restored secret
kubectl apply -f /tmp/ecommerce-secret-restored.yaml

# Verify
kubectl get secret ecommerce-secret -n ecommerce -o jsonpath='{.data.jwt-secret}' | base64 -d | head -c 8
echo "... (truncated for security)"

# Securely delete the decrypted file
shred -u /tmp/ecommerce-secret-restored.yaml
```

#### Secret Rotation

After a disaster recovery, rotate all secrets as a precaution:

1. Generate a new JWT secret (32+ character random string)
2. Rotate MongoDB user password
3. Rotate NATS user credentials
4. Rotate webhook HMAC secrets
5. Update the external secret manager
6. Restart API and Worker pods to pick up new secrets

---

### 3.6 DNS Recovery

#### DNS Configuration

| Record | Type | Value | TTL |
|---|---|---|---|
| `ecommerce.example.com` | CNAME or A | Load balancer / Ingress controller IP | 60s |
| `api.ecommerce.example.com` (optional) | CNAME | Same as above | 60s |

#### Failure Scenarios

**Scenario 1: DNS provider outage**

- If using Route 53, Cloud DNS, or Cloudflare, these providers have their own DR (multi-region anycast).
- Mitigation: maintain a secondary DNS provider as a hot standby. Use NS records to delegate to a secondary if the primary fails.

**Scenario 2: Incorrect DNS change**

- Revert the DNS record to the previous value. DNS propagation takes up to the TTL (60 seconds for our configuration).
- Keep a record of the last known-good DNS configuration in a secure location.

**Scenario 3: DNS record deleted**

- Recreate the record from the documented values in this section.
- For automated DNS management, the DNS records should be defined in Terraform or equivalent IaC.

```bash
# Using Route 53 CLI (example)
aws route53 change-resource-record-sets \
  --hosted-zone-id Z1234567890ABC \
  --change-batch '{
    "Changes": [{
      "Action": "UPSERT",
      "ResourceRecordSet": {
        "Name": "ecommerce.example.com",
        "Type": "CNAME",
        "TTL": 60,
        "ResourceRecords": [{"Value": "k8s-ingress-abcd1234.us-east-1.elb.amazonaws.com"}]
      }
    }]
  }'
```

---

## 4. Failure Scenarios and Runbooks

### 4.1 Scenario: MongoDB Data Corruption

**Symptoms:** Application returns 500 errors, readiness probe fails with MongoDB check, log entries show wire protocol errors or document validation failures.

**Runbook:**

1. Isolate: scale API and Worker to 0 to prevent further corruption.
2. Diagnose: connect to MongoDB and run `db.runCommand({dbCheck: 1})` on each database and collection.
3. If corruption is limited to a collection, drop and restore just that collection from backup.
4. If corruption is widespread, perform a full restore (see Section 3.1).
5. Verify data integrity counts against known baselines.
6. Scale application back up.

### 4.2 Scenario: MongoDB Primary Failover (Replica Set)

**Symptoms:** Brief write errors (a few seconds), then automatic recovery.

**Runbook:**

1. No action needed for transient failovers. MongoDB replica sets elect a new primary automatically (typically under 10 seconds).
2. If the application does not recover after 30 seconds, check the MongoDB connection string includes all replica set members.
3. Verify: `kubectl logs -l app.kubernetes.io/component=api -n ecommerce | grep -i "mongo"`.
4. If persistent, check the replica set status: `rs.status()` in `mongosh`.

### 4.3 Scenario: NATS Server Crash with Data Loss

**Symptoms:** Workers stop processing messages, NATS health check fails, JetStream streams show reduced message counts.

**Runbook:**

1. Check if the NATS storage volume is intact.
2. If the volume is intact, restart NATS. JetStream will replay messages from the storage.
3. If the volume is lost, restore from the JetStream storage backup (Section 3.2).
4. After restore, workers will reconnect and resume from their last acknowledged position.
5. If consumer state was lost, workers may reprocess some messages. Verify order processing integrity by checking order IDs against the orders collection.
6. Monitor the NATS monitoring endpoint (`http://nats:8222/jsz`) for stream and consumer health.

### 4.4 Scenario: Accidental Secret Deletion

**Symptoms:** Pods fail to start, events show `secret "ecommerce-secret" not found`, application crashes on startup with missing configuration.

**Runbook:**

1. Do NOT restart pods repeatedly — each restart attempt logs the missing secret name, which is benign but noisy.
2. Restore the secret from the encrypted backup (Section 3.5).
3. If no backup exists, regenerate all secrets (Section 3.5 Secret Rotation).
4. After secret restoration, pods will need to be restarted to pick up the new secret (K8s does not automatically inject updated secrets into running pods):
   ```bash
   kubectl rollout restart deployment/ecommerce-api -n ecommerce
   kubectl rollout restart deployment/ecommerce-worker -n ecommerce
   ```

### 4.5 Scenario: Kubernetes Node Failure

**Symptoms:** Pods in `Unknown` or `Terminating` state, reduced replica count, PDB preventing node drain.

**Runbook:**

1. Check node status: `kubectl get nodes`
2. For a single node failure, K8s automatically reschedules pods to healthy nodes. Verify with `kubectl get pods -n ecommerce -o wide -w`.
3. If the node is `NotReady` for more than 5 minutes, investigate: `kubectl describe node <node-name>`.
4. If the node is permanently lost, delete it from the cluster: `kubectl delete node <node-name>`.
5. Verify all pods are rescheduled and running: `kubectl get pods -n ecommerce`

### 4.6 Scenario: Full Region/Cluster Loss

**Symptoms:** Complete loss of access to the Kubernetes API, all services unreachable.

**Runbook:**

1. Follow the full cluster recovery procedure (Section 3.4).
2. If the IaC (Terraform) state is also lost, rebuild the infrastructure from the version-controlled Terraform code.
3. Restore MongoDB from the most recent cross-region backup (Section 3.1).
4. Restore NATS JetStream from backup (Section 3.2).
5. Restore secrets from the external secret manager (Section 3.5).
6. Update DNS to point to the new cluster's Ingress controller (Section 3.6).
7. Verify end-to-end: place a test order and confirm it is processed by the worker.

---

## 5. Recovery Verification Checklist

After any recovery procedure, complete this checklist before declaring the system recovered. Do not skip any step. Every item must pass.

### 5.1 Infrastructure Verification

- [ ] MongoDB is running and accepting connections
- [ ] `db.adminCommand('ping')` returns `{ ok: 1 }`
- [ ] MongoDB replica set status is healthy (if applicable)
- [ ] NATS is running and JetStream is enabled
- [ ] `curl -sf http://nats:8222/healthz` returns 200
- [ ] All expected JetStream streams exist
- [ ] All JetStream consumers are connected and not lagging

### 5.2 Application Verification

- [ ] API pods are running: `kubectl get pods -l app.kubernetes.io/component=api -n ecommerce`
- [ ] Worker pods are running: `kubectl get pods -l app.kubernetes.io/component=worker -n ecommerce`
- [ ] All pods pass readiness probes
- [ ] API responds to health check: `curl -sf https://ecommerce.example.com/health/ready | jq .` returns `"status": "ready"`
- [ ] All MongoDB/NATS/JetStream/KV health checks in readiness response report OK
- [ ] Prometheus metrics endpoint is accessible: `curl -sf http://api:9090/metrics` returns text

### 5.3 Data Verification

- [ ] User count matches expected baseline (query `db.users.countDocuments({})`)
- [ ] Product catalog is intact (query `db.products.countDocuments({})`)
- [ ] Recent orders exist and have correct status
- [ ] No duplicate orders were created during recovery (check by order ID uniqueness)
- [ ] Inventory counts match expected values
- [ ] Audit log entries exist for the recovery period

### 5.4 Functional Verification

- [ ] User can log in and receive a valid JWT
- [ ] Product listing returns results
- [ ] Product search returns results
- [ ] A test order can be created via `POST /orders`
- [ ] The test order appears in the admin order list
- [ ] The worker processes the test order (status transitions from pending to confirmed)
- [ ] Health and readiness endpoints return correct status

### 5.5 Security Verification

- [ ] Secrets are not exposed in pod environment (verify with `kubectl exec`)
- [ ] NetworkPolicies are in effect (verify with a connectivity test from an unauthorized pod)
- [ ] TLS certificate is valid and not expired
- [ ] No pods are running with unexpected privileges

### 5.6 Monitoring Verification

- [ ] Prometheus is scraping the API metrics endpoint
- [ ] No alert fires are active (check alertmanager)
- [ ] Log aggregation is receiving logs from all pods
- [ ] Dashboard shows normal traffic patterns

---

## 6. Backup Automation

### 6.1 Cron Job for MongoDB Backups

```bash
#!/bin/bash
# /opt/backup/backup-mongodb.sh
# Run via cron: 0 * * * * /opt/backup/backup-mongodb.sh
set -euo pipefail

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_DIR="/backup/mongodb/${TIMESTAMP}"
RETENTION_DAYS=30

mkdir -p "${BACKUP_DIR}"

mongodump \
  --uri="${MONGODB_URI:-mongodb://localhost:27017/ecommerce}" \
  --out="${BACKUP_DIR}" \
  --gzip

# Upload to object storage (AWS S3 example)
if command -v aws &>/dev/null; then
  aws s3 cp "${BACKUP_DIR}" "s3://ecommerce-backups/mongodb/${TIMESTAMP}/" --recursive
fi

# Clean up local backups older than retention period
find /backup/mongodb -mindepth 1 -maxdepth 1 -type d -mtime +${RETENTION_DAYS} -exec rm -rf {} +

echo "[$(date -u)] MongoDB backup complete: ${TIMESTAMP}"
```

### 6.2 Cron Job for NATS Storage Backups

```bash
#!/bin/bash
# /opt/backup/backup-nats.sh
# Run via cron: 0 */6 * * * /opt/backup/backup-nats.sh
set -euo pipefail

TIMESTAMP=$(date +%Y%m%d-%H%M%S)
BACKUP_FILE="/backup/nats-jetstream/${TIMESTAMP}.tar.gz"
RETENTION_DAYS=14

mkdir -p /backup/nats-jetstream

# Docker volume backup
docker run --rm \
  -v ecommerce_nats_data:/source:ro \
  -v /backup:/backup \
  alpine tar czf "${BACKUP_FILE}" -C /source .

# Upload to object storage
if command -v aws &>/dev/null; then
  aws s3 cp "${BACKUP_FILE}" "s3://ecommerce-backups/nats/${TIMESTAMP}.tar.gz"
fi

# Clean up
find /backup/nats-jetstream -name '*.tar.gz' -mtime +${RETENTION_DAYS} -delete

echo "[$(date -u)] NATS JetStream backup complete: ${TIMESTAMP}"
```

### 6.3 Kubernetes Secret Backup

```bash
#!/bin/bash
# /opt/backup/backup-secrets.sh
# Run via cron: 0 0 * * * /opt/backup/backup-secrets.sh
set -euo pipefail

TIMESTAMP=$(date +%Y%m%d)
BACKUP_DIR="/backup/secrets"
ENCRYPTED_FILE="${BACKUP_DIR}/ecommerce-secret-${TIMESTAMP}.yaml.age"

mkdir -p "${BACKUP_DIR}"

# Export secret
kubectl get secret ecommerce-secret -n ecommerce -o yaml > "/tmp/sec-${TIMESTAMP}.yaml"

# Encrypt with age
age -r <your-public-key> -o "${ENCRYPTED_FILE}" "/tmp/sec-${TIMESTAMP}.yaml"
rm -f "/tmp/sec-${TIMESTAMP}.yaml"

# Upload to object storage
if command -v aws &>/dev/null; then
  aws s3 cp "${ENCRYPTED_FILE}" "s3://ecommerce-backups/secrets/"
fi

# Keep only the last 7 daily backups
ls -t "${BACKUP_DIR}"/ecommerce-secret-*.yaml.age | tail -n +8 | xargs rm -f

echo "[$(date -u)] Secret backup complete: ${TIMESTAMP}"
```

---

## 7. DR Testing Schedule

Disaster recovery procedures that are never tested are not recovery procedures — they are hopes. The following testing schedule must be maintained.

| Test Type | Frequency | Scope | Notes |
|---|---|---|---|
| Backup verification | Weekly | Verify backup files exist and are non-empty in object storage | Automated: check S3 object size > 0 |
| MongoDB restore test | Monthly | Restore latest backup to a staging MongoDB instance, verify collection counts | [NOT TESTED] |
| NATS restore test | Monthly | Restore JetStream storage, verify streams and consumers recover | [NOT TESTED] |
| Secret restore test | Monthly | Decrypt and verify secret backup, confirm values match current secrets | [NOT TESTED] |
| Full DR drill | Quarterly | Simulate cluster loss, recover from scratch, run full verification checklist | [NOT TESTED] |
| DNS failover test | Semi-annually | Update DNS to point to DR environment, verify traffic routes correctly | [NOT TESTED] |

Every test must produce a dated report documenting: what was tested, what succeeded, what failed, and what changed as a result. Store these reports alongside this document.

---

## 8. Contact and Escalation

| Role | Responsibility | Escalation Path |
|---|---|---|
| On-call engineer | First responder, execute runbooks | Page platform lead if unresolved in 30 minutes |
| Platform lead | Coordinate recovery, communicate status | Page engineering manager if unresolved in 1 hour |
| Engineering manager | Resource allocation, stakeholder communication | Executive notification for extended outages (> 4 hours) |

---

## 9. Appendix: Quick Reference Commands

```bash
# --- MongoDB ---
mongosh "mongodb://localhost:27017/ecommerce" --eval 'db.runCommand({dbCheck:1})'
mongodump --uri="mongodb://localhost:27017/ecommerce" --gzip --out=/tmp/backup
mongorestore --uri="mongodb://localhost:27017/ecommerce" --drop --gzip /tmp/backup/ecommerce

# --- NATS ---
curl -sf http://localhost:8222/healthz
curl -sf http://localhost:8222/jsz | jq '.streams[] | {name, state: .state.messages}'
nats stream ls 2>/dev/null || curl -sf http://localhost:8222/jsz | jq '.streams[].name'

# --- Kubernetes ---
kubectl get all -n ecommerce
kubectl get pods -n ecommerce -o wide
kubectl describe pod <name> -n ecommerce
kubectl logs <name> -n ecommerce --previous
kubectl rollout restart deployment/ecommerce-api -n ecommerce
kubectl rollout status deployment/ecommerce-api -n ecommerce --timeout=120s
kubectl get events -n ecommerce --sort-by='.lastTimestamp'

# --- Application ---
curl -sf https://ecommerce.example.com/health/ready | jq .
curl -sf https://ecommerce.example.com/health/live
curl -sf http://localhost:9090/metrics | head -20
```