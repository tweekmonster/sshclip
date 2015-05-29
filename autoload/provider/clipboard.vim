let s:path = expand('<sfile>:p:h')
let s:bin = resolve(printf('%s/../../bin/sshclip-client', s:path))
let s:exit = 0
let s:stdout = []
let s:stderr = []
let s:regstore = {'+': 'clipboard', '*': 'primary'}
let s:lastreg = ''

let g:last_yank_pending = 0
let g:last_yank = []

function! s:async_cb(job, data, event)
    if a:event == 'stdout'
        let s:stdout = a:data
    elseif a:event == 'stderr'
        let s:stderr = a:data
    elseif a:event == 'exit'
        let s:exit = a:data
        if s:exit != 0
            echohl ErrorMsg
            echo '[sshclip] Error: '
            echon join(s:stderr, ' - ')
        else
            echohl Title
            echo printf('[sshclip] Register %s sent (%s)',
                        \ s:lastreg, s:regstore[s:lastreg])
        endif

        echohl None

        if exists('s:job_id')
            unlet s:job_id
        endif
    endif
endfunction

let s:callbacks = {
    \ 'on_stdout': function('s:async_cb'),
    \ 'on_stderr': function('s:async_cb'),
    \ 'on_exit': function('s:async_cb'),
    \ }

function! s:run_job(put, register, lines)
    let s:lastreg = a:register
    let cmd = [s:bin, '-i', '-selection', s:regstore[a:register]]
    if !get(g:, 'clipboard_system_passthru', 1)
        let cmd += ['--no-passthru']
    endif

    let s:job_id = jobstart(cmd, s:callbacks)
    let data = join(a:lines, "\n")
    let l = len(data)
    let i = 0
    let c = 1
    " For some reason, the entire payload for data that exceeds ~8000 bytes
    " doesn't get sent unless it's sent 1 byte at a time. Increasing the chunk
    " size beyond 1 seems to cause loss, too.  Not sure if I'm overlooking
    " something or I just really suck at vim script.
    " Tested by copying the entire contents of the sshclip-client script.
    while i < l
        call jobsend(s:job_id, data[i : i+c])
        let i += c + 1
    endwhile

    call jobclose(s:job_id, 'stdin')

    return s:stdout
endfunction

function! s:trim(str)
    return substitute(a:str, '\v^\s*(.{-})\s*$', '\1', '')
endfunction

let s:clipboard = {}

function! s:clipboard.set(lines, type, register)
    let minb = get(g:, 'clipboard_min_bytes', 0)
    if minb
        let excl_ws = get(g:, 'clipboard_exclude_whitespace', 1)
        let i = 0
        let l = len(a:lines)
        let bl = 0

        while i < l
            let line = get(a:lines, i, '')
            let bl += len(excl_ws ? s:trim(line) : line)
            if bl >= minb
                break
            endif
            let i += 1
        endwhile

        if bl < minb
            echohl WarningMsg
            echomsg '[sshclip] Not enough bytes to send'
            echohl None
            return
        endif
    endif

    call s:run_job(1, a:register, a:lines)
endfunction

function! s:clipboard.get(register)
    let cmd = [s:bin, '-o', '-selection', s:regstore[a:register]]
    if exists('s:job_id')
        " Not sure if this is necessary
        echo "[sshclip] Waiting for put job..."
        call jobwait([s:job_id])
    endif

    " Non-async call
    let stderr_file = tempname()
    let cmd += ['2>', stderr_file]
    let stdout = systemlist(join(cmd, ' '), [''], 1)
    let stderr = []
    let exit_status = v:shell_error

    if filereadable(stderr_file)
        let stderr = readfile(stderr_file, '', 5)
        call delete(stderr_file)
    endif

    if exit_status
        echohl ErrorMsg
        echo '[sshclip] Error'
        if len(stderr)
            echon ':' join(stderr, ' - ')
        endif
        return ''
    else
        echohl Title
        echo printf('[sshclip] Register %s retrieved (%s)',
                    \ a:register, s:regstore[a:register])
    endif

    echohl None

    return stdout
endfunction

function! provider#clipboard#Call(method, args)
    return call(s:clipboard[a:method], a:args, s:clipboard)
endfunction

" vim: set ts=4 sw=4 tw=78 et :
