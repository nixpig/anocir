#define _GNU_SOURCE
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>


static int get_ns_flag(const char *name) {
  if (strcmp(name, "pid") == 0) return CLONE_NEWPID;
  if (strcmp(name, "net") == 0) return CLONE_NEWNET;
  if (strcmp(name, "ipc") == 0) return CLONE_NEWIPC;
  if (strcmp(name, "uts") == 0) return CLONE_NEWUTS;
  if (strcmp(name, "user") == 0) return CLONE_NEWUSER;
  if (strcmp(name, "cgroup") == 0) return CLONE_NEWCGROUP;
  if (strcmp(name, "mnt") == 0) return CLONE_NEWNS;
  if (strcmp(name, "time") == 0) return CLONE_NEWTIME;
  return 0;
}

static void join_ns(const char *path, int flag) {
      char buf[512];
      snprintf(buf, sizeof(buf), "joining ns path=%s flag=%d", path, flag);

      int fd = open(path, O_RDONLY);
      if (fd < 0) {
          snprintf(buf, sizeof(buf), "failed to open %s: %s", path, strerror(errno));
          _exit(1);
      }

      if (setns(fd, flag) < 0) {
          snprintf(buf, sizeof(buf), "failed to setns %s: %s", path, strerror(errno));
          close(fd);
          _exit(1);
      }

      snprintf(buf, sizeof(buf), "joined ns %s successfully", path);
      close(fd);
}

__attribute__((constructor)) void nssetup(void) {
      char *join_ns_env = getenv("_ANOCIR_JOIN_NS");
      char *container_pid = getenv("_ANOCIR_CONTAINER_PID");

      char buf[1024];
      snprintf(buf, sizeof(buf), "nssetup called, _ANOCIR_JOIN_NS=%s, _ANOCIR_CONTAINER_PID=%s",
          join_ns_env ? join_ns_env : "(null)",
          container_pid ? container_pid : "(null)");

  // Format: "pid:/proc/123/ns/pid,net:/proc/123/ns/net,mnt:/proc/123/ns/mnt".
  if (!join_ns_env) return;

  // Before joining mount namespace, open fd to container's root so we can
  // chroot after joining.
  int root_fd = -1;
  if (container_pid) {
      char root_path[256];
      snprintf(root_path, sizeof(root_path), "/proc/%s/root", container_pid);
      root_fd = open(root_path, O_RDONLY | O_DIRECTORY);
      if (root_fd < 0) {
          snprintf(buf, sizeof(buf), "failed to open %s: %s", root_path, strerror(errno));
          // Continue without chroot - some use cases might not need it.
      }
  }

  char *env_copy = strdup(join_ns_env);
  if (!env_copy) {
      fprintf(stderr, "nssetup: strdup failed\n");
      if (root_fd >= 0) close(root_fd);
      _exit(1);
  }

  char *saveptr1, *saveptr2;
  char *entry = strtok_r(env_copy, ",", &saveptr1);

  while (entry) {
      char *type = strtok_r(entry, ":", &saveptr2);
      char *path = strtok_r(NULL, ":", &saveptr2);

      if (type && path) {
          int flag = get_ns_flag(type);
          if (flag) {
              join_ns(path, flag);
          }
      }

      entry = strtok_r(NULL, ",", &saveptr1);
  }

  free(env_copy);
  unsetenv("_ANOCIR_JOIN_NS");
  unsetenv("_ANOCIR_CONTAINER_PID");

  // After joining mount namespace, chroot using the fd we opened earlier.
  if (root_fd >= 0) {
      if (fchdir(root_fd) < 0) {
          snprintf(buf, sizeof(buf), "fchdir to container root failed: %s", strerror(errno));
          close(root_fd);
          _exit(1);
      }
      close(root_fd);

      if (chroot(".") < 0) {
          snprintf(buf, sizeof(buf), "chroot to container root failed: %s", strerror(errno));
          _exit(1);
      }

      if (chdir("/") < 0) {
          snprintf(buf, sizeof(buf), "chdir to / after chroot failed: %s", strerror(errno));
          _exit(1);
      }
  }
}
