Output demo.gif

Set FontSize 18
Set Theme "Catppuccin Mocha"
Set Width 960
Set Height 540
Set WindowBar Colorful
Set BorderRadius 10
Set Padding 20

Hide
Type "vagrant ssh"
Enter
Sleep 2s
Type "clear"
Enter
Sleep 1s
Show

Type "docker run -it --rm --runtime anocir ubuntu"
Sleep 1s
Enter
Sleep 3s

Type "[ -f /.dockerenv ] && echo 'Running inside Docker env'"
Sleep 1s
Enter
Sleep 2s

Type "clear"
Sleep 1s
Enter

Hide
Type "apt update"
Enter
Type "apt install kmod iproute2 -y"
Enter
Sleep 15s
Type "clear"
Enter
Show

Sleep 1s
Type "apt update && apt upgrade -y"
Sleep 1s
Enter
Sleep 8s
Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "hostname"
Sleep 1s
Enter
Sleep 1s

Type "ps aux"
Sleep 1s
Enter
Sleep 4s

Type "ps -p 1 -o command="
Sleep 1s
Enter
Sleep 3s

Type "mount | grep overlay"
Sleep 1s
Enter
Sleep 3s

Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "ip addr show"
Sleep 1s
Enter
Sleep 3s

Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "cat /proc/1/cgroup"
Sleep 1s
Enter
Sleep 3s

Type "cat /sys/fs/cgroup/$(basename $(cat /proc/1/cgroup | cut -d: -f3))/cgroup.controllers"
Sleep 1s
Enter
Sleep 3s

Type "cat /proc/1/status | grep Cap"
Sleep 1s
Enter
Sleep 3s

Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "ls -la /dev"
Sleep 1s
Enter
Sleep 3s

Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "mkdir -p /tmp/test"
Sleep 1s
Enter
Sleep 1s

Type "mount -t tmpfs none /tmp/test"
Sleep 1s
Enter
Sleep 3s

Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "ls -la /sys"
Sleep 1s
Enter
Sleep 3s

Type "touch /sys/test"
Sleep 1s
Enter
Sleep 3s

Type "clear"
Sleep 1s
Enter
Sleep 1s

Type "ls -la /proc/sys"
Sleep 1s
Enter
Sleep 3s

Type "touch /proc/sys/test"
Sleep 1s
Enter
Sleep 3s

Type "exit"
Sleep 1s
Enter
Sleep 2s

Type "clear"
Sleep 1s
Enter
Sleep 2s
