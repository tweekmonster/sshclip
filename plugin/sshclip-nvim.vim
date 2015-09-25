if !has('nvim') || exists('g:clipboard_setup')
    finish
endif

let g:clipboard_setup = 1

function! s:setup()
    if get(g:, 'clipboard_monitor')
        call sshclip#misc#msg('Starting clipboard monitor')
        call jobstart(sshclip#misc#command('--monitor'))
    endif
endfunction

autocmd VimEnter * :call s:setup()
autocmd User AirlineAfterInit call sshclip#misc#setup_airline()

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
