Vagrant.configure("1") do |config|
  config.vm.box = "bento/ubuntu-24.04"
  config.vm.synced_folder '.', '/brownie'

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "4096"
    vb.cpus = "2"
  end

  config.vm.provision "shell", inline: <<-SHELL
    set -e -x -o pipefail

    apt-get update && apt-get install -y ca-certificates wget make vim gcc libseccomp-dev

    # Install docker
    if ! command -v docker 2>&1 >/dev/null; then
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

      # Add user to docker group
      gpasswd -a vagrant docker
    fi

    # Stop and start Docker service with brownie runtime
    service docker stop
    dockerd --add-runtime brownie=/brownie/tmp/bin/brownie \
      > /dev/null 2>&1 & disown


    # Install go
    if ! command -v go 2>&1 >/dev/null; then
      wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz -O go.tar.gz
      tar -C /usr/local -xzf go.tar.gz
      echo "PATH=$PATH:/usr/local/go/bin" >> /etc/environment
    fi

    # systemd cgroup jiggery-pokery
    if ! grep -qs '/sys/fs/cgroup/systemd' /proc/mounts; then
      mkdir -p /sys/fs/cgroup/systemd
      mount -t cgroup -o none,name=systemd cgroup /sys/fs/cgroup/systemd
    fi
  SHELL
end
