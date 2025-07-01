karlon cluster ngupdate ec2-cluster --profile dynamic-2
clusterctl get kubeconfig ec2-cluster-capi-quickstart -n ec2-cluster > karlon-eks.kubeconfig #Generated the kubeconfig for ec2-cluster
cp karlon-eks.kubeconfig /home/runner/work/karlon/karlon/kubeconfig #placing the ec2-cluster config file in $KUBECONFIG
cp ~/.kube/config ~/.kube/temp.config #Swapping the ec2-cluster config with ~/.kube/config for assert.yaml
cp karlon-eks.kubeconfig ~/.kube/config