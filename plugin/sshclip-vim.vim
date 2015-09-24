if has('nvim') || exists('g:clipboard_setup')
    finish
endif

let g:clipboard_setup = 1

call sshclip#keys#setup_interface()
call sshclip#keys#setup_keymap()
