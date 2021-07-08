/*
1) Patch libdnf to remove references to glibc symbols which aren't forward-compatible.

This is necessary because our build system compiles libdnf with glibc-2.16, but the 
system-probe is compiled on a system with a more recent glibc version which doesn't
contain the __secure_getenv symbol (in glibc 2.17, the symbol was renamed secure_getenv).

2) Patch nikos to remove references to glibc symbols with a too recent version.

Commands used to find symbols requiring a new version of GLIBC:
// see version requirements of nikos
$ objdump -p nikos
// figure out which functions/symbols need that version
$ nm nikos | grep GLIBC_2.27
*/

#ifndef GLIB_WRAPPER_H
#define GLIB_WRAPPER_H

#include <features.h>
#include <glob.h>

#if defined(__GLIBC__) && defined(__GLIBC_PREREQ) && __GLIBC_PREREQ(2, 17)

char * ____secure_getenv_glibc_2_17(char const *name);

asm(".symver ____secure_getenv_glibc_2_17, secure_getenv@GLIBC_2.17");

char * __wrap___secure_getenv (char const *name) {
  return ____secure_getenv_glibc_2_17(name);
}

#endif

#if defined(__GLIBC__) 

#define GLOB_ARGS const char *pattern, int flags, int (* errfunc)(const char *, int), glob_t *pglob

#ifdef __x86_64__
#define GLIBC_VERS "GLIBC_2.2.5"
#elif defined(__aarch64__)
#define GLIBC_VERS "GLIBC_2.17"
#else
#error Unknown architecture
#endif

int __glob64_prior_glibc(GLOB_ARGS);

asm(".symver __glob64_prior_glibc, glob64@" GLIBC_VERS); 

int __wrap_glob64(GLOB_ARGS) {
  return __glob64_prior_glibc(pattern, flags, errfunc, pglob);
}

int __glob_prior_glibc(GLOB_ARGS);

asm(".symver __glob_prior_glibc, glob@" GLIBC_VERS); 

int __wrap_glob(GLOB_ARGS) {
  return __glob_prior_glibc(pattern, flags, errfunc, pglob);
}

#endif

#endif /* GLIB_WRAPPER_H */