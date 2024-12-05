Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-22.04"
  config.vm.synced_folder '.', '/brownie'

  config.vm.provider "virtualbox" do |vb|
    vb.gui = true
    vb.memory = "4096"
    vb.cpus = "2"
  end


  config.vm.provision "shell", inline: <<-SHELL
    set -e -x -o pipefail

    apt-get update && apt-get install -y ca-certificates wget make vim

    # Install docker
    wget \
      https://download.docker.com/linux/ubuntu/dists/jammy/pool/stable/amd64/containerd.io_1.7.24-1_amd64.deb \
      https://download.docker.com/linux/ubuntu/dists/jammy/pool/stable/amd64/docker-ce-cli_27.3.1-1~ubuntu.22.04~jammy_amd64.deb \
      https://download.docker.com/linux/ubuntu/dists/jammy/pool/stable/amd64/docker-ce_27.3.1-1~ubuntu.22.04~jammy_amd64.deb \
      https://download.docker.com/linux/ubuntu/dists/jammy/pool/stable/amd64/docker-buildx-plugin_0.17.1-1~ubuntu.22.04~jammy_amd64.deb \
      https://download.docker.com/linux/ubuntu/dists/jammy/pool/stable/amd64/docker-compose-plugin_2.29.7-1~ubuntu.22.04~jammy_amd64.deb

    dpkg -i \
      containerd.io_*_amd64.deb \
      docker-ce-cli_*_amd64.deb \
      docker-ce_*_amd64.deb \
      docker-buildx-plugin_*_amd64.deb \
      docker-compose-plugin_*_amd64.deb

    # Start docker service with brownie runtime
    service docker stop
    dockerd --add-runtime brownie=/brownie/tmp/bin/brownie \
      > /dev/null 2>&1 & disown

    # Add user to docker group
    gpasswd -a vagrant docker

    # Install go
    wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz -O go.tar.gz
    tar -C /usr/local -xzf go.tar.gz
    echo "PATH=$PATH:/usr/local/go/bin" >> /etc/environment

    # systemd cgroup jiggery-pokery
    mkdir /sys/fs/cgroup/systemd
    mount -t cgroup -o none,name=systemd cgroup /sys/fs/cgroup/systemd
  SHELL
end
