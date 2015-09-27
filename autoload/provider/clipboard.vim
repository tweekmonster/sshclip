function! provider#clipboard#Call(method, args)
    if a:method == 'get'
        return sshclip#register#get(a:args[0])
    elseif a:method == 'set'
        call sshclip#register#put(a:args[2], '', a:args[0], a:args[1])
    endif
endfunction

" vim: set ts=4 sw=4 tw=78 et :
