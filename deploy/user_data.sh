#!/bin/bash
set -e

sudo yum update -y

sudo yum install -y git

sudo yum install -y python3 python3-pip

GO_VERSION="1.22.6"
wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
export GOPATH=$HOME/go
echo "export PATH=$PATH:/usr/local/go/bin" >> /home/ec2-user/.bashrc
echo "export GOPATH=\$HOME/go" >> /home/ec2-user/.bashrc

sudo amazon-linux-extras enable docker
sudo yum install -y docker
sudo systemctl enable docker
sudo systemctl start docker
sudo usermod -aG docker ec2-user

sudo pip3 install docker-compose

cd /home/ec2-user
git clone https://github.com/atharva789/geospatial-web-scraper.git

cd geospatial-web-scraper/deploy/compose
docker-compose up -d
