#!/usr/bin/env bash

set -o errexit
set -o nounset
set -o pipefail

SCRIPT_ROOT=$(dirname "${BASH_SOURCE[0]}")/..
CODEGEN_PKG=${CODEGEN_PKG:-$(cd "${SCRIPT_ROOT}"; ls -d -1 ./vendor/k8s.io/code-generator 2>/dev/null || echo ../code-generator)}

# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
echo ${SCRIPT_ROOT}
bash "${CODEGEN_PKG}"/generate-groups.sh "client,lister,informer" \
  github.com/versori-oss/nats-account-operator/pkg/generated github.com/versori-oss/nats-account-operator/api \
  accounts:v1alpha1 \
  --go-header-file "${SCRIPT_ROOT}"/hack/boilerplate.go.txt \
  --output-base "${SCRIPT_ROOT}/generated" && \
rm -rf ${SCRIPT_ROOT}/pkg/generated && \
mv -v ${SCRIPT_ROOT}/generated/github.com/versori-oss/nats-account-operator/pkg/generated ${SCRIPT_ROOT}/pkg/generated/ && \
rm -rf ${SCRIPT_ROOT}/generated