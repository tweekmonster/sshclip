if !has('nvim') || exists('g:clipboard_setup')
    finish
endif

let g:clipboard_setup = 1
let s:bin = resolve(printf('%s/../bin/sshclip-client', expand('<sfile>:p:h')))

function! s:monitor()
    if get(g:, 'clipboard_monitor')
        echo '[sshclip] Starting clipboard monitor'
        call jobstart([s:bin, '--monitor'])
    endif
endfunction

autocmd VimEnter * :call s:monitor()
