#ifndef UTILS_H_93GF8N0H
#define UTILS_H_93GF8N0H

#include <stdio.h>
#include <stdarg.h>
#include <errno.h>
#include <unistd.h>
#include <string.h>
#include <syslog.h>
#include "config.h"

#ifndef DEBUG
#   define SC_LOG(level, fmt, ...) syslog(level, fmt, ##__VA_ARGS__)
#else
#   define SC_LOG(level, fmt, ...) syslog(level, ("(%s:%d) %s: " fmt), __FILE__, __LINE__, __func__, ##__VA_ARGS__)
#endif

#define SC_ERRNO(fmt, ...) \
do { \
    char *ename = NULL; \
    switch(errno) { \
        case EACCES: \
            ename = "EACCES"; \
            break; \
        case EIO: \
            ename = "EIO"; \
            break; \
        case ELOOP: \
            ename = "ELOOP"; \
            break; \
        case ENAMETOOLONG: \
            ename = "ENAMETOOLONG"; \
            break; \
        case ENOENT: \
            ename = "ENOENT"; \
            break; \
        case ENOTDIR: \
            ename = "ENOTDIR"; \
            break; \
        case EOVERFLOW: \
            ename = "EOVERFLOW"; \
            break; \
        case EEXIST: \
            ename = "EEXIST"; \
            break; \
        case EMLINK: \
            ename = "EMLINK"; \
            break; \
        case ENOSPC: \
            ename = "ENOSPC"; \
            break; \
        default: \
            ename = "UNKNOWN"; \
            break; \
    } \
    SC_LOG(LOG_ERR, "[%s] " fmt, ename, __VA_ARGS__); \
} while (0)


int fd_readable(FILE *fd);

int store_directory(const char *user, char *path, size_t size);

#endif /* end of include guard: UTILS_H_93GF8N0H */

/* vim: set ts=4 sw=4 tw=0 et :*/
