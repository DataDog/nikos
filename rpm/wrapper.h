/*
Patch libdnf to remove references to glibc symbols which aren't forward-compatible.

This is necessary because our build system compiles libdnf with glibc-2.16, but the 
system-probe is compiled on a system with a more recent glibc version which doesn't
contain the __secure_getenv symbol (in glibc 2.17, the symbol was renamed secure_getenv).
*/

#include <features.h>

#if defined(__GLIBC__) && defined(__GLIBC_PREREQ) && __GLIBC_PREREQ(2, 17)

char * ____secure_getenv_glibc_2_17(char const *name);

asm(".symver ____secure_getenv_glibc_2_17, secure_getenv@GLIBC_2.17");

char * __wrap___secure_getenv (char const *name) {
  return ____secure_getenv_glibc_2_17(name);
}

#endif
