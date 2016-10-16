# sshclip

## Note: This is being rewritten in Go to be much simpler to use

With [Neovim](https://github.com/neovim/neovim)'s provider infrastructure, the
unnamed (\*) and plus (+) registers can be securely sent to an SSH account that
uses a shell specifically for storing and retrieving the clipboard.  The
purpose behind this is to allow you to have access to the same clipboard data
across multiple computers (ideally in a closed network).

It works, but is very much not production ready.  This will be updated with
more information later.

If you want to try it, look through `bin` to get an idea of how it works.
`cshell` is a C version of the server script, it has a script for setting up
restricted user account for sshclip.
