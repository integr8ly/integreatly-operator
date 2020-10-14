#!/bin/sh
sudo dnf install unzip wget java -y
ssh-keygen -t rsa -N "" -f .ssh/id_rsa
cat .ssh/id_rsa.pub >> .ssh/authorized_keys
ssh -o StrictHostKeyChecking=no -i ~/.ssh/id_rsa localhost 'exit'
wget https://github.com/Hyperfoil/Hyperfoil/releases/download/release-0.9/hyperfoil-0.9.zip
unzip hyperfoil-0.9.zip
cd hyperfoil-0.9
#logs of the hyperfoil controller will be stored in ~/hyperfoil-0.9/nohup.out
nohup ./bin/controller.sh &



