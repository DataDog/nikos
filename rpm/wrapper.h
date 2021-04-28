/*
Patch libdnf to remove references to glibc symbols which aren't forward-compatible.

This is necessary because our build system compiles libdnf with glibc-2.16, but the 
system-probe is compiled on a system with a more recent glibc version which doesn't
contain the __secure_getenv symbol (in glibc 2.17, the symbol was renamed secure_getenv).
*/

#include <features.h>

#if defined(__GLIBC__) && defined(__GLIBC_PREREQ) && __GLIBC_PREREQ(2, 17)

#define symver_wrap___secure_getenv()                                  \
char * ____secure_getenv_glibc_2_17(char const *name);                 \
                                                                       \
asm(".symver ____secure_getenv_glibc_2_17, secure_getenv@GLIBC_2.17"); \
                                                                       \
char * __wrap___secure_getenv (char const *name) {                     \
  return ____secure_getenv_glibc_2_17(name);                           \
}

# else

// Use the function directly for older glibc / non-glibc environments

#define symver_wrap___secure_getenv()              \
char * __secure_getenv(char const *name);          \
                                                   \
char * __wrap___secure_getenv (char const *name) { \
  return __secure_getenv(name);                    \
}

#endif

symver_wrap___secure_getenv()
