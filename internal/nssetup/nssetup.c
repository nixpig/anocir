#define _GNU_SOURCE
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

__attribute__((constructor)) void nssetup(void) {
    char *mnt_ns = getenv("_ANOCIR_MNT_NS");
    if (!mnt_ns) return;

    int fd = open(mnt_ns, O_RDONLY);
    if (fd < 0) {
        fprintf(stderr, "nssetup: failed to open %s\n", mnt_ns);
        _exit(1);
    }

    if (setns(fd, CLONE_NEWNS) < 0) {
        fprintf(stderr, "nssetup: failed to setns %s\n", mnt_ns);
        close(fd);
        _exit(1);
    }

    close(fd);
    unsetenv("_ANOCIR_MNT_NS");
}
