function! provider#clipboard#Call(method, args)
    if a:method == 'get'
        " https://github.com/neovim/neovim/blob/acdac914d554fae421c4e71c9d1dffc5cea4505b/src/nvim/ops.c#L5333
        " According to ops.c, we can return a 2 item list with the last item
        " being the register type. The register type *must* be one character
        " long. sshclip doesn't store block types without a width so that it
        " can still work in regular Vim.
        let data = sshclip#register#get(a:args[0])
        return [data[0], data[1][0]]
    elseif a:method == 'set'
        call sshclip#register#put(a:args[2], '', a:args[0], getregtype('"'))
    endif
endfunction

" vim: set ts=4 sw=4 tw=78 et :
