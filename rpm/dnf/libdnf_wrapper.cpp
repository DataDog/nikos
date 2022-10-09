// +build dnf

#include "libdnf_wrapper.h"
#include <exception>
#include <string>
#include <string.h>

const char* newCString(std::string s) {
    size_t len = s.size()+1;
    char* msg = (char*) malloc(len);
    strncpy(msg, s.c_str(), len);
    return msg;
}

const char* getErrorMessage(GError* gerr) {
    if (gerr == nullptr) {
        return newCString("unknown error");
    }

    const char* msg = strdup(gerr->message);
    g_error_free(gerr);
    return msg;
}

const char* getErrorMessage(std::string prefix, GError* gerr) {
    if (gerr == nullptr) {
        return newCString(prefix + "unknown error");
    }

    const char* msg = newCString(prefix + std::string(gerr->message));
    g_error_free(gerr);
    return msg;
}

const char* SetupDNFSack(DnfContext* context) {
    try {
        DnfState* state = dnf_state_new();

        GError* gerr = nullptr;
        if (dnf_context_setup_sack(context, state, &gerr) == 0) {
            return getErrorMessage(gerr);
        }
    } catch(std::exception &e) {
        return strdup(e.what());
    }
    return nullptr;
}

LookupPackageResult LookupPackage(DnfContext* context, int filter, int comparison, const char* value) {
    LookupPackageResult result = {0};
    HyQuery query = nullptr;
    GPtrArray* pkglist = nullptr;
    try {
        DnfSack* sack = dnf_context_get_sack(context);
        query = hy_query_create(sack);
        hy_query_filter(query, filter, comparison, value);
        pkglist = hy_query_run(query);

        if (pkglist == nullptr || pkglist->len == 0) {
            result.err_msg = newCString("failed to find package");
        } else {
            result.pkg = static_cast<DnfPackage*>(g_object_ref(g_ptr_array_index(pkglist, 0)));
        }
    } catch(std::exception &e) {
        result.err_msg = strdup(e.what());
    }

    if (query != nullptr) {
        hy_query_free(query);
    }
    if (pkglist != nullptr) {
        g_ptr_array_unref(pkglist);
    }

    return result;
}

DownloadPackageResult DownloadPackage(DnfContext* context, DnfPackage* pkg, const char* output_dir) {
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

        g_log(NULL, G_LOG_LEVEL_INFO, "Downloading package");

        DnfState* state = dnf_context_get_state(context);
        dnf_package_download(pkg, output_dir, state, &gerr);
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

AddRepositoryResult AddRepository(
    DnfContext* context, const char* id, const char* baseurl, bool enabled,
    const char* gpgkey, const char *sslcacert, const char *sslclientcert,
    const char *sslclientkey)
{
    AddRepositoryResult result = {0};
    try {
        DnfRepo* libdnf_repo = dnf_repo_new(context);
        dnf_repo_set_kind(libdnf_repo, DNF_REPO_KIND_REMOTE);

        g_autoptr(GKeyFile) key_file = g_key_file_new();
        g_key_file_set_string(key_file, id, "baseurl", baseurl);

        if (gpgkey && strlen(gpgkey) != 0) {
            dnf_repo_set_gpgcheck(libdnf_repo, gboolean(1));
            g_key_file_set_string(key_file, id, "gpgkey", gpgkey); 
        }

        if (sslcacert && strlen(sslcacert) != 0) {
            g_key_file_set_string(key_file, id, "sslcacert", sslcacert);
        }

        if (sslclientcert && strlen(sslclientcert) != 0) {
            g_key_file_set_string(key_file, id, "sslclientcert", sslclientcert);
        }

        if (sslclientkey && strlen(sslclientkey) != 0) {
            g_key_file_set_string(key_file, id, "sslclientkey", sslclientkey);
        }

        dnf_repo_set_keyfile(libdnf_repo, key_file);
        dnf_repo_set_id(libdnf_repo, id);
        if (enabled) {
            dnf_repo_set_enabled(libdnf_repo, DNF_REPO_ENABLED_PACKAGES);
        } else {
            dnf_repo_set_enabled(libdnf_repo, DNF_REPO_ENABLED_NONE);
        }

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

const char* EnableRepository(DnfContext* context, DnfRepo* libdnf_repo) {
    try {
        GError* gerr = nullptr;
        if (dnf_context_repo_enable(context, dnf_repo_get_id(libdnf_repo), &gerr) == 0) {
            return getErrorMessage(gerr);
        }
    } catch(std::exception &e) {
        return strdup(e.what());
    }
    return nullptr;
}

const char* DisableRepository(DnfContext* context, DnfRepo* libdnf_repo) {
    try {
        GError* gerr = nullptr;
        if (dnf_context_repo_disable(context, dnf_repo_get_id(libdnf_repo), &gerr) == 0) {
            return getErrorMessage(gerr);
        }
    } catch(std::exception &e) {
        return strdup(e.what());
    }
    return nullptr;
}

int GetNumRepositories(DnfContext* context) {
    try {
        GPtrArray* repos = dnf_context_get_repos(context);
        if (repos != nullptr) {
            return repos->len;
        }
    } catch(std::exception &e) {
        g_log(NULL, G_LOG_LEVEL_INFO, "error fetching number of repositories: %s", e.what());
    }
    return 0;
}

bool GetRepositories(DnfContext* context, DnfRepo** repos_out, int repos_out_size) {
    try {
        GPtrArray* repos = dnf_context_get_repos(context);

        if (repos == nullptr || repos->len != repos_out_size) {
            return false;
        }

        for (int i=0; i<repos->len; i++) {
            repos_out[i] = (DnfRepo*) g_ptr_array_index(repos, i);
        }
    } catch(std::exception &e) {
        g_log(NULL, G_LOG_LEVEL_INFO, "error fetching repositories: %s", e.what());
        return false;
    }
    return true;
}

CreateAndSetupDNFContextResult CreateAndSetupDNFContext(const char* release, const char* repos_dir, const char* vars_dir) {
    CreateAndSetupDNFContextResult result = {0};
    try {
        DnfContext* context = dnf_context_new();

        const char* tmp_dir = "/tmp";
        const char* solv_dir = "/tmp/nikos-solv";
        const char* cache_dir = "/tmp/nikos-cache";
        const char* install_root = "/opt/nikos/embedded";

        DnfLock* lock = dnf_lock_new();
        dnf_lock_set_lock_dir(lock, tmp_dir);

        dnf_context_set_solv_dir(context, solv_dir);
        dnf_context_set_cache_dir(context, cache_dir);
        if (strlen(repos_dir) != 0) {
            dnf_context_set_repo_dir(context, repos_dir);
        }
        if (strlen(vars_dir) != 0) {
            dnf_context_set_vars_dir(context, vars_dir);
        }
        dnf_context_set_install_root(context, install_root);

        dnf_context_set_release_ver(context, release);

        const char* actual_solv_dir = dnf_context_get_solv_dir(context);
        if (actual_solv_dir != nullptr) {
            g_log(NULL, G_LOG_LEVEL_INFO, "Solv directory: %s", actual_solv_dir);
        }

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
