#!/usr/bin/env bash

set -e

# CloudHSM Client
curl https://s3.amazonaws.com/cloudhsmv2-software/CloudHsmClient/Noble/cloudhsm-cli_latest_u24.04_amd64.deb -o cloudhsm-cli.deb
sudo apt install ./cloudhsm-cli.deb
rm cloudhsm-cli.deb

# AWS CLI
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

rm awscliv2.zip

# CloudHSM PKCS#11
wget https://s3.amazonaws.com/cloudhsmv2-software/CloudHsmClient/Noble/cloudhsm-pkcs11_latest_u24.04_amd64.deb
sudo apt install ./cloudhsm-pkcs11_latest_u24.04_amd64.deb

rm ./cloudhsm-pkcs11_latest_u24.04_amd64.deb

# Golang
wget https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz

rm go1.24.3.linux-amd64.tar.gz

export PATH=$PATH:/usr/local/go/bin
