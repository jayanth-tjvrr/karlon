
kind delete cluster --name kind-karlon-testbed || true
docker stop karlon-testbed-gitserver || true
sudo rm -rf /tmp/karlon-testbed-git*
