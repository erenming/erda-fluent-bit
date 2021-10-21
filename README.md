# fluent-bit integrated in erda

export OUTPUT_ERDA_URL=http://localhost:1234
export OUTPUT_ERDA_AUTH_KEY=abc
export MASTER_VIP_URL=https://xxx.com


time="2021-10-21 07:52:20" level=info msg="[out_erda] INFO some error happened when enrich metadata: cannot find container with cid d137185d67f001562b2603ac3d45b4ddc813a02f2cf3580880d651df5d371c1a, podInfo: {id:c49ae207-8dfc-460f-acba-e2e546dd4a32 podName:fluent-bit-custom-7kbf2 podNamespace:logging podIP:172.17.0.3 containers:map[137185d67f001562b2603ac3d45b4ddc813a02f2cf3580880d651df5d371c1a:{id:137185d67f001562b2603ac3d45b4ddc813a02f2cf3580880d651df5d371c1a image:fluentbit-local:dev14 name:fluent-bit envMap:map[DICE_CLUSTER_NAME:terminus-xxx FLUENT_ELASTICSEARCH_HOST:elasticsesrssch FLUENT_ELASTICSEARCH_PORT:9200 MASTER_VIP_URL:https://10.96.0.1:443 NODE_NAME:minikube OUTPUT_ERDA_INGEST_URL:http://host.minikube.internal:7076]}]}"
