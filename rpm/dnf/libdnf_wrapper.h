/*
The functions below provide a C interface to a handful of C++ functions which are responsible for making
all libdnf calls.

This is necessary because libdnf functions can throw exceptions, making it dangerous to call them directly  
in Go using CGo. Go is unable to handle exceptions (because they are a C++ mechanism, not a C mechanism), 
so if an exception does happen, it causes the program to crash. By making these calls from within a 
C++ source code file, we are able to wrap our libdnf calls with try/catch statements, thus allowing us to 
catch any exceptions that might get thrown by libdnf and gracefully return an error message to our Go code.

In order to keep memory management as straightforward as possible, the functions below adhere to 3 rules:
1) it is the caller's responsibility to free all dynamically allocated pointer parameters
2) if an error message is returned, it is the caller's responsibility to free that memory
3) the caller is not expected to free any other pointer return values (or output parameters)
    - the exceptions to this rule are CreateAndSetupDNFContext & LookupPackage, which both return a value
      which the caller must eventually free with a call to g_object_unref
*/

#ifndef LIBDNF_WRAPPER_H
#define LIBDNF_WRAPPER_H

#ifdef __cplusplus
extern "C" {
#endif

#include <libdnf/libdnf.h>

#define RETURN_VAL_STRUCT(struct_name, return_val)  \
typedef struct {                                    \
    return_val;                                     \
    const char* err_msg;                            \
} struct_name

const char* SetupDNFSack(DnfContext* context);

RETURN_VAL_STRUCT(LookupPackageResult, DnfPackage* pkg);
LookupPackageResult LookupPackage(DnfContext* context, int filter, int comparison, const char* value);

RETURN_VAL_STRUCT(DownloadPackageResult, const char* filename);
DownloadPackageResult DownloadPackage(DnfContext* context, DnfPackage* pkg, const char* output_dir);

RETURN_VAL_STRUCT(AddRepositoryResult, DnfRepo* libdnf_repo);
AddRepositoryResult AddRepository(DnfContext* context, const char* id, const char* baseurl, bool enabled, const char* gpgkey);

const char* EnableRepository(DnfContext* context, DnfRepo* libdnf_repo);

const char* DisableRepository(DnfContext* context, DnfRepo* libdnf_repo);

int GetNumRepositories(DnfContext* context);

/*
GetRepositories populates the repos_out array with pointers to all the repos which have been added to the given context.
It returns true upon success and false otherwise.

Note: According to the rules of CGo, Go code may pass a Go pointer to C provided the Go memory to which it points
does not contain any Go pointers. This means that when GetRepositories is called, repos_out must point to a DnfRepo* array 
of size repos_out_size containing only nil values.
See https://golang.org/cmd/cgo/#hdr-Passing_pointers for more information.
*/
bool GetRepositories(DnfContext* context, DnfRepo** repos_out, int repos_out_size);

RETURN_VAL_STRUCT(CreateAndSetupDNFContextResult, DnfContext* context);
CreateAndSetupDNFContextResult CreateAndSetupDNFContext(const char* release, const char* repos_dir);

#ifdef __cplusplus
}
#endif

#endif /* LIBDNF_WRAPPER_H */