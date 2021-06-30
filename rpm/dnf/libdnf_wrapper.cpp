// +build dnf

#include "libdnf_wrapper.h"
#include <exception>
#include <string>
#include <string.h>

#include <libdnf/libdnf.h>

CreateAndSetupDNFContextResult CreateAndSetupDNFContext(const char* release, const char* repos_dir) {
    CreateAndSetupDNFContextResult result = {0};
    try {
        DnfContext* context = dnf_context_new();

        const char* tmp_dir = "/tmp";
        const char* solv_dir = "/tmp/nikos-solv";
        const char* cache_dir = "/tmp/nikos-cache";

        DnfLock* lock = dnf_lock_new();
        dnf_lock_set_lock_dir(lock, tmp_dir);

        dnf_context_set_solv_dir(context, solv_dir);
        dnf_context_set_cache_dir(context, cache_dir);
        if (strlen(repos_dir) != 0)
            dnf_context_set_repo_dir(context, repos_dir);

        dnf_context_set_release_ver(context, release);

        const char* actual_solv_dir = dnf_context_get_solv_dir(context);
        if (solv_dir)
            g_log(NULL, G_LOG_LEVEL_INFO, "Solv directory: %s", actual_solv_dir);

        GError* gerr = nullptr;
        if (dnf_context_setup(context, nullptr, &gerr) == 0) {
            result.err_msg = getErrorMessage(gerr);
            return result;
        }

        dnf_context_set_write_history(context, 0);

        g_object_unref(lock);
        result.context = context;
    } catch(std::exception &e) {
        result.err_msg = strdup(e.what());
    }
    return result;
}