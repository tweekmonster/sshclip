let s:path = expand('<sfile>:p:h')
let s:bin = resolve(printf('%s/../../bin/sshclip-client', s:path))
let s:regstore = {'+': 'clipboard', '*': 'primary'}
let s:status = ''
let s:encryption_key = expand('~/.cache/sshclip/.sshclip_key')


function! sshclip#misc#command(...)
    let cmd = [s:bin] + a:000
    if !get(g:, 'sshclip_enable_encryption', 1)
        return cmd + ['--no-encryption']
    endif
    return cmd
endfunction


function! sshclip#misc#command_str(...)
    return sshclip#misc#trim(join([s:bin] + a:000, ' '))
endfunction


function! sshclip#misc#msg(...)
    echohl Title
    echo join(['[sshclip]'] + a:000, ' ')
    echohl None
endfunction


function! sshclip#misc#err(...)
    echohl ErrorMsg
    echomsg join(['[sshclip]'] + a:000, ' ')
    echohl None
endfunction


function! sshclip#misc#start_monitor(encryption)
    if get(g:, 'clipboard_monitor')
        call sshclip#misc#msg('Starting clipboard monitor')
        call system(sshclip#misc#command_str('--monitor', '--background'))
    endif
endfunction


function! sshclip#misc#set_encryption_key()
    while 1
        let secret = inputsecret('[sshclip] Enter a secret key: ')
        let secret2 = inputsecret('[sshclip] Verify secret key: ')
        if secret ==# secret2
            let cache_dir = fnamemodify(s:encryption_key, ':p:h')
            if !isdirectory(cache_dir)
                call mkdir(cache_dir, 'p')
            endif
            call writefile([secret], s:encryption_key, 'b')
            call system(printf('chmod 0600 %s', s:encryption_key))
            call system(sshclip#misc#command_str('--kill'))
            call sshclip#misc#init()
            break
        endif
        call sshclip#misc#err('Keys don''t match!')
    endwhile
endfunction


function! sshclip#misc#init()
    if has('nvim')
        set clipboard+=unnamed
    endif

    if !has('nvim') && !exists('s:vim_key_setup')
        let s:vim_key_setup = 1
        call sshclip#keys#setup_interface()
        call sshclip#keys#setup_keymap()
    endif

    call sshclip#register#update_commands()

    if get(g:, 'sshclip_enable_encryption', 1)
        if !filereadable(s:encryption_key)
            call sshclip#misc#err('Encryption key is not readable.  Run :SSHClipKey or set g:sshclip_enable_encryption to 0')
            return
        endif
        call sshclip#misc#start_monitor(1)
    else
        call sshclip#misc#start_monitor(0)
    endif

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
