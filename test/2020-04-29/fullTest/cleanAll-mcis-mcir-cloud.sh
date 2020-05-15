#!/bin/bash
source ../conf.env
source ../credentials.conf

echo "####################################################################"
echo "## Remove MCIS test to Zero Base"
echo "####################################################################"

CSP=${1}
POSTFIX=${2:-developer}
if [ "${CSP}" == "aws" ]; then
	echo "[Test for AWS]"
	INDEX=1
elif [ "${CSP}" == "azure" ]; then
	echo "[Test for Azure]"
	INDEX=2
elif [ "${CSP}" == "gcp" ]; then
	echo "[Test for GCP]"
	INDEX=3
else
	echo "[No acceptable argument was provided (aws, azure, gcp, ..). Default: Test for AWS]"
	CSP="aws"
	INDEX=1
fi

../6.mcis/just-terminate-mcis.sh $CSP $POSTFIX
echo "============== sleep 60 before delete MCIS obj"
sleep 60
../6.mcis/status-mcis.sh $CSP $POSTFIX
../6.mcis/terminate-and-delete-mcis.sh $CSP $POSTFIX
../5.spec/unregister-spec.sh $CSP $POSTFIX
../4.image/unregister-image.sh $CSP $POSTFIX
../3.sshKey/delete-sshKey.sh $CSP $POSTFIX
sleep 10
../2.securityGroup/delete-securityGroup.sh $CSP $POSTFIX
sleep 10
../1.vNet/delete-vNet.sh $CSP $POSTFIX
#../0.settingTB/delete-ns.sh $CSP $POSTFIX
../0.settingSpider/unregister-cloud.sh $CSP $POSTFIX


_self="${0##*/}"

echo ""
echo "[Cleaning related commands in history file executionStatus]"
sed -i "/${CSP} ${POSTFIX}/d" ./executionStatus
echo ""
echo "[Executed Command List]"
cat  ./executionStatus

