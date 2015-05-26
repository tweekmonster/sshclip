/* #include <sys/select.h> */
#include <sys/poll.h>
#include <sys/types.h>
#include <sys/stat.h>
#include <stdlib.h>
#include "utils.h"

int fd_readable(FILE *fd)
{
    struct pollfd pfd;
    pfd.fd = fileno(fd);
    pfd.events = POLLIN;
    return poll(&pfd, 1, 500) == 1;
}


int create_directory(const char *path, char *resolved) {
    struct stat s;
    int e = lstat(path, &s);
    if (e == -1) {
        if (errno == ENOENT) {
            if (mkdir(path, 0700) == 0) {
                strncpy(resolved, path, strlen(path) + 1);
                SC_LOG(LOG_DEBUG, "Created directory: %s", path);
                return 1;
            }
        }
        SC_ERRNO("Could not stat: %s", path);
        return 0;
    }

    if (S_ISLNK(s.st_mode)) {
        char *r = realpath(path, NULL);
        if (r == NULL) {
            SC_ERRNO("CCould not get realpould not get realpath (dead symlink?): %s", path);
            return 0;
        }
        SC_LOG(LOG_DEBUG, "Following symlink: %s -> %s", path, r);
        int ret = create_directory(r, resolved);
        free(r);
        return ret;
    } else if (S_ISDIR(s.st_mode)) {
        strncpy(resolved, path, strlen(path) + 1);
        return 1;
    }

    return 0;
}


int store_create(const char *home, const char *user, size_t size) {
    // Don't need need to recursively create directories.
    struct stat s;

    // First check if the home exists, because it should
    if (stat(home, &s) == -1 || !S_ISDIR(s.st_mode)) {
        if (S_ISDIR(s.st_mode)) {
            SC_LOG(LOG_ERR, "Home path is not a directory: %s", home);
        }
        SC_ERRNO("Could not stat home directory: %s", home);
        return 0;
    }

    char store_path[size];
    char resolved[size];
    int ret;

    ret = snprintf(store_path, size, "%s/%s", home, STORE_BASE);
    if (ret && create_directory(store_path, resolved)) {
        if (user != NULL) {
            ret = snprintf(store_path, size, "%s/%s", resolved, user);
            if (ret && create_directory(store_path, resolved)) {
                return 1;
            }
        } else {
            return 1;
        }
    }
    return 0;
}


int store_directory(const char *user, char *path, size_t size)
{
    uid_t uid = getuid();
    // Valgrind reports a leak here, see under BUGS in getpwent(3)
    struct passwd *pw = getpwuid(uid);

    int ret = 0;
    char store_path[size];

    if (user != NULL) {
        ret = snprintf(store_path, size, "%s/%s/%s", pw->pw_dir, STORE_BASE, user);
    } else {
        ret = snprintf(store_path, size, "%s/%s", pw->pw_dir, STORE_BASE);
    }

    if (ret > size - 1) {
        // Paranoia
        SC_LOG(LOG_WARNING, "snprintf wrote more bytes than was expected");
        return 0;
    }

    if (ret > 0) {
        struct stat s;
        int e = stat(store_path, &s);
        if (e == -1) {
            if (errno == ENOENT) {
                SC_LOG(LOG_WARNING, "Store directory does not exist: %s", store_path);
                if (!store_create(pw->pw_dir, user, size)) {
                    SC_LOG(LOG_ERR, "Could not create store directory");
                    return 0;
                }
            } else {
                SC_LOG(LOG_WARNING, "Stat error: %d", errno);
                return 0;
            }
        }

        strncpy(path, store_path, ret + 1);
        if (path[ret] != '\0') {
            // Extra paranoia
            return 0;
        }
    }

    return ret;
}

/* vim: set ts=4 sw=4 tw=0 et :*/
