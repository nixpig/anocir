Vagrant.configure("2") do |config|
  config.vm.box = "bento/ubuntu-24.04"
  config.vm.synced_folder '.', '/anocir'

  config.vm.provider "virtualbox" do |vb|
    vb.memory = "4096"
    vb.cpus = "2"
  end

  config.vm.provision "shell", inline: <<-SHELL
    set -e -x -o pipefail

    apt-get update && apt-get install -y git ca-certificates wget make vim gcc libseccomp-dev

    # Install go
    if ! command -v go 2>&1 >/dev/null; then
      wget https://go.dev/dl/go1.23.4.linux-amd64.tar.gz -O go.tar.gz
      tar -C /usr/local -xzf go.tar.gz
      echo "PATH=$PATH:/usr/local/go/bin" >> /etc/environment
    fi

    # Clone runtime-tools repo
    if [ ! -d "runtime-tools" ]; then
      git clone https://github.com/opencontainers/runtime-tools.git
    fi

    # Checkout a specific tree so it's not potentially changing underneath us
    git -C runtime-tools checkout f7e3563b0271e5cd52d5c915684ea11ef2779572

    # Build runtime tools
    (cd runtime-tools && make runtimetest validation-executables)
  SHELL
end
