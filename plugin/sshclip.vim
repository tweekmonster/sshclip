if exists('g:clipboard_setup')
    finish
endif

let g:clipboard_setup = 1

command SSHClipKey :call sshclip#misc#set_encryption_key()
autocmd VimEnter * :call sshclip#misc#init()
autocmd User AirlineAfterInit call sshclip#misc#setup_airline()

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
