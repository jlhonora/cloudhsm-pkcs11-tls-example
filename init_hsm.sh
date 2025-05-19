#!/usr/bin/env bash

source env.sh

openssl genrsa -aes256 -out customerCA.key 2048
openssl req -new -x509 -days 3652 -key customerCA.key -out customerCA.crt

aws cloudhsmv2 describe-clusters --filters clusterIds=$CLUSTER_ID \
                                   --output text \
                                   --query 'Clusters[].Certificates.ClusterCsr' \
                                   > $CLUSTER_ID\_ClusterCsr.csr


openssl x509 -req -days 3652 -in $CLUSTER_ID\_ClusterCsr.csr \
                              -CA customerCA.crt \
                              -CAkey customerCA.key \
                              -CAcreateserial \
                              -out signedCert.crt

aws cloudhsmv2 initialize-cluster --cluster-id $CLUSTER_ID \
                                  --signed-cert file://signedCert.crt \
                                  --trust-anchor file://customerCA.crt
