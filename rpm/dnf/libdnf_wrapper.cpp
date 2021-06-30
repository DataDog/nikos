// +build dnf

#include "libdnf_wrapper.h"
#include <exception>
#include <string>
#include <string.h>

#include <libdnf/libdnf.h>

const char* newCString(std::string s) {
    char* msg = new char[s.size()+1];
    strcpy(msg, s.c_str());
    return msg;
}

const char* getErrorMessage(GError* gerr) {
    if (gerr == nullptr)
        return newCString("unknown error");

    const char* msg = strdup(gerr->message);
    g_error_free(gerr);
    return msg;
}

const char* getErrorMessage(std::string prefix, GError* gerr) {
    if (gerr == nullptr)
        return newCString(prefix + "unknown error");

    size_t prefix_size = prefix.size();
    char* msg = new char[prefix_size + strlen(gerr->message) + 1];

    strcpy(msg, prefix.c_str());
    strcpy(msg + prefix_size, gerr->message);

    g_error_free(gerr);
    return msg;
}

SetupDNFSackResult SetupDNFSack(DnfContext* context) {
    SetupDNFSackResult result = {0};
    try {
        result.dnf_state = dnf_state_new();

        GError* gerr = nullptr;
        if (dnf_context_setup_sack(context, result.dnf_state, &gerr) == 0) {
            result.err_msg = getErrorMessage(gerr);
            return result;
        }
    } catch(std::exception &e) {
        result.err_msg = strdup(e.what());
    }

    return result;
}

LookupPackageResult LookupPackage(DnfContext* context, int filter, int comparison, const char* value) {
    LookupPackageResult result = {0};
    HyQuery query;
    try {
        DnfSack* sack = dnf_context_get_sack(context);
        query = hy_query_create(sack);
        hy_query_filter(query, filter, comparison, value);
        GPtrArray* pkglist = hy_query_run(query);

        if (!pkglist || pkglist->len == 0)
            result.err_msg = newCString("failed to find package");
        else
            result.pkg = (DnfPackage*) g_object_ref(g_ptr_array_index(pkglist, 0));
    } catch(std::exception &e) {
        result.err_msg = strdup(e.what());
    }

    if (query != nullptr)
        hy_query_free(query);

    return result;
}

DownloadPackageResult DownloadPackage(DnfContext* context, DnfState* dnf_state, DnfPackage* pkg, const char* output_dir) {
    DownloadPackageResult result = {0};
    try {
        DnfTransaction* transaction = dnf_context_get_transaction(context);

        GError* gerr = nullptr;
        if (dnf_transaction_ensure_repo(transaction, pkg, &gerr) == 0) {
            result.err_msg = getErrorMessage("failed to set package repository: ", gerr);
            return result;
        }

        if (dnf_package_installed(pkg)) {
            result.err_msg = newCString("package already installed");
            return result;
        }

        // C.dnf_state_set_percentage_changed_cb(result.dnf_state)

        g_log(NULL, G_LOG_LEVEL_INFO, "Downloading package");

        // p := mpb.New()
        // bar = p.AddBar(int64(100), mpb.AppendDecorators(decor.Percentage()))

        dnf_package_download(pkg, output_dir, dnf_state, &gerr);
        if (gerr != nullptr) {
            result.err_msg = getErrorMessage("failed to download package: ", gerr);
            return result;
        }

        result.filename = dnf_package_get_filename(pkg);
    } catch(std::exception &e) {
        result.err_msg = strdup(e.what());
    }
    return result;
}

AddRepositoryResult AddRepository(DnfContext* context, const char* id, const char* baseurl, bool enabled, const char* gpgkey) {
    AddRepositoryResult result = {0};
    try {
        DnfRepo* libdnf_repo = dnf_repo_new(context);
        dnf_repo_set_kind(libdnf_repo, DNF_REPO_KIND_REMOTE);

        g_autoptr(GKeyFile) key_file = g_key_file_new();
        g_key_file_set_string(key_file, id, "baseurl", baseurl);

        if (strlen(gpgkey) != 0) {
            dnf_repo_set_gpgcheck(libdnf_repo, gboolean(1));
            g_key_file_set_string(key_file, id, "gpgkey", gpgkey); 
        }

        dnf_repo_set_keyfile(libdnf_repo, key_file);
        dnf_repo_set_id(libdnf_repo, id);
        if (enabled)
            dnf_repo_set_enabled(libdnf_repo, DNF_REPO_ENABLED_PACKAGES);
        else
            dnf_repo_set_enabled(libdnf_repo, DNF_REPO_ENABLED_NONE);

        const char* filename = std::string("/tmp/" + std::string(id) + ".repo").c_str();
        dnf_repo_set_filename(libdnf_repo, filename);

        GError* gerr = nullptr;
        if (dnf_repo_setup(libdnf_repo, &gerr) == 0) {
            result.err_msg = getErrorMessage(gerr);
            return result;
        }

        g_ptr_array_add(dnf_context_get_repos(context), gpointer(libdnf_repo));
        result.libdnf_repo = libdnf_repo;
    } catch(std::exception &e) {
        result.err_msg = strdup(e.what());
    }

    return result;
}

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