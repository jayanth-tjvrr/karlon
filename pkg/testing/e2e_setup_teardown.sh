#!/usr/bin/env bash
function wait_until() {
  for _ in $(seq 1 "$3"); do
    eval "$1" && return 0
    echo "Waiting for \"$1\" to evaluate to true ..."
    sleep "$2"
  done
  echo Timed out waiting for \""$1"\"
  return 1
}


if [ -z "${GIT_USER}" ]; then
  echo "Set the GIT_USER env variable"
  exit
fi

if [ -z "${GIT_PASSWORD}" ]; then
  echo "Set the GIT_PASSWORD env variable"
  exit
fi

git_server_port=8188

if which karlon &>/dev/null; then
  karlon cluster delete ec2-cluster
  wait_until "set -o pipefail; karlon cluster list 2> /dev/null | grep -v ec2-cluster" 60 20
else
  echo "karlon not installed on PATH"
  echo "cluster probably not created. Skipping cluster delete..."
fi

rm -rf ~/.karlon ~/.config/karlon
workspace_repo="/tmp/karlon-testbed-git-clone/myrepo"

pushd "${workspace_repo}" || exit 1
git pull
rm -rf "$(ls)"
git commit -am "cleanup e2e-test manifests"
git push "http://${GIT_USER}:${GIT_PASSWORD}@localhost:${git_server_port}/${GIT_USER}/myrepo.git"
popd || exit 1

if [ -d ~/external ]; then
  echo "Deleting external cluster kubeconfig directory and external cluster"
  rm -rf ~/external
  kind delete cluster --name external1 || true
fi

kind delete cluster --name karlon-e2e-testbed || true
sudo rm -rf /tmp/karlon-testbed-git*
