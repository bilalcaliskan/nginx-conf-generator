#!/bin/bash

# create namespace
kubectl create namespace test 2> /dev/null || echo "namespace test already exists"

# create rest of resources
kubectl create deployment nginx-a -n test --image=nginx --replicas=3 2> /dev/null || echo "deployment nginx-a already exists"
kubectl expose deployment nginx-a -n test --type NodePort --port 80 2> /dev/null || echo "service nginx-a already exposed"
kubectl annotate svc nginx-a -n test nginx-conf-generator/enabled='true' 2> /dev/null || echo "service nginx-a already annotated"

kubectl create deployment nginx-b -n test --image=nginx --replicas=3 2> /dev/null || echo "deployment nginx-b already exists"
kubectl expose deployment nginx-b -n test --type NodePort --port 80 2> /dev/null || echo "service nginx-b already exposed"
kubectl annotate svc nginx-b -n test nginx-conf-generator/enabled='true' 2> /dev/null || echo "service nginx-b already annotated"

kubectl create deployment nginx-c -n test --image=nginx --replicas=3 2> /dev/null || echo "deployment nginx-c already exists"
kubectl expose deployment nginx-c -n test --type NodePort --port 80 2> /dev/null || echo "service nginx-c already exposed"
kubectl annotate svc nginx-c -n test nginx-conf-generator/enabled='true' 2> /dev/null || echo "service nginx-c already annotated"
