#include <stdlib.h>
#include <time.h>
#include <string.h>
#include "main.h"
#include "utils.h"


const char * const stores[] = { VALID_STORES, NULL };

int is_valid_store(char *store)
{
    int i = 0;
    const char *store_check;
    while((store_check = stores[i]) != NULL) {
        if (strcmp(store, store_check) == 0) {
            return 1;
        }
        ++i;
    }
    return 0;
}


int handle_clipboard_request(int argc, char *argv[], int restrictc, char *restrictv[])
{
    int put = -1;
    char *store = NULL;

    if (argc != 2) {
        SC_LOG(LOG_DEBUG, "Incorrect number of arguments: %d", argc);
        fputs("Incorrect number of arguments", stderr);
        return SC_EX_CMD;
    }

    // Only deal with explicit get/put commands
    if (strcmp(argv[0], "get") == 0) {
        put = 0;
    } else if (strcmp(argv[0], "put") == 0) {
        put = 1;
    } else {
        SC_LOG(LOG_DEBUG, "Unknown command: %s", argv[0]);
        fputs("Bad command", stderr);
        return SC_EX_CMD;
    }

    store = argv[1];
    if (!is_valid_store(store)) {
        SC_LOG(LOG_WARNING, "Got an invalid store: %s", store);
        fprintf(stderr, "Invalid store: %s\n", store);
        return SC_EX_CMD;
    }

    int readonly = 0;
    char *user = NULL;

    for (int i = 0; i < restrictc; ++i) {
        if (strcmp(restrictv[i], "user") == 0) {
            if (i < restrictc - 1) {
                user = restrictv[++i];
                clean_str(user);
                continue;
            }
        }

        if (strcmp(restrictv[i], "readonly") == 0) {
            readonly = 1;
        }
    }

    if (store == NULL) {
        fputs("No valid store specified\n", stderr);
        return SC_EX_CMD;
    }

    if (put && readonly) {
        fputs("Can't put in readonly mode\n", stderr);
        return SC_EX_CMD;
    }

    char store_path[MAX_PATH_LEN];
    int ret;

    ret = store_directory(user, store_path, MAX_PATH_LEN);
    if (!ret) {
        SC_LOG(LOG_ERR, "Could not get store directory");
        return SC_EX_CMD;
    }

    char store_filename[MAX_PATH_LEN];
    ret = snprintf(store_filename, MAX_PATH_LEN, "%s/%s", store_path, store);
    if (!ret || ret > MAX_PATH_LEN - 1) {
        SC_LOG(LOG_ERR, "Could not get store filename");
        return SC_EX_CMD;
    }

    SC_LOG(LOG_DEBUG, "Using store file: %s", store_filename);
    SC_LOG(LOG_INFO, "Action: %s, Store: %s, Read Only: %s, User: %s",
            put ? "put" : "get", store, readonly ? "Yes" : "No", user);

    FILE *fd;
    int bytes_r = 0;
    int bytes_w = 0;
    size_t wb = 0;
    size_t rb = 0;
    char buff_r[1024];

    if (!put) {
        fd = fopen(store_filename, "rb");
        if (fd != NULL) {
            while ((rb = fread(buff_r, 1, 1024, fd)) != 0)
            {
                wb = fwrite(buff_r, 1, rb, stdout);
                if (wb == 0) {
                    errno = ferror(fd);  // Not sure if this is right
                    SC_ERRNO("Could not write byte to: %s", store_filename);
                    return SC_EX_CMD;
                }
                bytes_r += rb;
                bytes_w += wb;
                if (bytes_w >= MAX_READ_BYTES) {
                    SC_LOG(LOG_WARNING, "Hit read limit for: %s", store_filename);
                    break;
                }
            }

            fclose(fd);
        } else {
            SC_LOG(LOG_DEBUG, "Could not open store: %s", store_filename);
        }

        SC_LOG(LOG_DEBUG, "Wrote %d bytes and read %d from: %s", bytes_w, bytes_r, store_filename);
    } else if (put) {
        fd = fopen(store_filename, "wb");
        if (fd == NULL) {
            SC_ERRNO("Could not open file for writing: %s", store_filename);
            return SC_EX_CMD;
        }

        while (fd_readable(stdin) && (rb = fread(buff_r, 1, 1024, stdin)) != 0) {
            bytes_r += rb;
            if (is_valid_b64_str(buff_r, rb)) {
                wb = fwrite(buff_r, 1, rb, fd);
                if (wb == 0) {
                    errno = ferror(fd);  // Not sure if this is right
                    SC_ERRNO("Could not write byte to: %s", store_filename);
                    fclose(fd);
                    return SC_EX_CMD;
                }
                bytes_w += wb;
            } else {
                SC_LOG(LOG_WARNING, "Found non-b64 characters in stdin");
                return SC_EX_CMD;
            }

            if (bytes_r >= MAX_READ_BYTES) {
                SC_LOG(LOG_WARNING, "Hit write limit for: %s", store_filename);
                break;
            }
        }

        fclose(fd);
        SC_LOG(LOG_DEBUG, "Read %d bytes and wrote %d to: %s", bytes_r, bytes_w, store_filename);
    }

    return SC_EX_OK;
}


int process_commands(int argc, char *argv[])
{
    if (getuid() == 0) {
        SC_LOG(LOG_ALERT, "Refusing to run as root");
        return SC_EX_INVOCATION;
    }

    if (argc < 2 || strcmp(argv[1], "-c") != 0) {
        SC_LOG(LOG_WARNING, "Bad invocation");
        return SC_EX_INVOCATION;
    }

    argc-=2;
    argv+=2;

    for (int i = 0; i < argc; ++i) {
        SC_LOG(LOG_DEBUG, "argv: %s\n", argv[i]);
    }

    int cmd_len = 0;
    char *cmd_tok = NULL;
    char *orig_argv[MAX_CMDS];
    int orig_argc = 0;

    for (int i = 0; i < argc; ++i) {
        cmd_tok = strtok(argv[i], " ");
        while (cmd_tok != NULL && orig_argc < MAX_CMDS) {
            orig_argv[orig_argc] = cmd_tok;
            cmd_len += strlen(cmd_tok);
            if (cmd_len > MAX_CMD_LEN) {
                SC_LOG(LOG_ERR, "Refusing to handle argv with length greater than %d", cmd_len);
            }
            cmd_tok = strtok(NULL, " ");
            orig_argc++;
        }
    }

    char *ssh_orig_cmd = getenv("SSH_ORIGINAL_COMMAND");
    if (ssh_orig_cmd != NULL) {
        SC_LOG(LOG_DEBUG, "Running with SSH_ORIGINAL_COMMAND: \"%s\"", ssh_orig_cmd);
        char *orig_cmd[MAX_CMDS];
        int orig_cmdc = 0;
        cmd_len = 0;
        cmd_tok = strtok(ssh_orig_cmd, " ");
        while (cmd_tok != NULL && orig_cmdc < MAX_CMDS) {
            orig_cmd[orig_cmdc] = cmd_tok;
            cmd_len += strlen(cmd_tok);
            if (cmd_len > MAX_CMD_LEN) {
                SC_LOG(LOG_ERR, "Refusing to handle SSH_ORIGINAL_COMMAND with length greater than %d", cmd_len);
                return SC_EX_CMD;
            }
            cmd_tok = strtok(NULL, " ");
            orig_cmdc++;
        }

        return handle_clipboard_request(orig_cmdc, orig_cmd, orig_argc, orig_argv);
    }

    return handle_clipboard_request(orig_argc, orig_argv, 0, NULL);
}


int main(int argc, char *argv[])
{
#ifdef DEBUG
    openlog(LOG_TAG, LOG_PID | LOG_PERROR, LOG_AUTH);
    setlogmask(LOG_UPTO(LOG_DEBUG));
#else
    openlog(LOG_TAG, LOG_PID, LOG_AUTH);
    setlogmask(LOG_UPTO(LOG_INFO));
#endif

    int ret = process_commands(argc, argv);

    if (ret != SC_EX_OK) {
        srand(time(NULL));
        sleep((rand() % 10) + 1);
    }

    return ret;
}

/* vim: set ts=4 sw=4 tw=0 et :*/
