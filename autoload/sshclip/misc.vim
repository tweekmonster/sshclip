let s:path = expand('<sfile>:p:h')
let s:bin = resolve(printf('%s/../../bin/sshclip-client', s:path))
let s:regstore = {'+': 'clipboard', '*': 'primary'}


function! sshclip#misc#command(meth, register)
    let flag = '-o'
    if a:meth == 'put'
        let flag = '-i'
    endif
    return [s:bin, flag, '-selection', s:regstore[a:register]]
endfunction


function! sshclip#misc#command_str(meth, register)
    return join(sshclip#misc#command(a:meth, a:register), ' ')
endfunction
