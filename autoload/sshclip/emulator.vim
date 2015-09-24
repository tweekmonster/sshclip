function! s:count()
    return (v:count == v:count1) ? v:count : ''
endfunction


function! sshclip#emulator#fetch(register)
    return system(sshclip#misc#command_str('get', a:register))
endfunction


function! sshclip#emulator#op_yank(motion)
    echomsg 'Yank op'
    return sshclip#emulator#emulate('yank', '*', 'y', a:motion)
endfunction


function! sshclip#emulator#op_delete(motion)
    echomsg 'Delete op'
    return sshclip#emulator#emulate('delete', '*', 'd', a:motion)
endfunction


function! sshclip#emulator#handle(type, register, key, motion)
    echomsg printf('%s - %s - %s - %s', a:type, a:register, a:key, a:motion)

    if a:register != '*' && a:register != '+'
        execute 'normal! ' s:count() . '"' . a:register . a:key
        return
    endif

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
        " Paste
        let @@ = sshclip#emulator#fetch(a:register)
        execute 'normal! ' s:count() . a:key
    else
        let local_register = '0'
        if a:type == 'delete'
            if a:key ==? 'x'
                let local_register = '-'
            else
                let local_register = '1'
            endif
        endif
        execute 'normal! ' s:count() . a:key
        call system(sshclip#misc#command_str('put', a:register) . ' --bg', getreg('"'))
        call setreg(local_register, getreg('"'), getregtype('"'))
    endif
endfunction
