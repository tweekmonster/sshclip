let s:path = expand('<sfile>:p:h')
let s:bin = resolve(printf('%s/../../bin/sshclip-client', s:path))
let s:regstore = {'+': 'clipboard', '*': 'primary'}
let s:status = ''


function! sshclip#misc#command(...)
    return [s:bin] + a:000
endfunction


function! sshclip#misc#command_str(...)
    return join([s:bin] + a:000, ' ')
endfunction


function! sshclip#misc#msg(...)
    echohl Title
    echo '[sshclip] '
    echon join(a:000, ' ')
    echohl None
endfunction


function! sshclip#misc#err(...)
    echohl ErrorMsg
    echo '[sshclip] Error'
    echon join(a:000, ' ')
    echohl None
endfunction


function! sshclip#misc#status(...)
    return s:status
endfunction


function! sshclip#misc#set_status(reg, out)
    if a:reg == '!'
        let s:status = ' [sshclip ignore] '
    else
        let s:status = printf(' [sshclip %s %s] ', a:out ? '<-' : '->', a:reg)
    endif
endfunction


function! sshclip#misc#setup_airline()
    call airline#parts#define_function('sshclip', 'sshclip#misc#status')
    let g:airline_section_gutter = get(g:, 'airline_section_gutter', '') . airline#section#create_right(['sshclip'])

    autocmd CursorHold * let s:status = ''
    autocmd CursorHoldI * let s:status = ''
endfunction


function! sshclip#misc#trim(str)
    return substitute(a:str, '\v^\s*(.{-})\s*$', '\1', '')
endfunction


function! sshclip#misc#can_send_str(str)
    return sshclip#misc#can_send_lines(split(a:str, "\n"))
endfunction


function! sshclip#misc#can_send_lines(lines)
    let minb = get(g:, 'clipboard_min_bytes', 0)
    if minb
        let excl_ws = get(g:, 'clipboard_exclude_whitespace', 1)
        let i = 0
        let l = len(a:lines)
        let bl = 0

        while i < l
            let line = get(a:lines, i, '')
            let bl += len(excl_ws ? sshclip#misc#trim(line) : line)
            if bl >= minb
                break
            endif
            let i += 1
        endwhile

        if bl < minb
            return 0
        endif
    endif

    return 1
endfunction

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
