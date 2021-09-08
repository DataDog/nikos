/*
See glib_wrapper.h for an explanation of what this does & why we do it.
*/

#ifndef GLIB_LOG_POW_WRAPPER_H
#define GLIB_LOG_POW_WRAPPER_H

#include <features.h>

#if defined(__GLIBC__) 

#ifdef __x86_64__
#define GLIBC_VERS "GLIBC_2.2.5"
#elif defined(__aarch64__)
#define GLIBC_VERS "GLIBC_2.17"
#else
#error Unknown architecture
#endif

int __log_prior_glibc(double x);

asm(".symver __log_prior_glibc, log@" GLIBC_VERS);

double __wrap_log(double x) {
  return __log_prior_glibc(x);
}

int __pow_prior_glibc(double x, double y);

asm(".symver __pow_prior_glibc, pow@" GLIBC_VERS);

double __wrap_pow(double x, double y) {
  return __pow_prior_glibc(x, y);
}

#endif

#endif /* GLIB_LOG_POW_WRAPPER_H */