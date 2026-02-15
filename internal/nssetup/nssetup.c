#define _GNU_SOURCE
#include <fcntl.h>
#include <sched.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>
#include <unistd.h>
#include <errno.h>
#include <sys/mount.h>
#include <linux/limits.h>

// Environment variable names for mount setup.
#define ENV_ROOTFS "_ANOCIR_ROOTFS"
#define ENV_ROOTFS_PROPAGATION "_ANOCIR_ROOTFS_PROPAGATION"

// Parse rootfs propagation string to mount flags.
// NOTE: For rshared/shared, we STILL use rslave during initial setup.
// The actual shared propagation is set AFTER pivot_root in Go.
// This prevents mount operations from propagating to the host during setup.
static unsigned long parse_propagation(const char *prop) {
    if (!prop || prop[0] == '\0') {
        return MS_SLAVE | MS_REC; // Default: rslave
    }
    // For shared/rshared, we use rslave during setup to prevent propagation.
    // The Go code will set the actual propagation after pivot_root.
    if (strcmp(prop, "shared") == 0 || strcmp(prop, "rshared") == 0) {
        return MS_SLAVE | MS_REC;
    }
    if (strcmp(prop, "private") == 0) {
        return MS_PRIVATE;
    }
    if (strcmp(prop, "rprivate") == 0) {
        return MS_PRIVATE | MS_REC;
    }
    if (strcmp(prop, "slave") == 0) {
        return MS_SLAVE;
    }
    if (strcmp(prop, "rslave") == 0) {
        return MS_SLAVE | MS_REC;
    }
    if (strcmp(prop, "unbindable") == 0) {
        return MS_UNBINDABLE;
    }
    if (strcmp(prop, "runbindable") == 0) {
        return MS_UNBINDABLE | MS_REC;
    }
    return MS_SLAVE | MS_REC; // Default: rslave
}

// Make the nearest parent mount point of path private.
// This walks up the directory tree until it finds a mount point.
static int rootfs_parent_mount_private(const char *path) {
    char parent[PATH_MAX];
    strncpy(parent, path, PATH_MAX - 1);
    parent[PATH_MAX - 1] = '\0';

    while (1) {
        if (mount("", parent, "", MS_PRIVATE, "") == 0) {
            return 0; // Success
        }
        if (errno != EINVAL) {
            // Real error, not "not a mount point"
            fprintf(stderr, "nssetup: make %s private failed: %s\n",
                    parent, strerror(errno));
            return -1;
        }
        // EINVAL means not a mount point, try parent directory.
        if (strcmp(parent, "/") == 0) {
            // Reached root, "/" is always a mount point.
            // This shouldn't happen, but if it does, we're done.
            return 0;
        }
        char *last_slash = strrchr(parent, '/');
        if (last_slash == parent) {
            // At root level, try "/" itself
            parent[1] = '\0';
        } else if (last_slash) {
            *last_slash = '\0';
        } else {
            break;
        }
    }
    return 0;
}

// Setup rootfs mounts following runc's prepareRoot sequence:
// 1. Set "/" propagation based on rootfsPropagation
// 2. Make rootfs's parent mount private
// 3. Bind mount rootfs to itself
static void setup_rootfs_mounts(const char *rootfs, const char *propagation) {
    unsigned long flag = parse_propagation(propagation);

    // Step 1: Set "/" propagation.
    if (mount("", "/", "", flag, "") < 0) {
        fprintf(stderr, "nssetup: set / propagation failed: %s\n", strerror(errno));
        _exit(1);
    }

    // Step 2: Make rootfs's parent mount private.
    // This is critical for pivot_root and prevents propagation of rootfs
    // operations to the parent namespace.
    if (rootfs_parent_mount_private(rootfs) < 0) {
        _exit(1);
    }

    // Step 3: Bind mount rootfs to itself.
    if (mount(rootfs, rootfs, "bind", MS_BIND | MS_REC, "") < 0) {
        fprintf(stderr, "nssetup: bind mount rootfs failed: %s\n", strerror(errno));
        _exit(1);
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
      int fd = open(path, O_RDONLY);
      if (fd < 0) {
          fprintf(stderr, "nssetup: failed to open %s: %s\n", path, strerror(errno));
          _exit(1);
      }

      if (setns(fd, flag) < 0) {
          fprintf(stderr, "nssetup: failed to setns %s: %s\n", path, strerror(errno));
          close(fd);
          _exit(1);
      }

      close(fd);
}

__attribute__((constructor)) void nssetup(void) {
    char *join_ns_env = getenv("_ANOCIR_JOIN_NS");
    char *container_pid = getenv("_ANOCIR_CONTAINER_PID");
    char *rootfs = getenv(ENV_ROOTFS);
    char *rootfs_propagation = getenv(ENV_ROOTFS_PROPAGATION);

    // Handle rootfs mount setup for initial container creation.
    // This runs when we're in a new mount namespace (created by CLONE_NEWNS)
    // and need to set up the rootfs before Go does anything.
    // This MUST happen before Go starts to avoid thread-related issues.
    if (rootfs && rootfs[0] != '\0') {
        setup_rootfs_mounts(rootfs, rootfs_propagation);

        // Clear the environment variables.
        unsetenv(ENV_ROOTFS);
        unsetenv(ENV_ROOTFS_PROPAGATION);
    }

    // Format: "pid:/proc/123/ns/pid,net:/proc/123/ns/net,mnt:/proc/123/ns/mnt".
    if (!join_ns_env) return;

    // Before joining mount namespace, open fd to container's root so we can
    // chroot after joining.
    int root_fd = -1;
    if (container_pid) {
        char root_path[256];
        snprintf(root_path, sizeof(root_path), "/proc/%s/root", container_pid);
        root_fd = open(root_path, O_RDONLY | O_DIRECTORY);
        // Continue without chroot if open fails - some use cases might not need it.
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
            fprintf(stderr, "nssetup: fchdir to container root failed: %s\n", strerror(errno));
            close(root_fd);
            _exit(1);
        }
        close(root_fd);

        if (chroot(".") < 0) {
            fprintf(stderr, "nssetup: chroot to container root failed: %s\n", strerror(errno));
            _exit(1);
        }

        if (chdir("/") < 0) {
            fprintf(stderr, "nssetup: chdir to / after chroot failed: %s\n", strerror(errno));
            _exit(1);
        }
    }
}
