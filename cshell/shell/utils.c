#include <sys/stat.h>
#include <stdlib.h>
#include "utils.h"


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


int store_create(const char *user, size_t size) {
    char store_path[size];
    char resolved[size];
    int ret;

    if (create_directory(STORE_BASE, resolved)) {
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
    int ret = 0;
    char store_path[size];

    if (user != NULL) {
        ret = snprintf(store_path, size, "%s/%s", STORE_BASE, user);
    } else {
        ret = snprintf(store_path, size, "%s", STORE_BASE);
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
                if (!store_create(user, size)) {
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
