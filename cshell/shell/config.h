#ifndef CONFIG_H_FPJXDL3K
#define CONFIG_H_FPJXDL3K

// The identifier for syslog.
#define LOG_TAG "sshclip-shell"

// The directory for the store. This can contain multiple path components if
// the directory is created. The store_create function does not do recursive
// directory creation.
#define STORE_BASE ".sshclip_store"

// Maximum length of commands passed to the shell, either through the command
// line or through SSH_ORIGINAL_COMMAND. If this length is exceeded for either
// the shell will ignore the request.
#define MAX_CMD_LEN 60

// Maximum length for the store path.
#define MAX_PATH_LEN 255

// Maximum bytes that will be read before the rest of the stream is ignored.
#define MAX_READ_BYTES 786432

// Maximum number of arguments that will be read in SSH_ORIGINAL_COMMAND or argv.
#define MAX_CMDS 5

// Valid store files.
#define VALID_STORES "clipboard", "primary", "secondary"

#endif /* end of include guard: CONFIG_H_FPJXDL3K */

/* vim: set ts=4 sw=4 tw=0 et :*/
