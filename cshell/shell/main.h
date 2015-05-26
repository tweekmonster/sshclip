#ifndef MAIN_H_J6BN1YSU
#define MAIN_H_J6BN1YSU

// Error Codes
#define SC_EX_INVOCATION 126
#define SC_EX_CMD 2
#define SC_EX_OTHER 1
#define SC_EX_OK 0

#define IS_CLEAN_CHAR(c) (c == '-' || c == '_' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'))
#define IS_VALID_B64_CHAR(c) (c == '\n' || c == '=' || c == '/' || c == '+' || (c >= '0' && c <= '9') || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z'))

// inline int is_clean_char(char c)
// {
//     return (c == '-' || c == '_'
//             || (c >= '0' && c <= '9')
//             || (c >= 'A' && c <= 'Z')
//             || (c >= 'a' && c <= 'z'));
// }
//
//
// inline int is_valid_base64_char(char c)
// {
//     return (c == '=' || c == '/' || c == '+'
//             || (c >= '0' && c <= '9')
//             || (c >= 'A' && c <= 'Z')
//             || (c >= 'a' && c <= 'z'));
// }
//
inline int is_valid_b64_str(char *str, size_t len)
{
    while (len--) {
        if (!IS_VALID_B64_CHAR(*str)) {
            return 0;
        }
        str++;
    }
    return 1;
}


inline void clean_str(char *str)
{
    unsigned char *p, *s = (void *)str;
    p = s;
    while (*s != '\0') {
        if (IS_CLEAN_CHAR(*s)) {
            *(p++) = *s;
        }
        s++;
    }
    *p = '\0';
}

#endif /* end of include guard: MAIN_H_J6BN1YSU */

/* vim: set ts=4 sw=4 tw=0 et :*/
