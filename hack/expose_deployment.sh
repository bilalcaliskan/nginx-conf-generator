#!/bin/bash

kubectl expose deployment nginx-g -n test --type NodePort --port 80
kubectl annotate svc nginx-g -n test nginx-conf-generator/enabled='true'

kubectl expose deployment nginx-h -n test --type NodePort --port 80
kubectl annotate svc nginx-h -n test nginx-conf-generator/enabled='true'
