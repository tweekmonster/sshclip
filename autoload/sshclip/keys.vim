let s:maps = [
            \ ['yank', ['y', 'yy', 'Y']],
            \ ['delete', ['c', 'cc', 'C', 'x', 'X', 'd', 'dd', 'D']],
            \ ['paste', ['p', 'P', 'gp', 'gP', ']p', ']P', '[p', '[P']],
            \ ]

let s:registers = ['*', '+']
let s:ro_registers = [':', '.', '%', '#']


function! s:can_vmap(k)
    if len(a:k) == 2 && a:k[0] == a:k[1]
        return 0
    endif
    return 1
endfunction


function! sshclip#keys#setup_interface()
    for r in ['*', '+']
        execute 'noremap! <Plug>(sshclip-' . r . '-ins) <c-r>=sshclip#emulator#insert(''' . r . ''')<cr>'
        execute 'noremap! <Plug>(sshclip-' . r . '-ins-lit) <c-r><c-r>=sshclip#emulator#insert(''' . r . ''')<cr>'
        execute 'noremap! <Plug>(sshclip-' . r . '-ins-lit-noai) <c-r><c-o>=sshclip#emulator#insert(''' . r . ''')<cr>'
        execute 'inoremap <Plug>(sshclip-' . r . '-ins-lit-ai) <c-r><c-p>=sshclip#emulator#insert(''' . r . ''')<cr>'
    endfor

    if get(g:, 'sshclip_vim_map_all', 0)
        echomsg "Holy shit"
        " Holy shit
        nnoremap <silent> <Plug>(sshclip-op-y) :<C-u>set operatorfunc=sshclip#emulator#op_yank<Return>g@
        nnoremap <silent> <Plug>(sshclip-op-c) :<C-u>set operatorfunc=sshclip#emulator#op_change<Return>g@
        nnoremap <silent> <Plug>(sshclip-op-cc) :<C-u>set operatorfunc=sshclip#emulator#op_change<Return>g@g@
        nnoremap <silent> <Plug>(sshclip-op-C) :<C-u>set operatorfunc=sshclip#emulator#op_change<Return>g@$
        nnoremap <silent> <Plug>(sshclip-op-d) :<C-u>set operatorfunc=sshclip#emulator#op_delete<Return>g@
        nnoremap <silent> <Plug>(sshclip-op-dd) :<C-u>set operatorfunc=sshclip#emulator#op_delete<Return>g@g@
        nnoremap <silent> <Plug>(sshclip-op-D) :<C-u>set operatorfunc=sshclip#emulator#op_delete<Return>g@$

        " Unfortunately have to map all register shortcuts so that maps without a
        " register prefix can set unnamed to * (e.g. y, d, dd)
        call extend(s:registers, ['"', '-', ':', '.', '%', '#', '=', '-', '_', '/'])

        for c in range(48, 57) + range(97, 122)
            call add(s:registers, nr2char(c))
        endfor
    endif

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
    for r in ['*', '+']
        execute 'silent! map! <c-r>' . r . ' <Plug>(sshclip-' . r . '-ins)'
        execute 'silent! map! <c-r><c-r>' . r . ' <Plug>(sshclip-' . r . '-ins-lit)'
        execute 'silent! map! <c-r><c-o>' . r . ' <Plug>(sshclip-' . r . '-ins-lit-noai)'
        execute 'silent! imap <c-r><c-p>' . r . ' <Plug>(sshclip-' . r . '-ins-lit-ai)'
    endfor

    let map_all = get(g:, 'sshclip_vim_map_all', 0)

    for m in s:maps
        let m_type = m[0]

        for k in m[1]
            for r in s:registers
                if index(s:ro_registers, r) != -1 && (m_type == 'yank' || m_type == 'delete')
                    continue
                endif

                if map_all && r == '*'
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


    if map_all
        silent! nmap c <Plug>(sshclip-op-c)
        silent! nmap cc <Plug>(sshclip-op-cc)
        silent! nmap C <Plug>(sshclip-op-C)
        silent! nmap y <Plug>(sshclip-op-y)
        silent! nmap d <Plug>(sshclip-op-d)
        silent! nmap dd <Plug>(sshclip-op-dd)
        silent! nmap D <Plug>(sshclip-op-D)
    endif
endfunction

"  vim: set ft=vim ts=4 sw=4 tw=78 et :
