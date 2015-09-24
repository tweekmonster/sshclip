let s:maps = [
            \ ['yank', ['y', 'yy', 'Y']],
            \ ['delete', ['x', 'X', 'd', 'dd', 'D']],
            \ ['paste', ['p', 'P', 'gp', 'gP', ']p', ']P', '[p', '[P']],
            \ ]

" Unfortunately have to map all register shortcuts so that maps without a
" register prefix can set unnamed to *
let s:registers = ['"', '-', ':', '.', '%', '#', '=', '*', '+', '-', '_', '/']
let s:ro_registers = [':', '.', '%', '#']


function! s:can_vmap(k)
    if len(a:k) == 2 && a:k[0] == a:k[1]
        return 0
    endif
    return 1
endfunction


function! sshclip#keys#setup_interface()
    nnoremap <silent> <Plug>(sshclip-op-y) :<C-u>set operatorfunc=sshclip#emulator#op_yank<Return>g@
    nnoremap <silent> <Plug>(sshclip-op-d) :<C-u>set operatorfunc=sshclip#emulator#op_delete<Return>g@
    nnoremap <silent> <Plug>(sshclip-op-dd) :<C-u>set operatorfunc=sshclip#emulator#op_delete<Return>g@g@
    nnoremap <silent> <Plug>(sshclip-op-D) :<C-u>set operatorfunc=sshclip#emulator#op_delete<Return>g@$

    for c in range(48, 57) + range(97, 122)
        call add(s:registers, nr2char(c))
    endfor

    for m in s:maps
        let m_type = m[0]

        for k in m[1]
            for r in s:registers
                if index(s:ro_registers, r) != -1 && (m_type == 'yank' || m_type == 'delete')
                    continue
                endif

                execute 'nnoremap <silent> <Plug>(sshclip-' . r . '-' . k . ') :<C-u>call sshclip#emulator#handle(''' . m_type . ''', ''' . r . ''', ''' . k . ''', '''')<Return>'
                if s:can_vmap(k)
                    execute 'vnoremap <silent> <Plug>(sshclip-' . r . '-' . k . ') :<C-u>call sshclip#emulator#handle(''' . m_type . ''', ''' . r. ''', ''' . k . ''', visualmode())<Return>'
                endif
            endfor
        endfor
    endfor
endfunction

function! sshclip#keys#setup_keymap()
    for m in s:maps
        let m_type = m[0]

        for k in m[1]
            for r in s:registers
                if index(s:ro_registers, r) != -1 && (m_type == 'yank' || m_type == 'delete')
                    continue
                endif

                if r == '*'
                    execute 'silent! nmap ' . k . ' <Plug>(sshclip-' . r . '-' . k . ')'

                    if s:can_vmap(k)
                        execute 'silent! vmap ' . k . ' <Plug>(sshclip-' . r . '-' . k . ')'
                    endif
                endif

                execute 'silent! nmap "' . r . k . ' <Plug>(sshclip-' . r . '-' . k . ')'
                if s:can_vmap(k)
                    execute 'silent! vmap "' . r . k . ' <Plug>(sshclip-' . r . '-' . k . ')'
                endif
            endfor
        endfor
    endfor


    silent! nmap y <Plug>(sshclip-op-y)
    silent! nmap d <Plug>(sshclip-op-d)
    silent! nmap dd <Plug>(sshclip-op-dd)
    silent! nmap D <Plug>(sshclip-op-D)
endfunction
