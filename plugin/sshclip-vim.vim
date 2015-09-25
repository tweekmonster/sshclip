if has('nvim') || exists('g:clipboard_setup')
    finish
endif

let g:clipboard_setup = 1


function! s:setup()
    call sshclip#keys#setup_interface()
    call sshclip#keys#setup_keymap()

    if get(g:, 'clipboard_monitor')
        call sshclip#misc#msg('Starting clipboard monitor')
        call system(sshclip#misc#command_str('--monitor', '--background'))
    endif
endfunction


autocmd VimEnter * :call s:setup()
autocmd User AirlineAfterInit call sshclip#misc#setup_airline()
