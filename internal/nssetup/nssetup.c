#define _GNU_SOURCE
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>


static void debug_log(const char *msg) {
  FILE *f = fopen("/tmp/nssetup-debug.log", "a");
  if (f) {
      fprintf(f, "%s\n", msg);
      fclose(f);
  }
}

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
      debug_log(buf);

      int fd = open(path, O_RDONLY);
      if (fd < 0) {
          snprintf(buf, sizeof(buf), "failed to open %s: %s", path, strerror(errno));
          debug_log(buf);
          _exit(1);
      }

      if (setns(fd, flag) < 0) {
          snprintf(buf, sizeof(buf), "failed to setns %s: %s", path, strerror(errno));
          debug_log(buf);
          close(fd);
          _exit(1);
      }

      snprintf(buf, sizeof(buf), "joined ns %s successfully", path);
      debug_log(buf);
      close(fd);
}

__attribute__((constructor)) void nssetup(void) {
      char *join_ns_env = getenv("_ANOCIR_JOIN_NS");

      char buf[1024];
      snprintf(buf, sizeof(buf), "nssetup called, _ANOCIR_JOIN_NS=%s", join_ns_env ? join_ns_env :
  "(null)");
      debug_log(buf);

  // Format: "pid:/proc/123/ns/pid,net:/proc/123/ns/net,mnt:/proc/123/ns/mnt"
  if (!join_ns_env) return;

  char *env_copy = strdup(join_ns_env);
  if (!env_copy) {
      fprintf(stderr, "nssetup: strdup failed\n");
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
}
