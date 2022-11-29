#!/bin/bash
    

OUTPUTFILE="test-manifests.yaml"

function createNS {
  ns="$1"
  cat <<EOF >> $OUTPUTFILE 
apiVersion: v1
kind: Namespace
metadata:
  labels:
    operatorframework.io/kubectloperator: test
  name: $ns
---
EOF
}

function createOG {
  ns="$1"
  targets="$2"
  
  createNS $ns

cat <<EOF >> $OUTPUTFILE
apiVersion: operators.coreos.com/v1
kind: OperatorGroup
metadata:
  name: og
  namespace: $ns
  labels:
    operatorframework.coreos.com/kubectloperator: test
EOF
  if [ -n "$targets" ];then
    echo "spec:">> $OUTPUTFILE
    echo "  targetNamespaces:">> $OUTPUTFILE
    for i in ${targets//,/ }
    do
        echo "  - $i" >> $OUTPUTFILE
    done
  fi
  echo "---" >> $OUTPUTFILE

}



# 3 Levels to test
# o[name]==NS with OG
# o[name]s== NS with Self.
# n[name]==NS w/o OG (operand namespace)
#
# Good ones....
# oas
# obs --> ob1
# oc  --> oc1
#     --> oc2
# od --> od1 --> od11
#            --> od12
#    --> od2 --> od21
#            --> od22
# oe --> ne1
# of --> nf1
#    --> nf2
# og --> og1 --> ng11
#            --> ng12
#    --> og2 --> ng21
#            --> ng22
# oh --> oh1 --> nh11
# oh ----------> nh11
#
# Bad ones...
# oaa --> oaa1
# obb --> oaa1
# occ --> occ1 --> occ11
# odd --> odd1 --> odd11
#     --> odd2 --> occ11     

# Cleanup:
# kubectl delete ns -l operator-framework.coreos.com/kubectloperator=test

rm $OUTPUTFILE

createOG oa oa

createOG ob "ob,ob1"
createOG ob1 ob1

createOG oc "oc1,oc2"
createOG oc1 oc1
createOG oc2 oc2

createOG od "od,od1,od2"
createOG od1 "od1,od11,od12"
createOG od11 od11
createOG od12 od12
createOG od2 "od2,od21,od22"
createOG od21 od21
createOG od22 od22

createOG oe ne1
createNS ne1

createOG of "nf1,nf2"
createNS nf1
createNS nf2

createOG og "og1,og2"
createOG og1 "ng11,ng12"
createNS ng11
createNS ng12
createOG og2 "ng21,ng22"
createNS ng21
createNS ng22

createOG oh "oh1,th11"
createOG oh1 nh11
createNS nh11

# Error/Bad Topologies
createOG oaa oaa1
createOG oaa1 oaa1
createOG obb oaa1  # Intersection

createOG occ occ1
createOG occ1 occ11
createOG occ11 occ11
createOG odd odd1
createOG odd1 odd11
createOG odd2 occ11 # Intersection

