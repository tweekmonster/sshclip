" Cache command strings to avoid the repeated function call and join
let s:commands = {'*': {}, '+': {}}
let s:commands['*']['get'] = sshclip#misc#command_str('-o', '-selection', 'primary')
let s:commands['*']['put'] = sshclip#misc#command_str('-i', '-selection', 'primary', '--background')
let s:commands['+']['get'] = sshclip#misc#command_str('-o', '-selection', 'clipboard')
let s:commands['+']['put'] = sshclip#misc#command_str('-i', '-selection', 'clipboard', '--background')


function! sshclip#emulator#op_yank(motion)
    return sshclip#emulator#handle('yank', '*', 'y', a:motion)
endfunction


function! sshclip#emulator#op_delete(motion)
    return sshclip#emulator#handle('delete', '*', 'd', a:motion)
endfunction


function! sshclip#emulator#handle(type, register, key, motion)
    let c = (v:count == v:count1) ? v:count : ''

    if a:register != '*' && a:register != '+'
        execute 'normal! ' c . '"' . a:register . a:key
    else
        let o_selection = &selection
        let &selection = 'inclusive'

        if a:motion ==# 'char'
            normal! `[v`]
        elseif a:motion ==# 'line'
            normal! '[V']
        elseif a:motion ==# 'block'
            execute "normal! `[\<C-v>`]"
        elseif a:motion != ''
            normal! gv
        endif

        let &selection = o_selection

        if a:type == 'paste'
            let @@ = system(s:commands[a:register]['get'])
            execute 'normal! ' c . a:key
            call sshclip#misc#set_status(a:register, 0)
        else
            let local_register = '0'
            if a:type == 'delete'
                if a:key ==? 'x'
                    let local_register = '-'
                else
                    let local_register = '1'
                endif
            endif

            execute 'normal! ' c . a:key
            let data = getreg('"')

            if sshclip#misc#can_send_str(data)
                call system(s:commands[a:register]['put'], data)
                call sshclip#misc#set_status(a:register, 1)
            else
                call sshclip#misc#set_status('!', 1)
            endif

            call setreg(local_register, data, getregtype('"'))
        endif
    endif

    if a:type == 'delete'
        silent doautocmd User SSHClipDelete
    elseif a:type == 'paste'
        silent doautocmd User SSHClipPaste
    elseif a:type == 'yank'
        silent doautocmd User SSHClipYank
    endif
endfunction
