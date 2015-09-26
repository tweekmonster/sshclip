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
            let @@ = sshclip#register#get(a:register)
            if a:motion == 'v'
                normal! y
                normal! gv
                call sshclip#register#put('*', '1', getreg('"'), getregtype('"'))
            endif
            execute 'normal! ' c . a:key
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
            call sshclip#register#put(a:register, local_register, getreg('"'), getregtype('"'))
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
