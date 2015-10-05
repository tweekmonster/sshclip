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


function! sshclip#emulator#op_change(motion)
    return sshclip#emulator#handle('delete', '*', 'c', a:motion)
endfunction


function! sshclip#emulator#insert(register)
    let data = sshclip#register#get(a:register)
    return data[0]
endfunction


function! sshclip#emulator#handle(type, register, key, motion)
    let key_count = (v:count == v:count1) ? v:count : ''

    if a:register != '*' && a:register != '+'
        " Ignore all other registers and run their original command
        execute 'normal! ' key_count . '"' . a:register . a:key
    else
        " If motions are sent through from the operator functions, select
        " their appropriate ranges.  Otherwise, a:motion will indicate whether
        " or not the command was from visual mode.
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

        " Emulate the command using the unnamed buffer then send or retrieve
        " the desired register from the command line.
        if a:type == 'paste'
            let old_reg = getreg('"')
            let old_regtype = getregtype('"')
            let tmp = sshclip#register#get(a:register)

            if a:motion == 'v'
                " A visual paste replaces text.  The replaced text should be
                " placed back into the unnamed buffer.
                normal! y
                normal! gv
                call sshclip#register#put('*', '1', getreg('"'), getregtype('"'))
            endif

            call setreg('"', tmp[0], tmp[1])
            execute 'normal! ' key_count . a:key

            " Restore the unnamed register
            call setreg('"', old_reg, old_regtype)
        else
            let local_register = '0'
            if a:type == 'delete'
                if a:key ==? 'x' || a:key ==? 'c'
                    let local_register = '-'
                else
                    let local_register = '1'
                endif
            endif

            execute 'normal! ' key_count . a:key
            call sshclip#register#put(a:register, local_register, getreg('"'), getregtype('"'))

            " A change should put the user back into insert mode
            if a:key ==? 'c'
                if col('.') == 1
                    call feedkeys('i')
                else
                    call feedkeys('a')
                endif
            endif
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

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
