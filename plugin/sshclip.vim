if exists('g:clipboard_setup')
    finish
endif

let g:clipboard_setup = 1
let s:session_pidfile = printf('~/.cache/sshclip/vim_session_%d', getpid())


function! s:monitor()
    if get(g:, 'clipboard_monitor')
        call sshclip#misc#msg('Starting clipboard monitor')
        call system(sshclip#misc#command_str('--monitor', '--background'))
    endif
endfunction


function! s:setup_vim()
    call sshclip#keys#setup_interface()
    call sshclip#keys#setup_keymap()
endfunction


autocmd VimEnter * :call s:monitor()

if !has('nvim')
    autocmd VimEnter * :call s:setup_vim()
endif

autocmd User AirlineAfterInit call sshclip#misc#setup_airline()

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
