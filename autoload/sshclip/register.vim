" Cache command strings to avoid the repeated function call and join
let s:commands = {'*': {}, '+': {}}
let s:history = []
let s:tmp_register = {'*': '', '+': ''}
let s:last = {'*': reltime(), '+': reltime()}


function! sshclip#register#update_commands()
    let s:commands['*']['get'] = sshclip#misc#command_str('-o', '-selection', 'primary')
    let s:commands['*']['put'] = sshclip#misc#command_str('-i', '-selection', 'primary', '--background')
    let s:commands['+']['get'] = sshclip#misc#command_str('-o', '-selection', 'clipboard')
    let s:commands['+']['put'] = sshclip#misc#command_str('-i', '-selection', 'clipboard', '--background')
endfunction


function! sshclip#register#put(register, local_register, data, regtype)
    let s:last[a:register] = reltime()

    if type(a:data) == 3
        let data = join(a:data, "\n")
    else
        let data = a:data
    endif

    if sshclip#misc#can_send_str(data)
        let s:tmp_register[a:register] = printf('%s:%s', a:regtype, data)
        call system(s:commands[a:register]['put'], s:tmp_register[a:register])
        if v:shell_error
            call sshclip#misc#err(v:shell_error)
            return
        else
            call sshclip#misc#set_status(a:register, 1)
        endif
    else
        call sshclip#misc#set_status('!', 1)
    endif

    if a:local_register != '' && !has('nvim')
        call setreg(a:local_register, data, a:regtype)
    endif
endfunction


function! sshclip#register#get(register)
    let get_delta = str2float(reltimestr(reltime(s:last[a:register])))
    if get_delta < 0.25
        let data = s:tmp_register[a:register]
    else
        let data = system(s:commands[a:register]['get'])
        let s:tmp_register[a:register] = data
    endif

    let s:last[a:register] = reltime()

    if v:shell_error
        call sshclip#misc#err(v:shell_error)
        return
    else
        call sshclip#misc#set_status(a:register, 0)
    endif

    let regtype = 'V'
    let i = stridx(data, ':')
    if i != -1
        let regtype = data[:(i-1)]
        if strlen(regtype) < 5 && regtype =~ "\^\\(\<c-v>\\|v\\|V\\)\\d*"
            let data = data[(i+1):]
        else
            let regtype = 'V'
        endif
    endif

    if has('nvim')
        return [split(data, "\n", 1), regtype]
    endif

    return [data, regtype]
endfunction

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
